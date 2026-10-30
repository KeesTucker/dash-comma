package stats

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// Set at build time with -ldflags.
var (
	Version   = ""
	Buildhash = ""
)

// nodeArch spells GOARCH the way Node's os.arch() does, which dashdot sent.
var nodeArch = map[string]string{"amd64": "x64", "386": "ia32", "arm64": "arm64", "arm": "arm", "ppc64le": "ppc64", "s390x": "s390x", "riscv64": "riscv64"}

// resolveInHost follows symlinks, keeping absolute hops inside HostRoot, so
// a host /etc/os-release pointing at /usr/lib/os-release is read from the
// host and not from the container.
func resolveInHost(p string) string {
	for range 32 {
		target, err := os.Readlink(p)
		if err != nil {
			return p
		}
		if filepath.IsAbs(target) {
			p = fromHost(target)
		} else {
			p = filepath.Join(filepath.Dir(p), target)
		}
		// A relative link with enough ../ climbs out of the host mount.
		if HostRoot != "" && !strings.HasPrefix(p, HostRoot+"/") {
			p = fromHost(p)
		}
	}
	return p
}

func parseOSRelease(p string) map[string]string {
	f, err := os.Open(resolveInHost(p))
	if err != nil {
		return nil
	}
	defer f.Close()
	out := map[string]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		k, v, ok := strings.Cut(sc.Text(), "=")
		if !ok {
			continue
		}
		if u, err := strconv.Unquote(v); err == nil {
			v = u
		}
		out[k] = strings.Trim(v, `"'`)
	}
	return out
}

func OSStatic() (OSInfo, error) {
	var rel map[string]string
	for _, p := range []string{"/etc/os-release", "/usr/lib/os-release"} {
		if rel = parseOSRelease(fromHost(p)); rel != nil {
			break
		}
	}
	info := OSInfo{
		Platform:      runtime.GOOS,
		Distro:        "unknown",
		Release:       "unknown",
		Kernel:        readTrim(filepath.Join(procRoot, "sys/kernel/osrelease")),
		Arch:          nodeArch[runtime.GOARCH],
		DashVersion:   Version,
		DashBuildhash: Buildhash,
	}
	if info.Arch == "" {
		info.Arch = runtime.GOARCH
	}
	if info.DashVersion == "" {
		info.DashVersion = "unknown"
	}
	if rel != nil {
		if rel["NAME"] != "" {
			info.Distro = rel["NAME"]
		}
		switch {
		case rel["VERSION_ID"] != "":
			info.Release = rel["VERSION_ID"]
		case rel["BUILD_ID"] != "":
			info.Release = rel["BUILD_ID"]
		}
	}
	return info, nil
}

// Uptime is the host's uptime in seconds: /proc/uptime isn't namespaced.
func Uptime() float64 {
	f, _ := strconv.ParseFloat(strings.Fields(readTrim(filepath.Join(procRoot, "uptime"))+" 0")[0], 64)
	return f
}
