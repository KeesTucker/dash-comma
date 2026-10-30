// Package config reads dashdot's DASHDOT_* environment variables. The struct
// is sent to the browser as-is (/config and the static-info event), so its
// JSON has to match dashdot's Config type key for key.
package config

import (
	"os"
	"strconv"
	"strings"
)

type Override struct {
	OS                    *string            `json:"os,omitempty"`
	Arch                  *string            `json:"arch,omitempty"`
	CPUBrand              *string            `json:"cpu_brand,omitempty"`
	CPUModel              *string            `json:"cpu_model,omitempty"`
	CPUCores              *float64           `json:"cpu_cores,omitempty"`
	CPUThreads            *float64           `json:"cpu_threads,omitempty"`
	CPUFrequency          *float64           `json:"cpu_frequency,omitempty"`
	RAMBrand              *string            `json:"ram_brand,omitempty"`
	RAMSize               *float64           `json:"ram_size,omitempty"`
	RAMType               *string            `json:"ram_type,omitempty"`
	RAMFrequency          *float64           `json:"ram_frequency,omitempty"`
	NetworkType           *string            `json:"network_type,omitempty"`
	NetworkSpeedUp        *float64           `json:"network_speed_up,omitempty"`
	NetworkSpeedDown      *float64           `json:"network_speed_down,omitempty"`
	NetworkInterfaceSpeed *float64           `json:"network_interface_speed,omitempty"`
	NetworkPublicIP       *string            `json:"network_public_ip,omitempty"`
	StorageBrands         map[string]string  `json:"storage_brands"`
	StorageTypes          map[string]string  `json:"storage_types"`
	StorageSizes          map[string]float64 `json:"storage_sizes"`
	GPUBrands             []string           `json:"gpu_brands"`
	GPUModels             []string           `json:"gpu_models"`
	GPUMemories           []float64          `json:"gpu_memories"`
}

type Config struct {
	Port                             float64  `json:"port"`
	RoutingPath                      string   `json:"routing_path"`
	RunningInDocker                  bool     `json:"running_in_docker"`
	UseNetworkInterface              *string  `json:"use_network_interface,omitempty"`
	SpeedTestFromPath                *string  `json:"speed_test_from_path,omitempty"`
	AcceptOoklaEULA                  bool     `json:"accept_ookla_eula"`
	FSDeviceFilter                   []string `json:"fs_device_filter"`
	FSTypeFilter                     []string `json:"fs_type_filter"`
	FSVirtualMounts                  []string `json:"fs_virtual_mounts"`
	DisableIntegrations              bool     `json:"disable_integrations"`
	DisableBackgroundStatsCollection bool     `json:"disable_background_stats_collection"`

	ShowDashVersion       string   `json:"show_dash_version"`
	ShowHost              bool     `json:"show_host"`
	CustomHost            *string  `json:"custom_host,omitempty"`
	PageTitle             string   `json:"page_title"`
	UseImperial           bool     `json:"use_imperial"`
	NetworkSpeedAsBytes   bool     `json:"network_speed_as_bytes"`
	AlwaysShowPercentages bool     `json:"always_show_percentages"`
	EnableCPUTemps        bool     `json:"enable_cpu_temps"`
	CPUTempsMode          string   `json:"cpu_temps_mode"`
	GPUBrandFilter        []string `json:"gpu_brand_filter"`
	GPUModelFilter        []string `json:"gpu_model_filter"`

	WidgetList       []string `json:"widget_list"`
	OSLabelList      []string `json:"os_label_list"`
	CPULabelList     []string `json:"cpu_label_list"`
	StorageLabelList []string `json:"storage_label_list"`
	RAMLabelList     []string `json:"ram_label_list"`
	NetworkLabelList []string `json:"network_label_list"`
	GPULabelList     []string `json:"gpu_label_list"`

	OSWidgetGrow     float64 `json:"os_widget_grow"`
	OSWidgetMinWidth float64 `json:"os_widget_min_width"`

	CPUWidgetGrow      float64 `json:"cpu_widget_grow"`
	CPUWidgetMinWidth  float64 `json:"cpu_widget_min_width"`
	CPUShownDatapoints float64 `json:"cpu_shown_datapoints"`
	CPUPollInterval    float64 `json:"cpu_poll_interval"`
	CPUCoresToggleMode string  `json:"cpu_cores_toggle_mode"`

	StorageWidgetItemsPerPage float64 `json:"storage_widget_items_per_page"`
	StorageWidgetGrow         float64 `json:"storage_widget_grow"`
	StorageWidgetMinWidth     float64 `json:"storage_widget_min_width"`
	StoragePollInterval       float64 `json:"storage_poll_interval"`

	RAMWidgetGrow      float64 `json:"ram_widget_grow"`
	RAMWidgetMinWidth  float64 `json:"ram_widget_min_width"`
	RAMShownDatapoints float64 `json:"ram_shown_datapoints"`
	RAMPollInterval    float64 `json:"ram_poll_interval"`

	SpeedTestInterval      float64 `json:"speed_test_interval"`
	SpeedTestIntervalCron  *string `json:"speed_test_interval_cron,omitempty"`
	NetworkWidgetGrow      float64 `json:"network_widget_grow"`
	NetworkWidgetMinWidth  float64 `json:"network_widget_min_width"`
	NetworkShownDatapoints float64 `json:"network_shown_datapoints"`
	NetworkPollInterval    float64 `json:"network_poll_interval"`

	GPUWidgetGrow      float64 `json:"gpu_widget_grow"`
	GPUWidgetMinWidth  float64 `json:"gpu_widget_min_width"`
	GPUShownDatapoints float64 `json:"gpu_shown_datapoints"`
	GPUPollInterval    float64 `json:"gpu_poll_interval"`

	Override Override `json:"override"`
}

