package stats

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/KeesTucker/dash-comma/internal/config"
)

// fixture writes files under root, creating directories as needed.
func fixture(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for p, content := range files {
		full := filepath.Join(root, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func link(t *testing.T, target, name string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, name); err != nil {
		t.Fatal(err)
	}
}

// useFixtureHost points procRoot, sysRoot and HostRoot at a temp dir, as
// in Docker, and fakes statfs from sizes (size and used, by mount point,
// relative to the temp dir).
func useFixtureHost(t *testing.T, sizes map[string][2]uint64) string {
	t.Helper()
	root := t.TempDir()
	oldProc, oldSys, oldHost, oldStatfs := procRoot, sysRoot, HostRoot, statfs
	procRoot, sysRoot, HostRoot = filepath.Join(root, "proc"), filepath.Join(root, "sys"), filepath.Join(root, "mnt/host")
	statfs = func(p string) (uint64, uint64, error) {
		rel, _ := filepath.Rel(root, p)
		s, ok := sizes[rel]
		if !ok {
			return 0, 0, os.ErrNotExist
		}
		return s[0], s[1], nil
	}
	t.Cleanup(func() { procRoot, sysRoot, HostRoot, statfs = oldProc, oldSys, oldHost, oldStatfs })
	return root
}

// blockDev lays out a disk the way sysfs does: /sys/block/<name> links to
// the device directory, which /sys/class/block/<name> also links to.
func blockDev(t *testing.T, root, name, dev string, sectors string, files map[string]string) {
	dir := filepath.Join(root, "sys/devices/virtual/block", name)
	all := map[string]string{"dev": dev, "size": sectors}
	for k, v := range files {
		all[k] = v
	}
	fixture(t, dir, all)
	link(t, dir, filepath.Join(root, "sys/block", name))
	link(t, dir, filepath.Join(root, "sys/class/block", name))
}

func part(t *testing.T, root, disk, name, dev string) {
	dir := filepath.Join(root, "sys/devices/virtual/block", disk, name)
	fixture(t, dir, map[string]string{"dev": dev, "partition": "1", "size": "100"})
	link(t, dir, filepath.Join(root, "sys/class/block", name))
}

func holder(t *testing.T, root, member, md string) {
	if err := os.MkdirAll(filepath.Join(root, "sys/class/block", member, "holders", md), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestStorage(t *testing.T) {
	const tb = uint64(1_000_000_000_000)
	root := useFixtureHost(t, map[string][2]uint64{
		"mnt/host":              {400_000_000_000, 100_000_000_000},
		"mnt/host/data":         {2 * tb, tb},
		"mnt/host/srv/stinker1": {10 * tb, 3 * tb},
		"mnt/host/mnt/pool":     {8 * tb, 5 * tb},
		"mnt/host/mnt/d1":       {4 * tb, 2 * tb},
		"mnt/host/mnt/d2":       {4 * tb, 3 * tb},
	})

	// sda: the system disk, 500 GB with a 400 GB root partition.
	blockDev(t, root, "sda", "8:0", "976562500", map[string]string{"queue/rotational": "0", "device/model": "Samsung SSD 870 EVO 500GB", "device/vendor": "ATA"})
	part(t, root, "sda", "sda1", "8:1")
	// sdb and sdc: a RAID1 of two 2 TB disks, mounted at /data.
	blockDev(t, root, "sdb", "8:16", "3906250000", map[string]string{"queue/rotational": "1", "device/model": "WDC WD20EFRX-68E", "device/vendor": "ATA"})
	part(t, root, "sdb", "sdb1", "8:17")
	holder(t, root, "sdb1", "md0")
	blockDev(t, root, "sdc", "8:32", "3906250000", map[string]string{"queue/rotational": "1", "device/model": "ST2000VN004", "device/vendor": "ATA"})
	part(t, root, "sdc", "sdc1", "8:33")
	holder(t, root, "sdc1", "md0")
	blockDev(t, root, "md0", "9:0", "3906000000", map[string]string{"md/level": "raid1"})
	// sdd: a disk with nothing mounted. loop0 and zram0 are skipped.
	blockDev(t, root, "sdd", "8:48", "1000", map[string]string{"device/model": "Crucial CT1000"})
	blockDev(t, root, "loop0", "7:0", "1000", nil)
	blockDev(t, root, "zram0", "252:0", "1000", nil)

	// sde and sdf: under a mergerfs pool, sde as ext4 (and hidden by
	// DASHDOT_FS_DEVICE_FILTER, as people did with dashdot) and sdf in a ZFS
	// pool, tank.
	blockDev(t, root, "sde", "8:64", "7812500000", map[string]string{"queue/rotational": "1", "device/model": "ST4000VN008"})
	part(t, root, "sde", "sde1", "8:65")
	blockDev(t, root, "sdf", "8:80", "7812500000", map[string]string{"queue/rotational": "1", "device/model": "ST4000VN008"})
	part(t, root, "sdf", "sdf1", "8:81")

	fixture(t, root, map[string]string{
		"mnt/host/run/udev/data/b9:0":  "E:ID_FS_TYPE=ext4\nE:MD_NAME=nas:data\n",
		"mnt/host/run/udev/data/b8:81": "E:ID_FS_TYPE=zfs_member\nE:ID_FS_LABEL=tank\n",
		"proc/self/mountinfo": "" +
			"1 0 0:50 / / rw - overlay overlay rw\n" +
			"2 1 8:1 / " + root + "/mnt/host ro - ext4 /dev/sda1 rw\n" +
			"3 2 9:0 / " + root + "/mnt/host/data ro - ext4 /dev/md0 rw\n" +
			"4 2 0:60 / " + root + "/mnt/host/srv/stinker1 ro - zfs stinker1 rw\n" +
			"5 2 0:61 / " + root + "/mnt/host/mnt/nas ro - nfs4 nas:/share rw\n" +
			"6 1 8:1 /var/lib/docker " + root + "/etc/hosts rw - ext4 /dev/sda1 rw\n" +
			"7 2 8:65 / " + root + "/mnt/host/mnt/d1 ro - ext4 /dev/sde1 rw\n" +
			"8 2 0:62 / " + root + "/mnt/host/mnt/d2 ro - zfs tank/media rw\n" +
			"9 2 0:63 / " + root + "/mnt/host/mnt/pool ro - fuse.mergerfs /mnt/d1:/mnt/d2 rw\n",
	})

	env := map[string]string{"DASHDOT_FS_VIRTUAL_MOUNTS": "stinker1,missing,/mnt/d1:/mnt/d2", "DASHDOT_FS_DEVICE_FILTER": "sde"}
	cfg := config.From(func(k string) (string, bool) {
		v, ok := env[k]
		return v, ok
	})
	layout, err := StorageStatic(cfg)
	if err != nil {
		t.Fatal(err)
	}
	one := RaidOne
	want := StorageInfo{
		{Size: 500_000_000_000, Disks: []Disk{{Device: "sda", Brand: "Samsung", Type: "SSD"}}},
		{RaidName: "md0", RaidLabel: "nas", RaidType: &one, Size: 3906000000 * 512, Disks: []Disk{
			{Device: "sdb", Brand: "Western Digital", Type: "HD"},
			{Device: "sdc", Brand: "Seagate", Type: "HD"},
		}},
		{Size: 512000, Disks: []Disk{{Device: "sdd", Brand: "Crucial", Type: "SSD"}}},
		{Size: 10 * tb, Virtual: true, Disks: []Disk{{Device: "stinker1", Brand: "stinker1", Type: "VIRTUAL"}}},
		// sdf comes out of the layout into the pool; sde was filtered out but
		// still shows as a member.
		{Size: 8 * tb, Virtual: true, Disks: []Disk{
			{Device: "/mnt/d1:/mnt/d2", Brand: "/mnt/d1:/mnt/d2", Type: "VIRTUAL"},
			{Device: "sde", Brand: "Seagate", Type: "HD"},
			{Device: "sdf", Brand: "Seagate", Type: "HD"},
		}},
	}
	if !reflect.DeepEqual(layout, want) {
		t.Fatalf("layout:\n got %+v\nwant %+v", layout, want)
	}

	load, err := StorageLoadNow(cfg, layout)
	if err != nil {
		t.Fatal(err)
	}
	// sda: 100 GB used plus the 100 GB no filesystem covers; the bind mount
	// of sda1 outside HostRoot doesn't count twice. md0: what /data uses.
	// The pool reads its own usage, though fuse.mergerfs is type filtered.
	wantLoad := StorageLoad{200_000_000_000, float64(tb), -1, float64(3 * tb), float64(5 * tb)}
	if !reflect.DeepEqual(load, wantLoad) {
		t.Fatalf("load: got %v, want %v", load, wantLoad)
	}
}

func TestUnescapeMount(t *testing.T) {
	if got := unescapeMount(`/mnt/my\040disk`); got != "/mnt/my disk" {
		t.Fatalf("got %q", got)
	}
}

func TestDiskBrand(t *testing.T) {
	cases := map[[2]string]string{
		{"WDC WD40EFRX-68N32N0", "ATA"}: "Western Digital",
		{"ST4000VN008-2DR166", "ATA"}:   "Seagate",
		{"QEMU HARDDISK", "QEMU"}:       "QEMU",
		{"Mystery", "ATA"}:              "sdz",
	}
	for in, want := range cases {
		if got := diskBrand(in[0], in[1], "sdz"); got != want {
			t.Errorf("diskBrand(%q, %q) = %q, want %q", in[0], in[1], got, want)
		}
	}
}
