package stats

import (
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
	"time"
)

func inode(t *testing.T, p string) uint64 {
	t.Helper()
	info, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	return info.Sys().(*syscall.Stat_t).Ino
}

// TestNetGroups rates each cgroup's counters and names the busiest: a
// service by its unit, a container by its biggest process.
func TestNetGroups(t *testing.T) {
	root := useFixtureHost(t, nil)
	cg := filepath.Join(root, "mnt/host/sys/fs/cgroup")
	proc := filepath.Join(root, "mnt/host/proc")
	scope := "system.slice/docker-0123456789abcdef0123.scope"
	fixture(t, cg, map[string]string{
		"cgroup.controllers":                                          "",
		"system.slice/tailscaled.service/cgroup.procs":                "",
		scope + "/cgroup.procs":                                       "",
		"system.slice/docker-fedcba9876543210fedc.scope/cgroup.procs": "",
		"system.slice/docker-dash.scope/cgroup.procs":                 "",
	})
	// dash-comma's own cgroup namespace is its container's cgroup, so
	// /proc/<pid>/cgroup paths climb out of it.
	link(t, filepath.Join(cg, "system.slice/docker-dash.scope"), filepath.Join(root, "sys/fs/cgroup"))
	fixture(t, proc, map[string]string{
		"1/stat":    "1 (init) S 0",
		"60/cgroup": "0::/../../" + scope + "\n",
		"60/comm":   "s6-svscan\n",
		"60/statm":  "100 20 0 0 0 0 0\n",
		"61/cgroup": "0::/../../" + scope + "\n",
		"61/comm":   "Sonarr\n",
		"61/statm":  "9000 4000 0 0 0 0 0\n",
		"70/cgroup": "0::/\n",
		"70/comm":   "dash-comma\n",
		"70/statm":  "99999 99999 0 0 0 0 0\n",
	})
	svc := inode(t, filepath.Join(cg, "system.slice/tailscaled.service"))
	ctr := inode(t, filepath.Join(cg, scope))
	empty := inode(t, filepath.Join(cg, "system.slice/docker-fedcba9876543210fedc.scope"))

	counts := map[uint64][2]uint64{svc: {0, 0}, ctr: {1000, 0}, empty: {0, 0}, 1: {5, 5}}
	clock := time.Unix(0, 0)
	n := NewNetGroups()
	n.now = func() time.Time { return clock }
	n.read = func() map[uint64][2]uint64 {
		out := map[uint64][2]uint64{}
		for k, v := range counts {
			out[k] = v
		}
		return out
	}
	if got, _ := n.Sample(); len(got) != 0 {
		t.Fatalf("first sample: got %+v", got)
	}
	clock = clock.Add(2 * time.Second)
	counts[svc] = [2]uint64{400, 200}
	counts[ctr] = [2]uint64{11000, 2000}
	counts[empty] = [2]uint64{2, 0}
	got, _ := n.Sample()
	want := []NetworkGroup{
		{Name: "Sonarr", Down: 5000, Up: 1000},
		{Name: "tailscaled", Down: 200, Up: 100},
		{Name: "container fedcba987654", Down: 1, Up: 0},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestNetGroupsCounterReset(t *testing.T) {
	counts := map[uint64][2]uint64{7: {500, 500}}
	clock := time.Unix(0, 0)
	n := &NetGroups{now: func() time.Time { return clock }, read: func() map[uint64][2]uint64 { return map[uint64][2]uint64{7: counts[7]} }, names: map[uint64]string{7: "x"}}
	n.Sample()
	clock = clock.Add(time.Second)
	counts[7] = [2]uint64{10, 10}
	if got, _ := n.Sample(); len(got) != 0 {
		t.Fatalf("want an evicted counter skipped, got %+v", got)
	}
}
