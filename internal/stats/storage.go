package stats

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/KeesTucker/dash-comma/internal/config"
	"golang.org/x/sys/unix"
)

// statfs is swapped in tests.
var statfs = func(path string) (size, used uint64, err error) {
	var st unix.Statfs_t
	if err := unix.Statfs(path, &st); err != nil {
		return 0, 0, err
	}
	bs := uint64(st.Bsize)
	return st.Blocks * bs, (st.Blocks - st.Bfree) * bs, nil
}

var skipBlockRe = regexp.MustCompile(`^(loop|ram|zram|md|dm-|sr|fd)\d*`)

type block struct {
	name  string
	dev   string // major:minor
	size  uint64
	parts []string
	udev  map[string]string
}

func readBlock(dir string) block {
	name := filepath.Base(dir)
	b := block{name: name, dev: readTrim(filepath.Join(dir, "dev"))}
	if sectors, ok := readUint(filepath.Join(dir, "size")); ok {
		b.size = sectors * 512
	}
	b.udev = readUdev(b.dev)
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), name) && exists(filepath.Join(dir, e.Name(), "partition")) {
			b.parts = append(b.parts, e.Name())
		}
	}
	sort.Strings(b.parts)
	return b
}

// readUdev reads udev's database entry for a block device: the filesystem
// type, label and model that lsblk shows. It's under the host's /run.
func readUdev(dev string) map[string]string {
	out := map[string]string{}
	if dev == "" {
		return out
	}
	f, err := os.Open(fromHost("/run/udev/data/b" + dev))
	if err != nil {
		return out
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if k, v, ok := strings.Cut(strings.TrimPrefix(sc.Text(), "E:"), "="); ok {
			out[k] = v
		}
	}
	return out
}

// sysBlock is a device or partition's directory in /sys/class/block.
func sysBlock(name string) string {
	return filepath.Join(sysRoot, "class/block", name)
}

// deviceTree is a disk, its partitions and everything stacked on them (md
// arrays, LVM and dm-crypt volumes), by name.
func deviceTree(disk string) []string {
	seen := map[string]bool{}
	var walk func(string)
	walk = func(n string) {
		if seen[n] {
			return
		}
		seen[n] = true
		dir := sysBlock(n)
		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), n) && exists(filepath.Join(dir, e.Name(), "partition")) {
				walk(e.Name())
			}
		}
		hs, _ := os.ReadDir(filepath.Join(dir, "holders"))
		for _, h := range hs {
			walk(h.Name())
		}
	}
	walk(disk)
	out := make([]string, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

var diskBrands = []struct{ prefix, brand string }{
	{"wdc", "Western Digital"}, {"wd", "Western Digital"}, {"western digital", "Western Digital"},
	{"st", "Seagate"}, {"seagate", "Seagate"}, {"samsung", "Samsung"}, {"mz", "Samsung"},
	{"ct", "Crucial"}, {"crucial", "Crucial"}, {"kingston", "Kingston"}, {"sa400", "Kingston"},
	{"toshiba", "Toshiba"}, {"hgst", "HGST"}, {"hitachi", "Hitachi"}, {"intel", "Intel"},
	{"micron", "Micron"}, {"sandisk", "SanDisk"}, {"liteon", "Lite-On"}, {"lite-on", "Lite-On"},
	{"sk hynix", "SK hynix"}, {"hfs", "SK hynix"}, {"adata", "ADATA"}, {"corsair", "Corsair"},
	{"sabrent", "Sabrent"}, {"transcend", "Transcend"}, {"teamgroup", "TeamGroup"},
	{"pny", "PNY"}, {"patriot", "Patriot"}, {"phison", "Phison"}, {"kioxia", "Kioxia"},
	{"solidigm", "Solidigm"}, {"hp", "HP"}, {"lexar", "Lexar"}, {"netac", "Netac"},
}

// diskBrand guesses the maker from the model, as systeminformation does:
// "WDC WD40EFRX-68N32N0" is Western Digital, "ST4000VN008" is Seagate.
func diskBrand(model, vendor, fallback string) string {
	m := strings.ToLower(strings.TrimSpace(model))
	for _, b := range diskBrands {
		if strings.HasPrefix(m, b.prefix) {
			return b.brand
		}
	}
	v := strings.TrimSpace(vendor)
	if v != "" && !strings.EqualFold(v, "ATA") {
		return v
	}
	return fallback
}

type mount struct {
	dev    string // major:minor
	source string
	point  string
	fstype string
}

func readMounts() []mount {
	f, err := os.Open(filepath.Join(procRoot, "self/mountinfo"))
	if err != nil {
		return nil
	}
	defer f.Close()
	var out []mount
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		pre, post, ok := strings.Cut(sc.Text(), " - ")
		a, b := strings.Fields(pre), strings.Fields(post)
		if !ok || len(a) < 5 || len(b) < 2 {
			continue
		}
		out = append(out, mount{dev: a[2], point: unescapeMount(a[4]), fstype: b[0], source: unescapeMount(b[1])})
	}
	return out
}

