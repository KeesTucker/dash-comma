package stats

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// topProcesses is how many processes each list keeps.
const topProcesses = 5

// pidRoot is the /proc that lists the host's processes. In Docker the
// container's own /proc holds only dash-comma, so it reads the host's
// through the /mnt/host mount.
func pidRoot() string {
	if HostRoot != "" {
		if p := filepath.Join(HostRoot, "proc"); exists(filepath.Join(p, "1/stat")) {
			return p
		}
	}
	return procRoot
}

func listPIDs(root string) []int {
	entries, _ := os.ReadDir(root)
	out := make([]int, 0, len(entries))
	for _, e := range entries {
		if pid, err := strconv.Atoi(e.Name()); err == nil {
			out = append(out, pid)
		}
	}
	return out
}

func procName(root string, pid int) string {
	return readTrim(filepath.Join(root, strconv.Itoa(pid), "comm"))
}

// top sorts by load and keeps the busiest, dropping idle ones.
func top(all []ProcessLoad) []ProcessLoad {
	out := make([]ProcessLoad, 0, len(all))
	for _, p := range all {
		if p.Load > 0 {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Load != out[j].Load {
			return out[i].Load > out[j].Load
		}
		return out[i].PID < out[j].PID
	})
	return out[:min(len(out), topProcesses)]
}

// ProcessSampler finds the processes using the most CPU, from the deltas of
// each one's user and system time between calls.
type ProcessSampler struct {
	mu        sync.Mutex
	root      string
	last      map[int]uint64
	lastTotal uint64
}

func NewProcessSampler() *ProcessSampler {
	return &ProcessSampler{root: pidRoot()}
}

// totalJiffies is every CPU's time from /proc/stat. Process times are in the
// same clock ticks, so a share of it is a share of the whole machine.
func totalJiffies(root string) (uint64, error) {
	f, err := os.Open(filepath.Join(root, "stat"))
	if err != nil {
		return 0, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		parts := strings.Fields(sc.Text())
		if len(parts) < 5 || parts[0] != "cpu" {
			continue
		}
		var total uint64
		// guest and guest_nice are already counted in user and nice.
		for _, v := range parts[1:min(len(parts), 9)] {
			x, _ := strconv.ParseUint(v, 10, 64)
			total += x
		}
		return total, nil
	}
	return 0, sc.Err()
}

// parsePIDStat reads the name and utime+stime from /proc/<pid>/stat. The
// name sits in brackets and can hold spaces and brackets itself, so fields
// count from the last ')'.
func parsePIDStat(b []byte) (name string, ticks uint64, ok bool) {
	open, end := bytes.IndexByte(b, '('), bytes.LastIndexByte(b, ')')
	if open < 0 || end < open {
		return "", 0, false
	}
	fields := strings.Fields(string(b[end+1:]))
	// state is field 3 of the whole line; utime and stime are 14 and 15.
	if len(fields) < 13 {
		return "", 0, false
	}
	utime, err1 := strconv.ParseUint(fields[11], 10, 64)
	stime, err2 := strconv.ParseUint(fields[12], 10, 64)
	if err1 != nil || err2 != nil {
		return "", 0, false
	}
	return string(b[open+1 : end]), utime + stime, true
}

func (s *ProcessSampler) Sample() ([]ProcessLoad, error) {
	total, err := totalJiffies(s.root)
	if err != nil {
		return nil, err
	}
	now := map[int]uint64{}
	var all []ProcessLoad
	s.mu.Lock()
	defer s.mu.Unlock()
	dt := total - s.lastTotal
	for _, pid := range listPIDs(s.root) {
		b, err := os.ReadFile(filepath.Join(s.root, strconv.Itoa(pid), "stat"))
		if err != nil {
			continue
		}
		name, ticks, ok := parsePIDStat(b)
		if !ok {
			continue
		}
		now[pid] = ticks
		// A reused PID can start below the old one's time; skip it once.
		if prev, seen := s.last[pid]; seen && s.lastTotal > 0 && total > s.lastTotal && ticks >= prev {
			all = append(all, ProcessLoad{PID: pid, Name: name, Load: round(float64(ticks-prev)/float64(dt)*100, 2)})
		}
	}
	s.last, s.lastTotal = now, total
	return top(all), nil
}

// drmSampler reads each process's GPU time from its open DRM files, as the
// kernel's drm-usage-stats lists them in fdinfo. i915 and amdgpu count busy
// nanoseconds per engine; xe counts busy cycles against total cycles.
// Another user's fdinfo needs CAP_SYS_PTRACE, so without it only
// dash-comma's own user's processes show.
type drmSampler struct {
	mu     sync.Mutex
	root   string
	now    func() time.Time
	last   map[drmKey]drmClient
	lastAt time.Time
	// plain is each process's fds that aren't DRM files, so a scan skips
	// their fdinfo. It empties every drmRecheck scans, in case a process
	// reuses one of those fd numbers for a render node.
	plain map[int]map[string]bool
	scans int
}

