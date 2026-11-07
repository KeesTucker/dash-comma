package stats

import (
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/asm"
	ebpflink "github.com/cilium/ebpf/link"
	"golang.org/x/sys/unix"
)

// netFDsEnv carries the counter map's fd across the re-exec.
const netFDsEnv = "DASH_COMMA_NET_FD"

var (
	// netCounts is each cgroup's bytes in and out: key the cgroup ID, value
	// [ingress, egress]. Held here so the GC never closes its fd.
	netCounts *ebpf.Map
	// netKeep holds the pre-exec objects whose fds must stay open until exec.
	netKeep []any
)

// SetupNetCounters attaches two cgroup_skb programs to the host's root
// cgroup, which count every socket's traffic against its own cgroup. It
// then drops BPF and NET_ADMIN for good by re-executing dash-comma, so only
// startup ever holds them. Call it at the top of main: on success it doesn't
// return, and the new process picks the counters up here.
func SetupNetCounters(enabled bool) {
	if v, ok := os.LookupEnv(netFDsEnv); ok {
		_ = os.Unsetenv(netFDsEnv)
		if fd, err := strconv.Atoi(v); err == nil && fd >= 0 {
			if m, err := ebpf.NewMapFromFD(fd); err == nil {
				netCounts = m
			} else {
				log.Printf("network groups: %v", err)
			}
		}
		return
	}
	if !permitted(capBPF) && !permitted(capNetAdmin) {
		return
	}
	fd := -1
	if enabled {
		var err error
		if fd, err = attachNetCounters(cgroupRoot()); err != nil {
			log.Printf("network groups unavailable: %v", err)
		}
	}
	reexecWithout(fd)
}

func cgroupRoot() string {
	if HostRoot != "" {
		if p := filepath.Join(HostRoot, "sys/fs/cgroup"); exists(filepath.Join(p, "cgroup.controllers")) {
			return p
		}
	}
	return filepath.Join(sysRoot, "fs/cgroup")
}

// attachNetCounters returns the map's fd. It and the links' fds are left
// open across exec; the links keep the programs attached.
func attachNetCounters(root string) (int, error) {
	fd, err := -1, error(nil)
	withCaps(func() { fd, err = loadNetCounters(root) }, capBPF, capNetAdmin)
	return fd, err
}

func loadNetCounters(root string) (int, error) {
	m, err := ebpf.NewMap(&ebpf.MapSpec{Name: "dash_net", Type: ebpf.LRUHash, KeySize: 8, ValueSize: 16, MaxEntries: 4096})
	if err != nil {
		return -1, err
	}
	netKeep = append(netKeep, m)
	cg, err := os.Open(root)
	if err != nil {
		return -1, err
	}
	defer cg.Close()
	fds := []int{m.FD()}
	for i, attach := range []ebpf.AttachType{ebpf.AttachCGroupInetIngress, ebpf.AttachCGroupInetEgress} {
		prog, err := ebpf.NewProgram(&ebpf.ProgramSpec{
			Type:         ebpf.CGroupSKB,
			AttachType:   attach,
			License:      "MIT",
			Instructions: netCounterProg(m.FD(), int16(i*8)),
		})
		if err != nil {
			return -1, err
		}
		// The link holds the program, so its own fd can go.
		defer func() { _ = prog.Close() }()
		l, err := ebpflink.AttachRawLink(ebpflink.RawLinkOptions{Target: int(cg.Fd()), Program: prog, Attach: attach})
		if err != nil {
			return -1, err
		}
		netKeep = append(netKeep, l)
		fds = append(fds, l.FD())
	}
	for _, fd := range fds {
		if _, err := unix.FcntlInt(uintptr(fd), unix.F_SETFD, 0); err != nil {
			return -1, err
		}
	}
	return m.FD(), nil
}

