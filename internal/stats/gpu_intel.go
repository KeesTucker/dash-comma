package stats

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"unsafe"

	"golang.org/x/sys/unix"
)

// Engine class names as intel_gpu_top prints them, which dashdot passed on.
var (
	i915Classes = map[string]string{"rcs": "Render/3D", "bcs": "Blitter", "vcs": "Video", "vecs": "VideoEnhance", "ccs": "Compute"}
	xeClasses   = []string{"Render/3D", "Blitter", "Video", "VideoEnhance", "Compute"}
	i915BusyRe  = regexp.MustCompile(`^(rcs|bcs|vcs|vecs|ccs)\d+-busy$`)
)

// counter is one perf counter, read with its enabled time.
type counter struct {
	fd int
}

func (c *counter) read() (value, enabled uint64, err error) {
	var buf [16]byte
	if _, err := unix.Read(c.fd, buf[:]); err != nil {
		return 0, 0, err
	}
	return binary.LittleEndian.Uint64(buf[:8]), binary.LittleEndian.Uint64(buf[8:]), nil
}

// intelEngine is busy time over elapsed time. i915 counts busy nanoseconds,
// measured against the counter's enabled time; xe counts active ticks
// against a second counter of total ticks.
type intelEngine struct {
	class string
	busy  *counter
	total *counter

	primed              bool
	lastBusy, lastTotal uint64
}

func (e *intelEngine) sample() float64 {
	busy, total, err := e.busy.read()
	if err != nil {
		return 0
	}
	if e.total != nil {
		if total, _, err = e.total.read(); err != nil {
			return 0
		}
	}
	db, dt := busy-e.lastBusy, total-e.lastTotal
	wrapped := busy < e.lastBusy || total < e.lastTotal
	primed := e.primed
	e.primed, e.lastBusy, e.lastTotal = true, busy, total
	if !primed || wrapped || dt == 0 {
		return 0
	}
	return min(100, float64(db)/float64(dt)*100)
}

type intelSampler struct {
	mu      sync.Mutex
	engines []*intelEngine
}

func newIntel(slot string) gpuSampler {
	s := &intelSampler{}
	pmuSlot := strings.ReplaceAll(slot, ":", "_")
	open := openXe
	dir := pmuDir("xe_" + pmuSlot)
	if dir == "" {
		open = openI915
		// An integrated GPU's i915 PMU is just "i915".
		dir = pmuDir("i915_" + pmuSlot)
		if dir == "" && slot == "0000:00:02.0" {
			dir = pmuDir("i915")
		}
	}
	if dir == "" {
		log.Printf("No i915 or xe PMU for Intel GPU %s; it shows no load", slot)
		return s
	}
	var err error
	withPerfmon(func() { s.engines, err = open(dir) })
	if errors.Is(err, unix.EACCES) || errors.Is(err, unix.EPERM) {
		log.Printf("Can't read Intel GPU %s's engines: dash-comma needs CAP_PERFMON (cap_add: [PERFMON])", slot)
	} else if err != nil {
		log.Printf("Can't read Intel GPU %s's engines: %v", slot, err)
	}
	return s
}

func pmuDir(name string) string {
	d := filepath.Join(sysRoot, "bus/event_source/devices", name)
	if !exists(filepath.Join(d, "type")) {
		return ""
	}
	return d
}

func openI915(dir string) ([]*intelEngine, error) {
	events, _ := os.ReadDir(filepath.Join(dir, "events"))
	var out []*intelEngine
	for _, ev := range events {
		m := i915BusyRe.FindStringSubmatch(ev.Name())
		if m == nil {
			continue
		}
		c, err := openEvent(dir, ev.Name(), nil)
		if err != nil {
			return out, err
		}
		out = append(out, &intelEngine{class: i915Classes[m[1]], busy: c})
	}
	return out, nil
}

