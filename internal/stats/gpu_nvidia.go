package stats

import (
	"log"
	"sync"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
)

var (
	nvmlOnce sync.Once
	nvmlOK   bool
)

// newNVIDIA opens a card through NVML. NVML is loaded at runtime, so without
// the NVIDIA driver (or the container toolkit in Docker) the card still
// shows, just with no load.
func newNVIDIA(slot string) (s gpuSampler, name string, memMB float64, ok bool) {
	nvmlOnce.Do(func() {
		if ret := nvml.Init(); ret != nvml.SUCCESS {
			log.Printf("NVML unavailable, NVIDIA GPUs show no load: %s", nvml.ErrorString(ret))
			return
		}
		nvmlOK = true
	})
	if !nvmlOK {
		return nil, "", 0, false
	}
	dev, ret := nvml.DeviceGetHandleByPciBusId(slot)
	if ret != nvml.SUCCESS {
		log.Printf("NVML has no GPU at %s: %s", slot, nvml.ErrorString(ret))
		return nil, "", 0, false
	}
	name, _ = dev.GetName()
	if mem, ret := dev.GetMemoryInfo(); ret == nvml.SUCCESS {
		memMB = float64(mem.Total / 1024 / 1024)
	}
	return &nvidiaSampler{dev: dev, root: pidRoot()}, name, memMB, true
}

type nvidiaSampler struct {
	dev  nvml.Device
	root string

	mu       sync.Mutex
	lastSeen uint64
}

func (n *nvidiaSampler) sample() GPUUsage {
	u, ret := n.dev.GetUtilizationRates()
	if ret != nvml.SUCCESS {
		return GPUUsage{}
	}
	return GPUUsage{Load: float64(u.Gpu), Memory: float64(u.Memory), Processes: n.processes()}
}

// processes is each process's busiest of SM, encoder and decoder since the
// last call. NVML gives host PIDs, which the host's /proc names.
func (n *nvidiaSampler) processes() []ProcessLoad {
	n.mu.Lock()
	defer n.mu.Unlock()
	samples, ret := n.dev.GetProcessUtilization(n.lastSeen)
	if ret != nvml.SUCCESS {
		return nil
	}
	latest := map[uint32]nvml.ProcessUtilizationSample{}
	for _, s := range samples {
		if s.TimeStamp >= latest[s.Pid].TimeStamp {
			latest[s.Pid] = s
		}
		n.lastSeen = max(n.lastSeen, s.TimeStamp)
	}
	all := make([]ProcessLoad, 0, len(latest))
	for pid, s := range latest {
		all = append(all, ProcessLoad{PID: int(pid), Name: procName(n.root, int(pid)), Load: float64(max(s.SmUtil, s.EncUtil, s.DecUtil))})
	}
	return top(all)
}
