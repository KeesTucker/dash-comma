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
	return nvidiaSampler{dev}, name, memMB, true
}

type nvidiaSampler struct{ dev nvml.Device }

func (n nvidiaSampler) sample() GPUUsage {
	u, ret := n.dev.GetUtilizationRates()
	if ret != nvml.SUCCESS {
		return GPUUsage{}
	}
	return GPUUsage{Load: float64(u.Gpu), Memory: float64(u.Memory)}
}