// drmRecheck is how many scans the plain fds are trusted for: 30s at the
// default 1s poll.
const drmRecheck = 30

// drmKey is one client: a DRM file, which a process can share with its
// children or hold through several fds.
type drmKey struct {
	pdev string
	id   string
}

type drmClient struct {
	pid         int
	busy        map[string]uint64 // ns, or cycles on xe
	totalCycles map[string]uint64
	capacity    map[string]uint64
}

func newDRMSampler() *drmSampler {
	return &drmSampler{root: pidRoot(), now: time.Now}
}

func parseFdinfo(b []byte) (drmKey, drmClient, bool) {
	var k drmKey
	c := drmClient{busy: map[string]uint64{}, totalCycles: map[string]uint64{}, capacity: map[string]uint64{}}
	for line := range strings.Lines(string(b)) {
		key, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		v = strings.TrimSpace(v)
		n, _ := strconv.ParseUint(strings.TrimSuffix(v, " ns"), 10, 64)
		switch {
		case key == "drm-pdev":
			k.pdev = v
		case key == "drm-client-id":
			k.id = v
		case strings.HasPrefix(key, "drm-engine-capacity-"):
			c.capacity[strings.TrimPrefix(key, "drm-engine-capacity-")] = n
		case strings.HasPrefix(key, "drm-engine-"):
			c.busy[strings.TrimPrefix(key, "drm-engine-")] = n
		case strings.HasPrefix(key, "drm-total-cycles-"):
			c.totalCycles[strings.TrimPrefix(key, "drm-total-cycles-")] = n
		case strings.HasPrefix(key, "drm-cycles-"):
			c.busy[strings.TrimPrefix(key, "drm-cycles-")] = n
		}
	}
	return k, c, k.pdev != "" && k.id != ""
}

// scan reads fdinfo without looking at fd first: another user's fd dir is
// 0500, which CAP_SYS_PTRACE doesn't get past, but fdinfo only checks ptrace.
func (d *drmSampler) scan() map[drmKey]drmClient {
	if d.scans++; d.scans%drmRecheck == 0 {
		d.plain = nil
	}
	plain := map[int]map[string]bool{}
	out := map[drmKey]drmClient{}
	withPtrace(func() {
		for _, pid := range listPIDs(d.root) {
			dir := filepath.Join(d.root, strconv.Itoa(pid), "fdinfo")
			fds, err := os.ReadDir(dir)
			if err != nil {
				continue
			}
			known, skip := d.plain[pid], map[string]bool{}
			for _, fd := range fds {
				if known[fd.Name()] {
					skip[fd.Name()] = true
					continue
				}
				b, err := os.ReadFile(filepath.Join(dir, fd.Name()))
				if err != nil {
					continue
				}
				k, c, ok := parseFdinfo(b)
				if !ok {
					skip[fd.Name()] = true
					continue
				}
				if _, seen := out[k]; seen {
					continue
				}
				c.pid = pid
				out[k] = c
			}
			if len(skip) > 0 {
				plain[pid] = skip
			}
		}
	})
	d.plain = plain
	return out
}

// sample is each GPU's busiest processes, by PCI slot. A process's load is
// its busiest engine, summed over its clients.
func (d *drmSampler) sample() map[string][]ProcessLoad {
	d.mu.Lock()
	now := d.scan()
	at := d.now()
	prev, wall := d.last, at.Sub(d.lastAt).Nanoseconds()
	d.last, d.lastAt = now, at
	d.mu.Unlock()
	if wall <= 0 {
		return nil
	}

	type procKey struct {
		pdev string
		pid  int
	}
	engines := map[procKey]map[string]float64{}
	for k, c := range now {
		p, ok := prev[k]
		if !ok {
			continue
		}
		pk := procKey{k.pdev, c.pid}
		if engines[pk] == nil {
			engines[pk] = map[string]float64{}
		}
		for e, busy := range c.busy {
			last, ok := p.busy[e]
			if !ok || busy < last {
				continue
			}
			var load float64
			if total, ok := c.totalCycles[e]; ok {
				if lastTotal, ok := p.totalCycles[e]; ok && total > lastTotal {
					load = float64(busy-last) / float64(total-lastTotal) * 100
				}
			} else {
				load = float64(busy-last) / float64(wall) * 100
				if n := c.capacity[e]; n > 1 {
					load /= float64(n)
				}
			}
			engines[pk][e] += load
		}
	}

	byGPU := map[string][]ProcessLoad{}
	for pk, es := range engines {
		var load float64
		for _, l := range es {
			load = max(load, l)
		}
		byGPU[pk.pdev] = append(byGPU[pk.pdev], ProcessLoad{PID: pk.pid, Name: procName(d.root, pk.pid), Load: round(min(100, load), 2)})
	}
	for slot, ps := range byGPU {
		byGPU[slot] = top(ps)
	}
	return byGPU
}