// netCounterProg adds the packet's length to its socket's cgroup's counter
// at offset (0 in, 8 out) and lets the packet through.
func netCounterProg(mapFD int, offset int16) asm.Instructions {
	lookup := func() asm.Instructions {
		return asm.Instructions{
			asm.LoadMapPtr(asm.R1, mapFD),
			asm.Mov.Reg(asm.R2, asm.RFP),
			asm.Add.Imm(asm.R2, -8),
			asm.FnMapLookupElem.Call(),
		}
	}
	insns := asm.Instructions{
		asm.Mov.Reg(asm.R6, asm.R1),
		asm.LoadMem(asm.R7, asm.R6, 0, asm.Word), // __sk_buff.len
		asm.FnSkbCgroupId.Call(),
		asm.StoreMem(asm.RFP, -8, asm.R0, asm.DWord),
	}
	insns = append(insns, lookup()...)
	insns = append(insns,
		asm.JNE.Imm(asm.R0, 0, "add"),
		// cilium's StoreImm encodes a DWord as an invalid opcode, so zero
		// the new value through a register.
		asm.Mov.Imm(asm.R1, 0),
		asm.StoreMem(asm.RFP, -24, asm.R1, asm.DWord),
		asm.StoreMem(asm.RFP, -16, asm.R1, asm.DWord),
		asm.LoadMapPtr(asm.R1, mapFD),
		asm.Mov.Reg(asm.R2, asm.RFP),
		asm.Add.Imm(asm.R2, -8),
		asm.Mov.Reg(asm.R3, asm.RFP),
		asm.Add.Imm(asm.R3, -24),
		asm.Mov.Imm(asm.R4, int32(ebpf.UpdateNoExist)),
		asm.FnMapUpdateElem.Call(),
	)
	insns = append(insns, lookup()...)
	return append(insns,
		asm.JEq.Imm(asm.R0, 0, "out"),
		asm.AddAtomic.Mem(asm.R0, asm.R7, asm.DWord, offset).WithSymbol("add"),
		asm.Mov.Imm(asm.R0, 1).WithSymbol("out"),
		asm.Return(),
	)
}

// reexecWithout drops BPF, NET_ADMIN and SETPCAP from the bounding set and
// execs dash-comma again. Exec grants the file's caps masked by the bounding
// set, so the new process never gets them back. Clearing them in place
// wouldn't do: caps are per thread, and with cgo Go can't change every
// thread's. The bounding set is per thread too, and exec uses the caller's,
// so this all runs on one locked thread.
func reexecWithout(fd int) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if !permitted(capSetpcap) {
		log.Print("Keeping BPF and NET_ADMIN: dropping them needs SETPCAP")
		return
	}
	hdr, data, err := capGet()
	if err != nil {
		log.Printf("Keeping BPF and NET_ADMIN: %v", err)
		return
	}
	data[0].Effective |= 1 << capSetpcap
	if err := unix.Capset(&hdr, &data[0]); err != nil {
		log.Printf("Keeping BPF and NET_ADMIN: %v", err)
		return
	}
	for _, c := range []int{capBPF, capNetAdmin, capSetpcap} {
		if err := unix.Prctl(unix.PR_CAPBSET_DROP, uintptr(c), 0, 0, 0); err != nil {
			log.Printf("Keeping BPF and NET_ADMIN: %v", err)
			return
		}
	}
	env := append(os.Environ(), netFDsEnv+"="+strconv.Itoa(fd))
	err = unix.Exec("/proc/self/exe", os.Args, env)
	runtime.KeepAlive(netKeep)
	log.Fatalf("re-exec without BPF and NET_ADMIN: %v", err)
}

// NetGroups turns the counters into each cgroup's rate: a container, a
// systemd service or a desktop app.
type NetGroups struct {
	mu     sync.Mutex
	root   string
	now    func() time.Time
	read   func() map[uint64][2]uint64
	last   map[uint64][2]uint64
	lastAt time.Time

	names    map[uint64]string
	paths    map[uint64]string
	walkedAt time.Time
}

// netRewalk is how often an unknown cgroup ID can trigger a fresh walk of
// the cgroup tree.
const netRewalk = 10 * time.Second

func NewNetGroups() *NetGroups {
	return &NetGroups{root: cgroupRoot(), now: time.Now, read: readNetCounts, names: map[uint64]string{}}
}

func readNetCounts() map[uint64][2]uint64 {
	if netCounts == nil {
		return nil
	}
	out := map[uint64][2]uint64{}
	var k uint64
	var v [2]uint64
	it := netCounts.Iterate()
	for it.Next(&k, &v) {
		out[k] = v
	}
	return out
}

// Sample is the busiest cgroups' bytes per second since the last call.
func (n *NetGroups) Sample() ([]NetworkGroup, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	cur, at := n.read(), n.now()
	prev, secs := n.last, at.Sub(n.lastAt).Seconds()
	n.last, n.lastAt = cur, at
	out := []NetworkGroup{}
	if prev == nil || secs <= 0 {
		return out, nil
	}
	type rate struct {
		id       uint64
		down, up float64
	}
	var all []rate
	for id, v := range cur {
		// An LRU eviction restarts a counter at zero; skip it once.
		if p, ok := prev[id]; ok && v[0] >= p[0] && v[1] >= p[1] && v != p {
			all = append(all, rate{id, float64(v[0]-p[0]) / secs, float64(v[1]-p[1]) / secs})
		}
	}
	sort.Slice(all, func(i, j int) bool {
		a, b := all[i].down+all[i].up, all[j].down+all[j].up
		if a != b {
			return a > b
		}
		return all[i].id < all[j].id
	})
	all = all[:min(len(all), topProcesses)]
	ids := make([]uint64, len(all))
	for i, r := range all {
		ids[i] = r.id
	}
	n.nameAll(ids)
	for _, r := range all {
		name, ok := n.names[r.id]
		if !ok {
			// Gone before we looked, or new since the last walk.
			name = "cgroup " + strconv.FormatUint(r.id, 10)
		}
		out = append(out, NetworkGroup{Name: name, Down: round(r.down, 2), Up: round(r.up, 2)})
	}
	return out, nil
}