// unescapeMount undoes mountinfo's octal escapes (\040 for a space).
func unescapeMount(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var out strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+3 < len(s) {
			var c byte
			valid := true
			for _, d := range s[i+1 : i+4] {
				if d < '0' || d > '7' {
					valid = false
					break
				}
				c = c*8 + byte(d-'0')
			}
			if valid {
				out.WriteByte(c)
				i += 3
				continue
			}
		}
		out.WriteByte(s[i])
	}
	return out.String()
}

// hostMounts are the mounts that belong to the host: under HostRoot in
// Docker (the container's own overlay and volumes don't count), and not of
// a filtered type.
func hostMounts(cfg *config.Config, all []mount) []mount {
	var out []mount
	for _, m := range all {
		if HostRoot != "" && m.point != HostRoot && !strings.HasPrefix(m.point, HostRoot+"/") {
			continue
		}
		if slices.Contains(cfg.FSTypeFilter, m.fstype) {
			continue
		}
		out = append(out, m)
	}
	return out
}

// dmName maps a /dev/mapper name to its dm-N device.
func dmName(source string) string {
	name := strings.TrimPrefix(source, "/dev/mapper/")
	dms, _ := filepath.Glob(filepath.Join(sysRoot, "class/block/dm-*"))
	for _, d := range dms {
		if readTrim(filepath.Join(d, "dm/name")) == name {
			return filepath.Base(d)
		}
	}
	return ""
}

func StorageStatic(cfg *config.Config) (StorageInfo, error) {
	dirs, _ := filepath.Glob(filepath.Join(sysRoot, "block/*"))
	sort.Strings(dirs)
	out := StorageInfo{}
	for _, d := range dirs {
		b := readBlock(d)
		if b.size == 0 || skipBlockRe.MatchString(b.name) ||
			slices.Contains(cfg.FSDeviceFilter, b.name) ||
			slices.Contains(cfg.FSTypeFilter, b.udev["ID_FS_TYPE"]) {
			continue
		}

		disk := describeDisk(b, d)
		if md := raidOf(b); md != "" {
			if i := slices.IndexFunc(out, func(e StorageEntry) bool { return e.RaidName == md }); i >= 0 {
				out[i].Disks = append(out[i].Disks, disk)
				continue
			}
			out = append(out, raidEntry(md, disk))
			continue
		}
		out = append(out, StorageEntry{Size: b.size, Disks: []Disk{disk}})
	}
	return virtualMounts(cfg, out), nil
}

// describeDisk names a whole disk's maker and kind from sysfs and udev.
func describeDisk(b block, dir string) Disk {
	model := strings.TrimSpace(readTrim(filepath.Join(dir, "device/model")))
	if model == "" {
		model = strings.ReplaceAll(b.udev["ID_MODEL"], "_", " ")
	}
	disk := Disk{Device: b.name, Brand: diskBrand(model, readTrim(filepath.Join(dir, "device/vendor")), b.name)}
	switch {
	case strings.HasPrefix(b.name, "nvme"):
		disk.Type = "NVMe"
	case readTrim(filepath.Join(dir, "queue/rotational")) == "1":
		disk.Type = "HD"
	default:
		disk.Type = "SSD"
	}
	return disk
}

