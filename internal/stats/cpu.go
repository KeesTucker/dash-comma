package stats

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// procRoot and sysRoot are swapped for fixtures in tests.
var (
	procRoot = "/proc"
	sysRoot  = "/sys"
)

var (
	coreCountRe   = regexp.MustCompile(`[A-Za-z0-9]*-Core`)
	cpuJunkRe     = regexp.MustCompile(`\s*(\(R\)|\(TM\)|®|™|CPU|Processor|@.*$)`)
	spaceRe       = regexp.MustCompile(`\s+`)
	vendorsByID   = map[string]string{"GenuineIntel": "Intel", "AuthenticAMD": "AMD", "HygonGenuine": "Hygon"}
	armImplements = map[string]string{"0x41": "ARM", "0x42": "Broadcom", "0x43": "Cavium", "0x48": "HiSilicon", "0x4e": "NVIDIA", "0x51": "Qualcomm", "0x61": "Apple", "0xc0": "Ampere"}
)

// cpuModel turns "Intel(R) Core(TM) i7-8665U CPU @ 1.90GHz" into
// "Core i7-8665U", and "AMD Ryzen 9 5950X 16-Core Processor" into
// "Ryzen 9 5950X".
func cpuModel(name, brand string) string {
	s := cpuJunkRe.ReplaceAllString(name, " ")
	s = coreCountRe.ReplaceAllString(s, "")
	s = spaceRe.ReplaceAllString(s, " ")
	s = strings.TrimSpace(s)
	if brand != "" {
		s = strings.TrimSpace(strings.TrimPrefix(s, brand))
	}
	return s
}

type cpuTopology struct {
	cpu     int
	pkg     int
	core    int
	isAtom  bool
	present bool
}

func readTopology() []cpuTopology {
	dirs, _ := filepath.Glob(filepath.Join(sysRoot, "devices/system/cpu/cpu[0-9]*"))
	atoms := map[int]bool{}
	for _, c := range parseCPUList(readTrim(filepath.Join(sysRoot, "devices/cpu_atom/cpus"))) {
		atoms[c] = true
	}
	var out []cpuTopology
	for _, d := range dirs {
		n, err := strconv.Atoi(strings.TrimPrefix(filepath.Base(d), "cpu"))
		if err != nil {
			continue
		}
		pkg, ok1 := readInt(filepath.Join(d, "topology/physical_package_id"))
		core, ok2 := readInt(filepath.Join(d, "topology/core_id"))
		out = append(out, cpuTopology{cpu: n, pkg: int(pkg), core: int(core), isAtom: atoms[n], present: ok1 && ok2})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].cpu < out[j].cpu })
	return out
}

func CPUStatic() (CPUInfo, error) {
	f, err := os.Open(filepath.Join(procRoot, "cpuinfo"))
	if err != nil {
		return CPUInfo{}, err
	}
	defer f.Close()

	fields := map[string]string{}
	threads := 0
	var maxMHz float64
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		k, v, ok := strings.Cut(sc.Text(), ":")
		if !ok {
			continue
		}
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		if k == "processor" {
			threads++
		}
		if k == "cpu MHz" {
			if mhz, err := strconv.ParseFloat(v, 64); err == nil && mhz > maxMHz {
				maxMHz = mhz
			}
		}
		if _, seen := fields[k]; !seen {
			fields[k] = v
		}
	}

	info := CPUInfo{Threads: threads}
	info.Brand = vendorsByID[fields["vendor_id"]]
	if info.Brand == "" {
		info.Brand = armImplements[strings.ToLower(fields["CPU implementer"])]
	}
	name := fields["model name"]
	if name == "" {
		name = fields["Model"]
	}
	if info.Brand == "" {
		info.Brand, _, _ = strings.Cut(name, " ")
	}
	info.Model = cpuModel(name, info.Brand)

	topo := readTopology()
	if len(topo) > 0 {
		info.Threads = len(topo)
	}
	pcores, ecores := map[[2]int]bool{}, map[[2]int]bool{}
	for _, t := range topo {
		key := [2]int{t.pkg, t.core}
		if t.isAtom {
			ecores[key] = true
		} else {
			pcores[key] = true
		}
	}
	info.Cores = len(pcores) + len(ecores)
	if info.Cores == 0 {
		info.Cores = info.Threads
	}
	// Not hybrid: every core is a P-core, as systeminformation reports it.
	info.Pcores, info.Ecores = len(pcores), len(ecores)
	if info.Pcores == 0 {
		info.Pcores = info.Cores - info.Ecores
	}

	if khz, ok := readInt(filepath.Join(sysRoot, "devices/system/cpu/cpu0/cpufreq/cpuinfo_max_freq")); ok && khz > 0 {
		info.Frequency = round(float64(khz)/1e6, 2)
	} else if maxMHz > 0 {
		info.Frequency = round(maxMHz/1000, 2)
	}
	return info, nil
}

