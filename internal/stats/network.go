package stats

import (
	"bufio"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// defaultInterface reads the default route. In Docker it reads PID 1's table
// through the host's /proc, which sees the host's network namespace without
// nsenter or privileges.
func defaultInterface() string {
	paths := []string{filepath.Join(procRoot, "net/route")}
	if HostRoot != "" {
		paths = append([]string{fromHost("/proc/1/net/route")}, paths...)
	}
	for _, p := range paths {
		ifaces := defaultRoutes(p)
		if len(ifaces) > 1 {
			log.Printf("Multiple default network interfaces found %v, using %q", ifaces, ifaces[0])
		}
		if len(ifaces) > 0 {
			return ifaces[0]
		}
	}
	return ""
}

// defaultRoutes lists the interfaces with a default route in a
// /proc/net/route table.
func defaultRoutes(p string) []string {
	f, err := os.Open(p)
	if err != nil {
		return nil
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		// Destination 00000000 with mask 00000000 is the default route.
		if len(fields) >= 8 && fields[1] == "00000000" && fields[7] == "00000000" {
			out = append(out, fields[0])
		}
	}
	return out
}

// Network reads one interface's counters from sysfs. In Docker the host's
// /sys/class/net (under HostRoot) lists the host's interfaces, where the
// container's own only has its veth.
type Network struct {
	path string

	mu     sync.Mutex
	lastRx uint64
	lastTx uint64
	lastAt time.Time
}

func NewNetwork(iface *string) *Network {
	name := ""
	if iface != nil {
		name = *iface
	} else {
		name = defaultInterface()
	}
	n := &Network{}
	if name == "" {
		log.Print("Could not find the default network interface")
		return n
	}
	for _, p := range []string{fromHost("/sys/class/net/" + name), filepath.Join(sysRoot, "class/net", name)} {
		if exists(p) {
			n.path = p
			log.Printf("Using network interface %q (%s)", name, p)
			return n
		}
	}
	log.Printf("No path for network interface %q found", name)
	return n
}

func (n *Network) Static() NetworkInfo {
	if n.path == "" {
		return NetworkInfo{}
	}
	info := NetworkInfo{Type: "Wired"}
	switch {
	case exists(filepath.Join(n.path, "wireless")):
		info.Type = "Wireless"
	case exists(filepath.Join(n.path, "bridge")):
		info.Type = "Bridge"
	case exists(filepath.Join(n.path, "bonding")):
		info.Type = "Bond"
	case exists(filepath.Join(n.path, "tun_flags")):
		info.Type = "TAP"
	}
	// Wireless has no fixed speed, and a bridge or a down link reports -1.
	if info.Type != "Wireless" {
		if speed, ok := readInt(filepath.Join(n.path, "speed")); ok && speed > 0 {
			info.InterfaceSpeed = float64(speed)
		}
	}
	return info
}

// Sample is bytes per second up and down since the last call; the first
// call reports zero.
func (n *Network) Sample() (NetworkLoad, error) {
	if n.path == "" {
		return NetworkLoad{}, errors.New("could not get network stats: no interface")
	}
	rx, ok1 := readUint(filepath.Join(n.path, "statistics/rx_bytes"))
	tx, ok2 := readUint(filepath.Join(n.path, "statistics/tx_bytes"))
	if !ok1 || !ok2 {
		return NetworkLoad{}, errors.New("could not get network stats")
	}
	now := time.Now()

	n.mu.Lock()
	defer n.mu.Unlock()
	var out NetworkLoad
	if !n.lastAt.IsZero() && rx >= n.lastRx && tx >= n.lastTx {
		secs := now.Sub(n.lastAt).Seconds()
		if secs > 0 {
			out = NetworkLoad{Up: round(float64(tx-n.lastTx)/secs, 2), Down: round(float64(rx-n.lastRx)/secs, 2)}
		}
	}
	n.lastRx, n.lastTx, n.lastAt = rx, tx, now
	return out, nil
}