// raidOf finds the md array a disk, or one of its partitions, belongs to.
func raidOf(b block) string {
	names := append([]string{b.name}, b.parts...)
	for _, n := range names {
		hs, _ := os.ReadDir(filepath.Join(sysBlock(n), "holders"))
		for _, h := range hs {
			if strings.HasPrefix(h.Name(), "md") {
				return h.Name()
			}
		}
	}
	return ""
}

func raidEntry(md string, disk Disk) StorageEntry {
	dir := sysBlock(md)
	e := StorageEntry{RaidName: md, Disks: []Disk{disk}}
	if sectors, ok := readUint(filepath.Join(dir, "size")); ok {
		e.Size = sectors * 512
	}
	t := RaidOne
	if readTrim(filepath.Join(dir, "md/level")) == "raid0" {
		t = RaidZero
	}
	e.RaidType = &t
	udev := readUdev(readTrim(filepath.Join(dir, "dev")))
	e.RaidLabel = udev["ID_FS_LABEL"]
	if e.RaidLabel == "" {
		// MD_NAME is "host:name"; dashdot showed the part before the colon.
		e.RaidLabel, _, _ = strings.Cut(udev["MD_NAME"], ":")
	}
	if e.RaidLabel == "" {
		e.RaidLabel = md
	}
	return e
}

// virtualMounts are DASHDOT_FS_VIRTUAL_MOUNTS: filesystems with no single
// disk behind them (mergerfs, ZFS), matched by the mount's source. The
// first disk is the pool itself; the disks under it follow, and come out of
// the layout, so the pool's space isn't counted twice.
func virtualMounts(cfg *config.Config, layout StorageInfo) StorageInfo {
	mounts := readMounts()
	for _, name := range cfg.FSVirtualMounts {
		i := slices.IndexFunc(mounts, func(m mount) bool { return m.source == name })
		if i < 0 {
			continue
		}
		size, _, err := statfs(mounts[i].point)
		if err != nil {
			continue
		}
		e := StorageEntry{Size: size, Virtual: true, Disks: []Disk{{Device: name, Brand: name, Type: "VIRTUAL"}}}
		for _, member := range poolMembers(mounts[i], mounts) {
			j := slices.IndexFunc(layout, func(l StorageEntry) bool {
				return !l.Virtual && slices.ContainsFunc(l.Disks, func(d Disk) bool { return d.Device == member.Device })
			})
			if j >= 0 {
				layout = slices.Delete(layout, j, j+1)
			}
			if !slices.Contains(e.Disks, member) {
				e.Disks = append(e.Disks, member)
			}
		}
		layout = append(layout, e)
	}
	return layout
}

// poolMembers finds the disks under a pool: a mergerfs pool's branches, or
// a ZFS pool's member disks, through each mergerfs branch too.
func poolMembers(pool mount, mounts []mount) []Disk {
	if pool.fstype == "zfs" {
		name, _, _ := strings.Cut(pool.source, "/")
		return zfsMembers(name)
	}
	if !strings.HasPrefix(pool.fstype, "fuse.mergerfs") {
		return nil
	}
	var out []Disk
	for _, branch := range mergerfsBranches(pool) {
		m, ok := mountAt(fromHost(branch), mounts)
		if !ok {
			continue
		}
		if m.fstype == "zfs" {
			name, _, _ := strings.Cut(m.source, "/")
			out = append(out, zfsMembers(name)...)
		} else if d, ok := diskOf(m); ok {
			out = append(out, d)
		}
	}
	return out
}

// mergerfsBranches reads a pool's branches from its control file's xattr
// ("/mnt/a=RW:/mnt/b=NC,100G"), falling back to an fsname of paths
// ("/mnt/a:/mnt/b"), which is mergerfs's default.
func mergerfsBranches(pool mount) []string {
	raw := pool.source
	buf := make([]byte, 64*1024)
	if n, err := unix.Getxattr(filepath.Join(pool.point, ".mergerfs"), "user.mergerfs.branches", buf); err == nil {
		raw = string(buf[:n])
	}
	var out []string
	for _, b := range strings.Split(raw, ":") {
		b, _, _ = strings.Cut(b, "=")
		if filepath.IsAbs(b) {
			out = append(out, b)
		}
	}
	return out
}