// Widget reports whether name is in the widget list.
func (c *Config) Widget(name string) bool {
	for _, w := range c.WidgetList {
		if w == name {
			return true
		}
	}
	return false
}

// Load reads the config from the environment, with dashdot's defaults.
func Load() *Config {
	return From(os.LookupEnv)
}

// From reads the config through lookup, so tests can pass a fake environment.
func From(lookup func(string) (string, bool)) *Config {
	e := env{lookup}
	return &Config{
		Port:                             e.num("PORT", 3001),
		RoutingPath:                      e.str("ROUTING_PATH", "/"),
		RunningInDocker:                  e.flag("RUNNING_IN_DOCKER"),
		AcceptOoklaEULA:                  e.flag("ACCEPT_OOKLA_EULA"),
		UseNetworkInterface:              e.opt("USE_NETWORK_INTERFACE"),
		SpeedTestFromPath:                e.opt("SPEED_TEST_FROM_PATH"),
		FSDeviceFilter:                   e.list("FS_DEVICE_FILTER", ""),
		FSTypeFilter:                     e.list("FS_TYPE_FILTER", "cifs,9p,fuse.rclone,fuse.mergerfs,nfs4,iso9660,fuse.shfs,autofs"),
		FSVirtualMounts:                  e.list("FS_VIRTUAL_MOUNTS", ""),
		DisableIntegrations:              e.flag("DISABLE_INTEGRATIONS"),
		DisableBackgroundStatsCollection: e.flag("DISABLE_BACKGROUND_STATS_COLLECTION"),

		ShowDashVersion:       e.str("SHOW_DASH_VERSION", "icon_hover"),
		ShowHost:              e.flag("SHOW_HOST"),
		CustomHost:            e.opt("CUSTOM_HOST"),
		PageTitle:             e.str("PAGE_TITLE", "d,"),
		UseImperial:           e.flag("USE_IMPERIAL"),
		NetworkSpeedAsBytes:   e.flag("NETWORK_SPEED_AS_BYTES"),
		AlwaysShowPercentages: e.flag("ALWAYS_SHOW_PERCENTAGES"),
		EnableCPUTemps:        e.flag("ENABLE_CPU_TEMPS"),
		CPUTempsMode:          e.str("CPU_TEMPS_MODE", "avg"),
		GPUBrandFilter:        e.list("GPU_BRAND_FILTER", ""),
		GPUModelFilter:        e.list("GPU_MODEL_FILTER", ""),

		WidgetList:       e.list("WIDGET_LIST", "os,cpu,storage,ram,network"),
		OSLabelList:      e.list("OS_LABEL_LIST", "os,arch,up_since"),
		CPULabelList:     e.list("CPU_LABEL_LIST", "brand,model,cores,threads,frequency"),
		StorageLabelList: e.list("STORAGE_LABEL_LIST", "brand,type,size,raid"),
		RAMLabelList:     e.list("RAM_LABEL_LIST", "brand,type,size,frequency"),
		NetworkLabelList: e.list("NETWORK_LABEL_LIST", "type,speed_up,speed_down,interface_speed"),
		GPULabelList:     e.list("GPU_LABEL_LIST", "brand,model,memory"),

		OSWidgetGrow:     e.num("OS_WIDGET_GROW", 2.5),
		OSWidgetMinWidth: e.num("OS_WIDGET_MIN_WIDTH", 300),

		CPUWidgetGrow:      e.num("CPU_WIDGET_GROW", 4),
		CPUWidgetMinWidth:  e.num("CPU_WIDGET_MIN_WIDTH", 500),
		CPUShownDatapoints: e.num("CPU_SHOWN_DATAPOINTS", 20),
		CPUPollInterval:    e.num("CPU_POLL_INTERVAL", 1000),
		CPUCoresToggleMode: e.str("CPU_CORES_TOGGLE_MODE", "toggle"),

		StorageWidgetItemsPerPage: e.num("STORAGE_WIDGET_ITEMS_PER_PAGE", 3),
		StorageWidgetGrow:         e.num("STORAGE_WIDGET_GROW", 3.5),
		StorageWidgetMinWidth:     e.num("STORAGE_WIDGET_MIN_WIDTH", 500),
		StoragePollInterval:       e.num("STORAGE_POLL_INTERVAL", 60000),

		RAMWidgetGrow:      e.num("RAM_WIDGET_GROW", 4),
		RAMWidgetMinWidth:  e.num("RAM_WIDGET_MIN_WIDTH", 500),
		RAMShownDatapoints: e.num("RAM_SHOWN_DATAPOINTS", 20),
		RAMPollInterval:    e.num("RAM_POLL_INTERVAL", 1000),

		SpeedTestInterval:      e.num("SPEED_TEST_INTERVAL", 60*4),
		SpeedTestIntervalCron:  e.opt("SPEED_TEST_INTERVAL_CRON"),
		NetworkWidgetGrow:      e.num("NETWORK_WIDGET_GROW", 6),
		NetworkWidgetMinWidth:  e.num("NETWORK_WIDGET_MIN_WIDTH", 500),
		NetworkShownDatapoints: e.num("NETWORK_SHOWN_DATAPOINTS", 20),
		NetworkPollInterval:    e.num("NETWORK_POLL_INTERVAL", 1000),

		GPUWidgetGrow:      e.num("GPU_WIDGET_GROW", 6),
		GPUWidgetMinWidth:  e.num("GPU_WIDGET_MIN_WIDTH", 700),
		GPUShownDatapoints: e.num("GPU_SHOWN_DATAPOINTS", 20),
		GPUPollInterval:    e.num("GPU_POLL_INTERVAL", 1000),

		Override: Override{
			OS:                    e.opt("OVERRIDE_OS"),
			Arch:                  e.opt("OVERRIDE_ARCH"),
			CPUBrand:              e.opt("OVERRIDE_CPU_BRAND"),
			CPUModel:              e.opt("OVERRIDE_CPU_MODEL"),
			CPUCores:              e.optNum("OVERRIDE_CPU_CORES"),
			CPUThreads:            e.optNum("OVERRIDE_CPU_THREADS"),
			CPUFrequency:          e.optNum("OVERRIDE_CPU_FREQUENCY"),
			RAMBrand:              e.opt("OVERRIDE_RAM_BRAND"),
			RAMSize:               e.optNum("OVERRIDE_RAM_SIZE"),
			RAMType:               e.opt("OVERRIDE_RAM_TYPE"),
			RAMFrequency:          e.optNum("OVERRIDE_RAM_FREQUENCY"),
			NetworkType:           e.opt("OVERRIDE_NETWORK_TYPE"),
			NetworkSpeedUp:        e.optNum("OVERRIDE_NETWORK_SPEED_UP"),
			NetworkSpeedDown:      e.optNum("OVERRIDE_NETWORK_SPEED_DOWN"),
			NetworkInterfaceSpeed: e.optNum("OVERRIDE_NETWORK_INTERFACE_SPEED"),
			NetworkPublicIP:       e.opt("OVERRIDE_NETWORK_PUBLIC_IP"),
			StorageBrands:         e.strMap("OVERRIDE_STORAGE_BRANDS"),
			StorageSizes:          e.numMap("OVERRIDE_STORAGE_SIZES"),
			StorageTypes:          e.strMap("OVERRIDE_STORAGE_TYPES"),
			GPUBrands:             e.list("OVERRIDE_GPU_BRANDS", ""),
			GPUModels:             e.list("OVERRIDE_GPU_MODELS", ""),
			GPUMemories:           e.numList("OVERRIDE_GPU_MEMORIES"),
		},
	}
}

