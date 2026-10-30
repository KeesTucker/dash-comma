package config

import (
	"encoding/json"
	"strings"
	"testing"
)

func envOf(vars map[string]string) func(string) (string, bool) {
	return func(k string) (string, bool) {
		v, ok := vars[k]
		return v, ok
	}
}

// The defaults go to the browser as JSON, so check them as JSON: dashdot
// sends [] for empty lists, {} for empty maps, and leaves unset options out.
func TestDefaultsJSON(t *testing.T) {
	b, err := json.Marshal(From(envOf(nil)))
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{
		`"port":3001`, `"routing_path":"/"`, `"fs_device_filter":[]`,
		`"widget_list":["os","cpu","storage","ram","network"]`,
		`"os_widget_grow":2.5`, `"storage_poll_interval":60000`, `"speed_test_interval":240`,
		`"cpu_temps_mode":"avg"`, `"show_dash_version":"icon_hover"`, `"page_title":"dash."`,
		`"override":{"storage_brands":{},"storage_types":{},"storage_sizes":{},"gpu_brands":[],"gpu_models":[],"gpu_memories":[]}`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %s in %s", want, s)
		}
	}
	for _, unwanted := range []string{"use_network_interface", "custom_host", "speed_test_interval_cron", `"os":`} {
		if strings.Contains(s, unwanted) {
			t.Errorf("unset %s should be left out", unwanted)
		}
	}
}

func TestOverrides(t *testing.T) {
	c := From(envOf(map[string]string{
		"DASHDOT_OVERRIDE_STORAGE_BRANDS": "sda=Samsung,stinker1=Media Pool",
		"DASHDOT_OVERRIDE_STORAGE_SIZES":  "sda=1000,sdb=",
		"DASHDOT_OVERRIDE_GPU_MEMORIES":   "8192",
		"DASHDOT_OVERRIDE_RAM_FREQUENCY":  "2133",
		"DASHDOT_PORT":                    "oops",
	}))
	if c.Override.StorageBrands["stinker1"] != "Media Pool" || c.Override.StorageSizes["sda"] != 1000 {
		t.Fatalf("storage overrides: %+v", c.Override)
	}
	if _, ok := c.Override.StorageSizes["sdb"]; ok {
		t.Fatal("an empty size should be left out")
	}
	if len(c.Override.GPUMemories) != 1 || c.Override.GPUMemories[0] != 8192 || *c.Override.RAMFrequency != 2133 {
		t.Fatalf("number overrides: %+v", c.Override)
	}
	if c.Port != 3001 {
		t.Fatalf("a bad port should fall back to 3001, got %v", c.Port)
	}
}