// nameAll names the ids it hasn't yet, with n.mu held. Only the few busiest
// get named, and each just once, so a sample rarely walks the cgroup tree or
// /proc.
func (n *NetGroups) nameAll(ids []uint64) {
	var missing []uint64
	for _, id := range ids {
		if _, ok := n.names[id]; !ok {
			missing = append(missing, id)
		}
	}
	if len(missing) == 0 {
		return
	}
	for _, id := range missing {
		if _, ok := n.paths[id]; !ok && n.now().Sub(n.walkedAt) >= netRewalk {
			n.paths, n.walkedAt = walkCgroups(n.root), n.now()
			break
		}
	}
	want := map[string]bool{}
	for _, id := range missing {
		if rel, ok := n.paths[id]; ok {
			want[rel] = true
		}
	}
	if len(want) == 0 {
		return
	}
	biggest := biggestProcesses(pidRoot(), n.nsRoot(), want)
	for _, id := range missing {
		if rel, ok := n.paths[id]; ok {
			n.names[id] = cgroupName(rel, biggest[rel])
		}
	}
}

// nsRoot is where dash-comma's own cgroup namespace sits in the host's tree:
// in Docker, its container's cgroup. /proc/<pid>/cgroup paths are relative
// to it.
func (n *NetGroups) nsRoot() string {
	info, err := os.Stat(filepath.Join(sysRoot, "fs/cgroup"))
	if err != nil {
		return "."
	}
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		if rel, ok := n.paths[st.Ino]; ok {
			return rel
		}
	}
	return "."
}

// walkCgroups maps each cgroup's ID, its directory's inode, to its path
// under root.
func walkCgroups(root string) map[uint64]string {
	out := map[uint64]string{}
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if st, ok := info.Sys().(*syscall.Stat_t); ok {
			rel, _ := filepath.Rel(root, p)
			out[st.Ino] = rel
		}
		return nil
	})
	return out
}

var containerScopeRe = regexp.MustCompile(`^(?:docker|libpod|cri-containerd|crio)-([0-9a-f]{12,})\.scope$`)

// cgroupName names a cgroup for people: a service by its unit, and anything
// else, containers included, by its biggest process, which is the app rather
// than its init (Sonarr, not s6-svscan).
func cgroupName(rel, biggest string) string {
	if rel == "." {
		return "kernel"
	}
	base := filepath.Base(rel)
	if unit, ok := strings.CutSuffix(base, ".service"); ok {
		return unit
	}
	if biggest != "" {
		return biggest
	}
	if m := containerScopeRe.FindStringSubmatch(base); m != nil {
		return "container " + m[1][:12]
	}
	return strings.TrimSuffix(strings.TrimSuffix(base, ".scope"), ".slice")
}

// biggestProcesses names the process with the most resident memory in each
// wanted cgroup. It goes from each process to its cgroup, not the other way:
// cgroup.procs lists pids as the reader's pid namespace sees them, which in
// Docker turns every host pid into 0.
func biggestProcesses(proc, ns string, want map[string]bool) map[string]string {
	type best struct {
		name string
		rss  int64
	}
	found := map[string]best{}
	for _, pid := range listPIDs(proc) {
		dir := filepath.Join(proc, strconv.Itoa(pid))
		rel, ok := hostCgroup(ns, readTrim(filepath.Join(dir, "cgroup")))
		if !ok || !want[rel] {
			continue
		}
		statm := strings.Fields(readTrim(filepath.Join(dir, "statm")))
		if len(statm) < 2 {
			continue
		}
		rss, err := strconv.ParseInt(statm[1], 10, 64)
		if b, seen := found[rel]; err != nil || (seen && rss <= b.rss) {
			continue
		}
		if comm := readTrim(filepath.Join(dir, "comm")); comm != "" {
			found[rel] = best{comm, rss}
		}
	}
	out := map[string]string{}
	for rel, b := range found {
		out[rel] = b.name
	}
	return out
}

// hostCgroup turns a /proc/<pid>/cgroup v2 line ("0::/../tdarr.slice/x"),
// relative to the reader's namespace root ns, into a path under the host's
// root as walkCgroups names it.
func hostCgroup(ns, line string) (string, bool) {
	p, ok := strings.CutPrefix(line, "0::")
	if !ok {
		return "", false
	}
	rel := strings.TrimPrefix(path.Clean("/"+ns+"/"+p), "/")
	if rel == "" {
		rel = "."
	}
	return rel, true
}