type env struct {
	lookup func(string) (string, bool)
}

func (e env) opt(key string) *string {
	v, ok := e.lookup("DASHDOT_" + key)
	if !ok {
		return nil
	}
	return &v
}

func (e env) str(key, def string) string {
	if v := e.opt(key); v != nil {
		return *v
	}
	return def
}

func (e env) flag(key string) bool {
	return e.str(key, "") == "true"
}

// parseNum is dashdot's numNull: empty means unset. Unlike JS's `+val` it
// treats a typo as unset too, rather than sending NaN to the browser.
func parseNum(s string) (float64, bool) {
	if strings.TrimSpace(s) == "" {
		return 0, false
	}
	n, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return n, err == nil
}

func (e env) num(key string, def float64) float64 {
	if n, ok := parseNum(e.str(key, "")); ok {
		return n
	}
	return def
}

func (e env) optNum(key string) *float64 {
	if n, ok := parseNum(e.str(key, "")); ok {
		return &n
	}
	return nil
}

// list splits on commas, and is never nil: dashdot sends [] for an empty list.
func (e env) list(key, def string) []string {
	s := e.str(key, def)
	if s == "" {
		return []string{}
	}
	return strings.Split(s, ",")
}

func (e env) numList(key string) []float64 {
	out := []float64{}
	for _, s := range e.list(key, "") {
		n, _ := parseNum(s)
		out = append(out, n)
	}
	return out
}

func (e env) strMap(key string) map[string]string {
	out := map[string]string{}
	for _, p := range e.list(key, "") {
		k, v, _ := strings.Cut(p, "=")
		out[k] = v
	}
	return out
}

func (e env) numMap(key string) map[string]float64 {
	out := map[string]float64{}
	for _, p := range e.list(key, "") {
		k, v, _ := strings.Cut(p, "=")
		if n, ok := parseNum(v); ok {
			out[k] = n
		}
	}
	return out
}
