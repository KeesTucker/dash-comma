package stats

import (
	"bufio"
	"encoding/binary"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func readMeminfo() (map[string]uint64, error) {
	f, err := os.Open(filepath.Join(procRoot, "meminfo"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	out := map[string]uint64{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		k, v, ok := strings.Cut(sc.Text(), ":")
		if !ok {
			continue
		}
		n, err := strconv.ParseUint(strings.TrimSuffix(strings.TrimSpace(v), " kB"), 10, 64)
		if err == nil {
			out[k] = n * 1024
		}
	}
	return out, sc.Err()
}

// RAMLoadNow is the bytes in use: everything the kernel can't hand back.
func RAMLoadNow() (RAMLoad, error) {
	m, err := readMeminfo()
	if err != nil {
		return 0, err
	}
	avail, ok := m["MemAvailable"]
	if !ok {
		avail = m["MemFree"] + m["Buffers"] + m["Cached"] + m["SReclaimable"]
	}
	return RAMLoad(m["MemTotal"] - avail), nil
}

func RAMStatic() (RAMInfo, error) {
	m, err := readMeminfo()
	if err != nil {
		return RAMInfo{Layout: []RAMModule{}}, err
	}
	return RAMInfo{Size: m["MemTotal"], Layout: ramModules()}, nil
}

var smbiosMemoryTypes = map[byte]string{
	0x03: "DRAM", 0x0F: "SDRAM", 0x12: "DDR", 0x13: "DDR2", 0x14: "DDR2 FB-DIMM",
	0x18: "DDR3", 0x19: "FBD2", 0x1A: "DDR4", 0x1B: "LPDDR", 0x1C: "LPDDR2",
	0x1D: "LPDDR3", 0x1E: "LPDDR4", 0x20: "HBM", 0x21: "HBM2", 0x22: "DDR5",
	0x23: "LPDDR5", 0x24: "HBM3",
}

// ramModules reads the SMBIOS memory devices (type 17). The raw tables are
// root-only, so an unprivileged container gets none, the same as dashdot
// without dmidecode; the RAM overrides cover it.
func ramModules() []RAMModule {
	out := []RAMModule{}
	entries, _ := filepath.Glob(filepath.Join(sysRoot, "firmware/dmi/entries/17-*/raw"))
	for _, e := range entries {
		raw, err := os.ReadFile(e)
		if err != nil {
			continue
		}
		if m, ok := parseMemoryDevice(raw); ok {
			out = append(out, m)
		}
	}
	return out
}

func parseMemoryDevice(raw []byte) (RAMModule, bool) {
	if len(raw) < 0x17 || raw[0] != 17 {
		return RAMModule{}, false
	}
	length := int(raw[1])
	if length > len(raw) || length < 0x17 {
		return RAMModule{}, false
	}
	size := binary.LittleEndian.Uint16(raw[0x0C:])
	if size == 0 {
		return RAMModule{}, false
	}
	m := RAMModule{Type: smbiosMemoryTypes[raw[0x12]]}
	speed := binary.LittleEndian.Uint16(raw[0x15:])
	if length >= 0x22 {
		if configured := binary.LittleEndian.Uint16(raw[0x20:]); configured != 0 && configured != 0xFFFF {
			speed = configured
		}
	}
	if speed != 0 && speed != 0xFFFF {
		m.Frequency = float64(speed)
	}
	if length > 0x17 {
		m.Brand = smbiosString(raw, length, raw[0x17])
	}
	return m, true
}

// smbiosString finds string number idx (1-based) in the strings that follow
// a structure's formatted part.
func smbiosString(raw []byte, length int, idx byte) string {
	if idx == 0 {
		return ""
	}
	strs := strings.Split(string(raw[length:]), "\x00")
	if int(idx) > len(strs) {
		return ""
	}
	return strings.TrimSpace(strs[idx-1])
}
