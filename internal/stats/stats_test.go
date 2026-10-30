package stats

import (
	"path/filepath"
	"testing"
)

func TestCPUModel(t *testing.T) {
	cases := map[string][2]string{
		"Intel(R) Core(TM) i7-8665U CPU @ 1.90GHz":   {"Intel", "Core i7-8665U"},
		"AMD Ryzen 9 5950X 16-Core Processor":        {"AMD", "Ryzen 9 5950X"},
		"Intel(R) Xeon(R) CPU E5-2680 v4 @ 2.40GHz":  {"Intel", "Xeon E5-2680 v4"},
		"AMD EPYC 7763 64-Core Processor":            {"AMD", "EPYC 7763"},
		"12th Gen Intel(R) Core(TM) i5-12600K":       {"Intel", "12th Gen Intel Core i5-12600K"},
		"Cortex-A72":                                 {"ARM", "Cortex-A72"},
	}
	for in, want := range cases {
		if got := cpuModel(in, want[0]); got != want[1] {
			t.Errorf("cpuModel(%q) = %q, want %q", in, got, want[1])
		}
	}
}

func TestEventConfig(t *testing.T) {
	dir := t.TempDir()
	fixture(t, dir, map[string]string{
		"events/rcs0-busy":           "config=0x10",
		"events/engine-active-ticks": "event=0x02",
		"format/event":               "config:0-11",
		"format/engine_class":        "config:20-27",
		"format/engine_instance":     "config:12-19",
		"format/gt":                  "config:60-63",
	})
	if got, err := eventConfig(dir, "rcs0-busy", nil); err != nil || got != 0x10 {
		t.Fatalf("i915: got %#x, %v", got, err)
	}
	got, err := eventConfig(dir, "engine-active-ticks", map[string]uint64{"gt": 1, "engine_class": 2, "engine_instance": 3})
	want := uint64(0x02) | 3<<12 | 2<<20 | 1<<60
	if err != nil || got != want {
		t.Fatalf("xe: got %#x, want %#x, %v", got, want, err)
	}
	if _, err := eventConfig(dir, "engine-active-ticks", map[string]uint64{"nope": 1}); err == nil {
		t.Fatal("want an error for a term with no format")
	}
}

func TestParseMemoryDevice(t *testing.T) {
	raw := make([]byte, 0x28)
	raw[0], raw[1] = 17, 0x28
	raw[0x0C], raw[0x0D] = 0x00, 0x20 // 8192 MB
	raw[0x12] = 0x1A                  // DDR4
	raw[0x15], raw[0x16] = 0x60, 0x09 // 2400 MT/s
	raw[0x17] = 1                     // manufacturer: string 1
	raw[0x20], raw[0x21] = 0x55, 0x08 // configured 2133 MT/s
	raw = append(raw, "Crucial\x00\x00"...)

	m, ok := parseMemoryDevice(raw)
	if !ok || m.Brand != "Crucial" || m.Type != "DDR4" || m.Frequency != 2133 {
		t.Fatalf("got %+v, %v", m, ok)
	}

	empty := append([]byte(nil), raw...)
	empty[0x0C], empty[0x0D] = 0, 0
	if _, ok := parseMemoryDevice(empty); ok {
		t.Fatal("an empty slot should be skipped")
	}
}

func TestParseCPUList(t *testing.T) {
	got := parseCPUList("0-2,8,10-11\n")
	want := []int{0, 1, 2, 8, 10, 11}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v", got)
		}
	}
}

func TestOSRelease(t *testing.T) {
	root := useFixtureHost(t, nil)
	fixture(t, root, map[string]string{"mnt/host/usr/lib/os-release": "NAME=\"Ubuntu\"\nVERSION_ID=\"24.04\"\n"})
	// The host's /etc/os-release is an absolute link, which must resolve
	// inside the host mount, not the container.
	link(t, "/usr/lib/os-release", filepath.Join(root, "mnt/host/etc/os-release"))
	info, _ := OSStatic()
	if info.Distro != "Ubuntu" || info.Release != "24.04" {
		t.Fatalf("got %+v", info)
	}
}