// mountAt is the mount a path is on: the longest mount point above it.
func mountAt(path string, mounts []mount) (mount, bool) {
	var best mount
	found := false
	for _, m := range mounts {
		if (path == m.point || strings.HasPrefix(path, strings.TrimSuffix(m.point, "/")+"/")) && len(m.point) >= len(best.point) {
			best, found = m, true
		}
	}
	return best, found
}

// diskOf is the whole disk a mount's device sits on, through partitions,
// md arrays and dm volumes.
func diskOf(m mount) (Disk, bool) {
	name := strings.TrimPrefix(m.source, "/dev/")
	if strings.HasPrefix(m.source, "/dev/mapper/") {
		name = dmName(m.source)
	}
	disks, _ := filepath.Glob(filepath.Join(sysRoot, "block/*"))
	for _, d := range disks {
		disk := filepath.Base(d)
		if skipBlockRe.MatchString(disk) {
			continue
		}
		for _, n := range deviceTree(disk) {
			if n == name || readTrim(filepath.Join(sysBlock(n), "dev")) == m.dev {
				return describeDisk(readBlock(d), d), true
			}
		}
	}
	return Disk{}, false
}

// zfsMembers are the disks with a partition (or whole-disk filesystem)
// udev labels as a member of the named ZFS pool.
func zfsMembers(pool string) []Disk {
	var out []Disk
	disks, _ := filepath.Glob(filepath.Join(sysRoot, "block/*"))
	sort.Strings(disks)
	for _, d := range disks {
		b := readBlock(d)
		if skipBlockRe.MatchString(b.name) {
			continue
		}
		for _, n := range append([]string{b.name}, b.parts...) {
			u := readUdev(readTrim(filepath.Join(sysBlock(n), "dev")))
			if u["ID_FS_TYPE"] == "zfs_member" && u["ID_FS_LABEL"] == pool {
				out = append(out, describeDisk(b, d))
				break
			}
		}
	}
	return out
}

// StorageLoadNow is the bytes used on each StorageStatic entry: the used
// space of every filesystem on the disk, plus whatever of the disk no
// filesystem covers (swap, unpartitioned space). An LVM disk counts only
// what's used, as its free extents would otherwise read as full.
func StorageLoadNow(cfg *config.Config, layout StorageInfo) (StorageLoad, error) {
	all := readMounts()
	mounts := hostMounts(cfg, all)
	out := make(StorageLoad, len(layout))
	for i, e := range layout {
		if e.Virtual {
			// Named by the user, so not type filtered: fuse.mergerfs is in
			// the default DASHDOT_FS_TYPE_FILTER.
			out[i] = 0
			for _, m := range all {
				if len(e.Disks) > 0 && m.source == e.Disks[0].Device {
					if _, used, err := statfs(m.point); err == nil {
						out[i] = float64(used)
					}
					break
				}
			}
			continue
		}

		names := map[string]bool{}
		devs := map[string]bool{}
		lvm := false
		roots := []string{}
		for _, d := range e.Disks {
			roots = append(roots, d.Device)
		}
		if e.RaidName != "" {
			roots = append(roots, e.RaidName)
		}
		for _, r := range roots {
			for _, n := range deviceTree(r) {
				names[n] = true
				devs[readTrim(filepath.Join(sysBlock(n), "dev"))] = true
				if strings.HasPrefix(readTrim(filepath.Join(sysBlock(n), "dm/uuid")), "LVM-") {
					lvm = true
				}
			}
		}

		// One statfs per filesystem: btrfs subvolumes and bind mounts share it.
		var used, total uint64
		seen := map[string]bool{}
		for _, m := range mounts {
			src := strings.TrimPrefix(m.source, "/dev/")
			if strings.HasPrefix(m.source, "/dev/mapper/") {
				src = dmName(m.source)
			}
			if !names[src] && !devs[m.dev] {
				continue
			}
			key := m.dev + "|" + m.source
			if seen[key] {
				continue
			}
			seen[key] = true
			size, u, err := statfs(m.point)
			if err != nil {
				continue
			}
			used += u
			total += size
		}
		switch {
		case len(seen) == 0:
			out[i] = -1
		case lvm || total >= e.Size:
			out[i] = float64(used)
		default:
			out[i] = float64(used + e.Size - total)
		}
	}
	return out, nil
}
