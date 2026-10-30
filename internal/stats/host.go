package stats

import (
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// HostRoot is where the host's / is mounted in Docker (dashdot's /mnt/host),
// or "" when running on the host itself.
var HostRoot = ""

// fromHost maps a host path into the container. /proc and /sys aren't mapped:
// the container's own already show the host's CPUs, memory, disks and GPUs.
func fromHost(p string) string {
	if HostRoot == "" || strings.HasPrefix(p, HostRoot) {
		return p
	}
	return filepath.Join(HostRoot, p)
}

func readTrim(p string) string {
	b, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func readInt(p string) (int64, bool) {
	n, err := strconv.ParseInt(readTrim(p), 0, 64)
	return n, err == nil
}

func readUint(p string) (uint64, bool) {
	n, err := strconv.ParseUint(readTrim(p), 0, 64)
	return n, err == nil
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// parseCPUList reads a kernel CPU list like "0-3,8,10-11".
func parseCPUList(s string) []int {
	var out []int
	for _, part := range strings.Split(strings.TrimSpace(s), ",") {
		if part == "" {
			continue
		}
		lo, hi, isRange := strings.Cut(part, "-")
		a, err := strconv.Atoi(lo)
		if err != nil {
			continue
		}
		b := a
		if isRange {
			if b, err = strconv.Atoi(hi); err != nil {
				continue
			}
		}
		for i := a; i <= b; i++ {
			out = append(out, i)
		}
	}
	return out
}

// round keeps n decimal places, so the JSON doesn't carry float noise.
func round(f float64, n int) float64 {
	p := math.Pow10(n)
	return math.Round(f*p) / p
}