type cpuTimes struct{ busy, total uint64 }

// CPUSampler works out per-thread load from /proc/stat deltas between calls.
type CPUSampler struct {
	mu    sync.Mutex
	last  map[int]cpuTimes
	temps bool
	topo  []cpuTopology
}

func NewCPUSampler(temps bool) *CPUSampler {
	return &CPUSampler{temps: temps, topo: readTopology()}
}

func readProcStat() (map[int]cpuTimes, error) {
	f, err := os.Open(filepath.Join(procRoot, "stat"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	out := map[int]cpuTimes{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		parts := strings.Fields(sc.Text())
		if len(parts) < 5 || !strings.HasPrefix(parts[0], "cpu") || parts[0] == "cpu" {
			continue
		}
		n, err := strconv.Atoi(parts[0][3:])
		if err != nil {
			continue
		}
		var t cpuTimes
		for i, v := range parts[1:] {
			// guest and guest_nice are already counted in user and nice.
			if i >= 8 {
				break
			}
			x, _ := strconv.ParseUint(v, 10, 64)
			t.total += x
			// idle and iowait.
			if i != 3 && i != 4 {
				t.busy += x
			}
		}
		out[n] = t
	}
	return out, sc.Err()
}

func (s *CPUSampler) Sample() (CPULoad, error) {
	now, err := readProcStat()
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	prev := s.last
	s.last = now
	s.mu.Unlock()

	var temps map[int]float64
	var main float64
	if s.temps {
		temps, main = cpuTemps(s.topo)
	}

	cpus := make([]int, 0, len(now))
	for n := range now {
		cpus = append(cpus, n)
	}
	sort.Ints(cpus)
	out := make(CPULoad, 0, len(cpus))
	for i, n := range cpus {
		load := 0.0
		// Counters restart when a CPU comes back online; skip that reading
		// rather than let the unsigned difference wrap.
		if p, ok := prev[n]; ok && now[n].total > p.total && now[n].busy >= p.busy {
			load = float64(now[n].busy-p.busy) / float64(now[n].total-p.total) * 100
		}
		temp, ok := temps[n]
		if !ok {
			temp = main
		}
		out = append(out, CoreLoad{Core: i, Load: round(load, 2), Temp: temp})
	}
	return out, nil
}

// cpuTemps reads hwmon: a temperature per logical CPU where the driver has
// per-core sensors (coretemp), and a main one for the rest.
func cpuTemps(topo []cpuTopology) (map[int]float64, float64) {
	hwmons, _ := filepath.Glob(filepath.Join(sysRoot, "class/hwmon/hwmon*"))
	perCore := map[[2]int]float64{}
	var main float64
	for _, h := range hwmons {
		name := readTrim(filepath.Join(h, "name"))
		inputs, _ := filepath.Glob(filepath.Join(h, "temp*_input"))
		switch name {
		case "coretemp":
			pkg := -1
			cores := map[int]float64{}
			for _, in := range inputs {
				label := readTrim(strings.TrimSuffix(in, "_input") + "_label")
				v, ok := readInt(in)
				if !ok {
					continue
				}
				t := float64(v) / 1000
				var n int
				if _, err := fmt.Sscanf(label, "Package id %d", &n); err == nil {
					pkg = n
					if main == 0 {
						main = t
					}
				} else if _, err := fmt.Sscanf(label, "Core %d", &n); err == nil {
					cores[n] = t
				}
			}
			if pkg < 0 {
				pkg = 0
			}
			for c, t := range cores {
				perCore[[2]int{pkg, c}] = t
			}
		case "k10temp", "zenpower":
			for _, in := range inputs {
				label := readTrim(strings.TrimSuffix(in, "_input") + "_label")
				if v, ok := readInt(in); ok && (label == "Tctl" || label == "Tdie") {
					main = float64(v) / 1000
				}
			}
		case "cpu_thermal", "soc_thermal", "cpu-thermal":
			if v, ok := readInt(filepath.Join(h, "temp1_input")); ok && main == 0 {
				main = float64(v) / 1000
			}
		}
	}
	if main == 0 && len(perCore) > 0 {
		var sum float64
		for _, t := range perCore {
			sum += t
		}
		main = sum / float64(len(perCore))
	}
	if main == 0 {
		zones, _ := filepath.Glob(filepath.Join(sysRoot, "class/thermal/thermal_zone*"))
		for _, z := range zones {
			if readTrim(filepath.Join(z, "type")) == "x86_pkg_temp" {
				if v, ok := readInt(filepath.Join(z, "temp")); ok {
					main = float64(v) / 1000
					break
				}
			}
		}
	}

	out := map[int]float64{}
	for _, t := range topo {
		if v, ok := perCore[[2]int{t.pkg, t.core}]; ok && t.present {
			out[t.cpu] = v
		}
	}
	return out, main
}
