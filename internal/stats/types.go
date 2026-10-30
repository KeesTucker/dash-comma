// Package stats reads the host's hardware and load straight from procfs,
// sysfs and the GPU drivers. The types match dashdot's @dashdot/common, so
// the JSON on the wire is the same.
package stats

import "github.com/KeesTucker/dash-comma/internal/config"

type OSInfo struct {
	Platform      string  `json:"platform"`
	Distro        string  `json:"distro"`
	Release       string  `json:"release"`
	Kernel        string  `json:"kernel"`
	Arch          string  `json:"arch"`
	Uptime        float64 `json:"uptime"`
	DashVersion   string  `json:"dash_version"`
	DashBuildhash string  `json:"dash_buildhash"`
}

type CPUInfo struct {
	Brand     string  `json:"brand"`
	Model     string  `json:"model"`
	Cores     int     `json:"cores"`
	Ecores    int     `json:"ecores"`
	Pcores    int     `json:"pcores"`
	Threads   int     `json:"threads"`
	Frequency float64 `json:"frequency"`
}

type CoreLoad struct {
	Core int     `json:"core"`
	Load float64 `json:"load"`
	Temp float64 `json:"temp"`
}

type CPULoad []CoreLoad

type RAMModule struct {
	Brand     string  `json:"brand,omitempty"`
	Type      string  `json:"type,omitempty"`
	Frequency float64 `json:"frequency,omitempty"`
}

type RAMInfo struct {
	Size   uint64      `json:"size"`
	Layout []RAMModule `json:"layout"`
}

type RAMLoad float64

const (
	RaidZero = 0
	RaidOne  = 1
)

type Disk struct {
	Device string `json:"device"`
	Brand  string `json:"brand"`
	Type   string `json:"type"`
}

type StorageEntry struct {
	RaidType  *int   `json:"raidType,omitempty"`
	RaidLabel string `json:"raidLabel,omitempty"`
	RaidName  string `json:"raidName,omitempty"`
	Size      uint64 `json:"size"`
	Virtual   bool   `json:"virtual,omitempty"`
	Disks     []Disk `json:"disks"`
}

type StorageInfo []StorageEntry

// StorageLoad is the bytes used per StorageInfo entry, or -1 where no mount
// was found.
type StorageLoad []float64

type NetworkInfo struct {
	InterfaceSpeed float64 `json:"interfaceSpeed"`
	SpeedDown      float64 `json:"speedDown"`
	SpeedUp        float64 `json:"speedUp"`
	LastSpeedTest  float64 `json:"lastSpeedTest"`
	Type           string  `json:"type"`
	PublicIP       string  `json:"publicIp"`
}

type NetworkLoad struct {
	Up   float64 `json:"up"`
	Down float64 `json:"down"`
}

type GPU struct {
	Brand  string  `json:"brand"`
	Model  string  `json:"model"`
	Memory float64 `json:"memory"`
}

type GPUInfo struct {
	Layout []GPU `json:"layout"`
}

type GPUUsage struct {
	Load      float64            `json:"load"`
	Memory    float64            `json:"memory"`
	Engines   map[string]float64 `json:"engines,omitempty"`
	Processes []ProcessLoad      `json:"processes,omitempty"`
}

// ProcessLoad is one process's share of the whole CPU, or of a GPU's
// busiest engine. dashdot has no equivalent.
type ProcessLoad struct {
	PID  int     `json:"pid"`
	Name string  `json:"name"`
	Load float64 `json:"load"`
}

type GPULoad struct {
	Layout []GPUUsage `json:"layout"`
}

type HardwareInfo struct {
	OS      OSInfo      `json:"os"`
	CPU     CPUInfo     `json:"cpu"`
	RAM     RAMInfo     `json:"ram"`
	Storage StorageInfo `json:"storage"`
	Network NetworkInfo `json:"network"`
	GPU     GPUInfo     `json:"gpu"`
}

type ServerInfo struct {
	HardwareInfo
	Config *config.Config `json:"config"`
}

// EmptyHardware is what a disabled widget reports: dashdot's starting value.
func EmptyHardware() HardwareInfo {
	return HardwareInfo{
		RAM:     RAMInfo{Layout: []RAMModule{}},
		Storage: StorageInfo{},
		GPU:     GPUInfo{Layout: []GPU{}},
	}
}