// openXe probes every GT, class and instance, as xe has no list of engines
// in sysfs: opening a counter for one that doesn't exist fails.
func openXe(dir string) ([]*intelEngine, error) {
	var out []*intelEngine
	for gt := range uint64(4) {
		for class := range uint64(len(xeClasses)) {
			for inst := range uint64(16) {
				terms := map[string]uint64{"gt": gt, "engine_class": class, "engine_instance": inst}
				active, err := openEvent(dir, "engine-active-ticks", terms)
				if errors.Is(err, unix.EACCES) || errors.Is(err, unix.EPERM) {
					return out, err
				}
				if err != nil {
					break // no such engine: on to the next class
				}
				total, err := openEvent(dir, "engine-total-ticks", terms)
				if err != nil {
					_ = unix.Close(active.fd)
					return out, err
				}
				out = append(out, &intelEngine{class: xeClasses[class], busy: active, total: total})
			}
		}
	}
	return out, nil
}

func openEvent(dir, event string, extra map[string]uint64) (*counter, error) {
	pmuType, err := strconv.ParseUint(readTrim(filepath.Join(dir, "type")), 10, 32)
	if err != nil {
		return nil, fmt.Errorf("no PMU type in %s: %w", dir, err)
	}
	config, err := eventConfig(dir, event, extra)
	if err != nil {
		return nil, err
	}
	cpu := 0
	if cpus := parseCPUList(readTrim(filepath.Join(dir, "cpumask"))); len(cpus) > 0 {
		cpu = cpus[0]
	}
	attr := unix.PerfEventAttr{
		Type:        uint32(pmuType),
		Config:      config,
		Read_format: unix.PERF_FORMAT_TOTAL_TIME_ENABLED,
	}
	attr.Size = uint32(unsafe.Sizeof(attr))
	fd, err := unix.PerfEventOpen(&attr, -1, cpu, -1, unix.PERF_FLAG_FD_CLOEXEC)
	if err != nil {
		return nil, err
	}
	return &counter{fd: fd}, nil
}

// eventConfig builds perf_event_attr.config from an event file
// ("event=0x02") and extra terms, placed by the PMU's format files
// ("config:20-27").
func eventConfig(dir, event string, extra map[string]uint64) (uint64, error) {
	terms := map[string]uint64{}
	for _, t := range strings.Split(readTrim(filepath.Join(dir, "events", event)), ",") {
		k, v, ok := strings.Cut(t, "=")
		if !ok {
			continue
		}
		n, err := strconv.ParseUint(v, 0, 64)
		if err != nil {
			return 0, fmt.Errorf("event %s: %w", event, err)
		}
		terms[k] = n
	}
	for k, v := range extra {
		terms[k] = v
	}
	var config uint64
	for k, v := range terms {
		if k == "config" {
			config |= v
			continue
		}
		field, bits, ok := strings.Cut(readTrim(filepath.Join(dir, "format", k)), ":")
		if !ok || field != "config" {
			return 0, fmt.Errorf("event %s: no config format for %q", event, k)
		}
		lo, _, _ := strings.Cut(bits, "-")
		shift, err := strconv.Atoi(lo)
		if err != nil {
			return 0, fmt.Errorf("event %s: format %q: %w", event, bits, err)
		}
		config |= v << shift
	}
	return config, nil
}

// sample averages each class's instances, as intel_gpu_top's class view
// does, and the load is the busiest class.
func (s *intelSampler) sample() GPUUsage {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.engines) == 0 {
		return GPUUsage{}
	}
	sums, counts := map[string]float64{}, map[string]int{}
	for _, e := range s.engines {
		sums[e.class] += e.sample()
		counts[e.class]++
	}
	u := GPUUsage{Engines: map[string]float64{}}
	for class, sum := range sums {
		busy := round(sum/float64(counts[class]), 2)
		u.Engines[class] = busy
		u.Load = max(u.Load, busy)
	}
	return u
}
