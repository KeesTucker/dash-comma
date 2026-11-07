package stats

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func pidStat(name string, utime, stime, rss int) string {
	return fmt.Sprintf("42 (%s) S 1 42 42 0 -1 4194304 100 0 0 0 %d %d 0 0 20 0 1 0 100 0 %d 0 0\n", name, utime, stime, rss)
}

func TestParsePIDStat(t *testing.T) {
	name, ticks, rss, ok := parsePIDStat([]byte(pidStat("Web Content (x)", 30, 12, 9)))
	if !ok || name != "Web Content (x)" || ticks != 42 || rss != 9 {
		t.Fatalf("got %q, %d, %d, %v", name, ticks, rss, ok)
	}
	if _, _, _, ok := parsePIDStat([]byte("42 (cut")); ok {
		t.Fatal("want a truncated line rejected")
	}
}

// TestProcessMemory shares each process's RSS out of MemTotal, from the
// same scan the CPU list just made.
func TestProcessMemory(t *testing.T) {
	root := useFixtureHost(t, nil)
	host := filepath.Join(root, "mnt/host/proc")
	page := os.Getpagesize()
	fixture(t, root, map[string]string{"proc/meminfo": fmt.Sprintf("MemTotal: %d kB\n", 1000*page/1024)})
	fixture(t, host, map[string]string{
		"stat":   "cpu  100 0 0 0\n",
		"1/stat": pidStat("init", 0, 0, 10),
		"7/stat": pidStat("plex", 0, 0, 250),
		"8/stat": pidStat("kthreadd", 0, 0, 0),
	})
	s := NewProcessSampler()
	if _, err := s.Sample(); err != nil {
		t.Fatal(err)
	}
	// Changed after the CPU scan, so a fresh walk would see it.
	fixture(t, host, map[string]string{"7/stat": pidStat("plex", 0, 0, 999)})
	got, err := s.Memory()
	if err != nil {
		t.Fatal(err)
	}
	want := []ProcessLoad{
		{PID: 7, Name: "plex", Load: 25, Bytes: uint64(250 * page)},
		{PID: 1, Name: "init", Load: 1, Bytes: uint64(10 * page)},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

// TestProcessSampler reads the host's processes through /mnt/host/proc, as
// in Docker, and shares each one's ticks out of every CPU's.
func TestProcessSampler(t *testing.T) {
	root := useFixtureHost(t, nil)
	host := filepath.Join(root, "mnt/host/proc")
	write := func(total, a, b int) {
		fixture(t, host, map[string]string{
			"stat":   fmt.Sprintf("cpu  %d 0 0 0 0 0 0 0 0 0\ncpu0 1 0 0 0\n", total),
			"1/stat": pidStat("init", a, 0, 0),
			"7/stat": pidStat("ffmpeg", b, b, 0),
		})
	}
	fixture(t, root, map[string]string{"proc/stat": "cpu  0 0 0 0\n"})

	write(1000, 10, 100)
	s := NewProcessSampler()
	if got, err := s.Sample(); err != nil || len(got) != 0 {
		t.Fatalf("first sample: got %v, %v", got, err)
	}
	write(1400, 14, 150)
	got, err := s.Sample()
	if err != nil {
		t.Fatal(err)
	}
	want := []ProcessLoad{{PID: 7, Name: "ffmpeg", Load: 25}, {PID: 1, Name: "init", Load: 1}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestTopKeepsBusiest(t *testing.T) {
	var all []ProcessLoad
	for i := range 8 {
		all = append(all, ProcessLoad{PID: i, Load: float64(i)})
	}
	got := top(all)
	if len(got) != topProcesses || got[0].PID != 7 || got[4].PID != 3 {
		t.Fatalf("got %+v", got)
	}
}

func TestGPUTempInput(t *testing.T) {
	dir := t.TempDir()
	fixture(t, dir, map[string]string{
		"amd/hwmon/hwmon2/temp1_input": "50000", "amd/hwmon/hwmon2/temp1_label": "junction",
		"amd/hwmon/hwmon2/temp2_input": "45000", "amd/hwmon/hwmon2/temp2_label": "edge",
		"i915/hwmon/hwmon5/temp1_input": "58000",
	})
	if got := gpuTempInput(filepath.Join(dir, "amd")); got != filepath.Join(dir, "amd/hwmon/hwmon2/temp2_input") {
		t.Errorf("amd: got %q", got)
	}
	if got := gpuTempInput(filepath.Join(dir, "i915")); got != filepath.Join(dir, "i915/hwmon/hwmon5/temp1_input") {
		t.Errorf("i915: got %q", got)
	}
	if got := gpuTempInput(filepath.Join(dir, "none")); got != "" {
		t.Errorf("none: got %q", got)
	}
}

// drmProc fakes a process holding a render node. It has fdinfo but no fd
// dir, as another user's process looks without CAP_DAC_READ_SEARCH.
func drmProc(t *testing.T, proc string, pid int, name, fdinfo string) {
	fixture(t, filepath.Join(proc, fmt.Sprint(pid)), map[string]string{"comm": name + "\n", "fdinfo/5": fdinfo})
}

func TestDRMSampler(t *testing.T) {
	proc := t.TempDir()
	i915 := func(id string, render uint64) string {
		return fmt.Sprintf("pos:\t0\ndrm-driver:\ti915\ndrm-client-id:\t%s\ndrm-pdev:\t0000:00:02.0\ndrm-engine-render:\t%d ns\ndrm-engine-video:\t0 ns\n", id, render)
	}
	xe := func(cycles, total uint64) string {
		return fmt.Sprintf("drm-driver:\txe\ndrm-client-id:\t9\ndrm-pdev:\t0000:03:00.0\ndrm-cycles-rcs:\t%d\ndrm-total-cycles-rcs:\t%d\n", cycles, total)
	}
	drmProc(t, proc, 100, "Xorg", i915("3", 0))
	drmProc(t, proc, 200, "jellyfin", xe(0, 0))
	// A child sharing its parent's client counts once, for the parent.
	drmProc(t, proc, 101, "Xorg-child", i915("3", 0))
	// Not a DRM file.
	fixture(t, proc, map[string]string{"300/comm": "bash\n", "300/fdinfo/1": "pos:\t0\n"})

	clock := time.Unix(0, 0)
	d := &drmSampler{root: proc, now: func() time.Time { return clock }}
	d.sample()

	clock = clock.Add(time.Second)
	fixture(t, proc, map[string]string{
		"100/fdinfo/5": i915("3", 250_000_000),
		"101/fdinfo/5": i915("3", 250_000_000),
		"200/fdinfo/5": xe(600, 1000),
	})
	got := d.sample()
	want := map[string][]ProcessLoad{
		"0000:00:02.0": {{PID: 100, Name: "Xorg", Load: 25}},
		"0000:03:00.0": {{PID: 200, Name: "jellyfin", Load: 60}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

// TestDRMSamplerRecheck finds a render node on an fd it had cached as plain,
// once the cache empties.
func TestDRMSamplerRecheck(t *testing.T) {
	proc := t.TempDir()
	fixture(t, proc, map[string]string{"300/fdinfo/1": "pos:\t0\n"})
	d := &drmSampler{root: proc, now: time.Now}
	d.scan()

	fixture(t, proc, map[string]string{"300/fdinfo/1": "drm-client-id:\t7\ndrm-pdev:\t0000:00:02.0\n"})
	for i := 2; i < drmRecheck; i++ {
		if got := d.scan(); len(got) != 0 {
			t.Fatalf("scan %d: got %+v before the recheck", i, got)
		}
	}
	if got := d.scan(); len(got) != 1 {
		t.Fatalf("recheck: got %+v", got)
	}
}
