package stats

import (
	"bufio"
	"log"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/KeesTucker/dash-comma/internal/config"
)

const (
	vendorIntel  = "0x8086"
	vendorAMD    = "0x1002"
	vendorNVIDIA = "0x10de"
)

var gpuBrands = map[string]string{vendorIntel: "Intel", vendorAMD: "AMD", vendorNVIDIA: "NVIDIA"}

// gpuSampler reads one GPU's load: NVML for NVIDIA, sysfs for AMD and the
// perf PMU for Intel.
type gpuSampler interface {
	sample() GPUUsage
}

type gpu struct {
	slot    string // PCI address, 0000:03:00.0
	info    GPU
	sampler gpuSampler
}

// GPUs is every display controller on the PCI bus, in bus order.
type GPUs struct {
	list []gpu
}

func NewGPUs(cfg *config.Config) *GPUs {
	g := &GPUs{}
	devs, _ := filepath.Glob(filepath.Join(sysRoot, "bus/pci/devices/*"))
	sort.Strings(devs)
	for _, d := range devs {
		// PCI class 0x03 is a display controller (VGA, 3D or other).
		if !strings.HasPrefix(readTrim(filepath.Join(d, "class")), "0x03") {
			continue
		}
		vendor := readTrim(filepath.Join(d, "vendor"))
		device := readTrim(filepath.Join(d, "device"))
		vendorName, model := pciName(vendor, device)
		brand := gpuBrands[vendor]
		if brand == "" {
			brand = strings.TrimSpace(strings.ReplaceAll(vendorName, "Corporation", ""))
		}
		x := gpu{slot: filepath.Base(d), info: GPU{Brand: brand, Model: model}}

		switch vendor {
		case vendorNVIDIA:
			if s, name, mem, ok := newNVIDIA(x.slot); ok {
				x.sampler, x.info.Memory = s, mem
				x.info.Model = strings.TrimSpace(strings.TrimPrefix(name, "NVIDIA "))
			}
		case vendorAMD:
			x.sampler = amdSampler{dir: d}
			if vram, ok := readUint(filepath.Join(d, "mem_info_vram_total")); ok {
				x.info.Memory = float64(vram / 1024 / 1024)
			}
			if p := readTrim(filepath.Join(d, "product_name")); p != "" {
				x.info.Model = p
			}
		case vendorIntel:
			x.sampler = newIntel(x.slot)
		}

		if len(cfg.GPUBrandFilter) > 0 && !slices.Contains(cfg.GPUBrandFilter, x.info.Brand) {
			continue
		}
		if len(cfg.GPUModelFilter) > 0 && !slices.Contains(cfg.GPUModelFilter, x.info.Model) {
			continue
		}
		g.list = append(g.list, x)
	}
	return g
}

func (g *GPUs) Static() GPUInfo {
	out := GPUInfo{Layout: []GPU{}}
	for _, x := range g.list {
		out.Layout = append(out.Layout, x.info)
	}
	return out
}

func (g *GPUs) Sample() (GPULoad, error) {
	out := GPULoad{Layout: []GPUUsage{}}
	for _, x := range g.list {
		u := GPUUsage{}
		if x.sampler != nil {
			u = x.sampler.sample()
		}
		out.Layout = append(out.Layout, u)
	}
	return out, nil
}

type amdSampler struct{ dir string }

func (a amdSampler) sample() GPUUsage {
	load, _ := readInt(filepath.Join(a.dir, "gpu_busy_percent"))
	mem, _ := readInt(filepath.Join(a.dir, "mem_busy_percent"))
	return GPUUsage{Load: float64(load), Memory: float64(mem)}
}

var pciIDPaths = []string{"/usr/share/hwdata/pci.ids", "/usr/share/misc/pci.ids", "/usr/share/pci.ids"}

// pciName looks a device up in pci.ids. The model is the marketing name in
// brackets where there is one: "DG2 [Arc A750]" is "Arc A750".
func pciName(vendor, device string) (vendorName, model string) {
	v := strings.TrimPrefix(vendor, "0x")
	dv := strings.TrimPrefix(device, "0x")
	model = device
	for _, p := range pciIDPaths {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		defer f.Close()
		inVendor := false
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := sc.Text()
			if line == "" || line[0] == '#' {
				continue
			}
			if line[0] != '\t' {
				if inVendor {
					return vendorName, model
				}
				if strings.HasPrefix(line, v+"  ") {
					inVendor = true
					vendorName = strings.TrimSpace(line[len(v):])
				}
				continue
			}
			if inVendor && strings.HasPrefix(line, "\t"+dv+"  ") {
				name := strings.TrimSpace(line[len(dv)+1:])
				if i, j := strings.Index(name, "["), strings.LastIndex(name, "]"); i >= 0 && j > i {
					name = name[i+1 : j]
				}
				return vendorName, name
			}
		}
		return vendorName, model
	}
	log.Printf("pci.ids not found; GPU %s:%s shows by its PCI ID", vendor, device)
	return "", model
}
