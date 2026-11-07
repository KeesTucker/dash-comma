package stats

import (
	"runtime"

	"golang.org/x/sys/unix"
)

const (
	capNetAdmin  = 12
	capSetpcap   = 8
	capSysPtrace = 19
	capPerfmon   = 38
	capBPF       = 39
)

func capGet() (unix.CapUserHeader, [2]unix.CapUserData, error) {
	hdr := unix.CapUserHeader{Version: unix.LINUX_CAPABILITY_VERSION_3}
	var data [2]unix.CapUserData
	err := unix.Capget(&hdr, &data[0])
	return hdr, data, err
}

// permitted reports whether this thread may raise every one of caps.
func permitted(caps ...int) bool {
	_, data, err := capGet()
	if err != nil {
		return false
	}
	for _, c := range caps {
		if data[c/32].Permitted&(1<<(c%32)) == 0 {
			return false
		}
	}
	return true
}

// withCaps runs fn with caps in this thread's effective set, where they're
// permitted. The image gives the binary its caps as permitted only (setcap
// =p), so a non-root container with the matching cap_add can use them, and
// one without still starts. Caps are per thread, so fn runs locked to the
// raised thread, which drops them again.
func withCaps(fn func(), caps ...int) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hdr, data, err := capGet()
	if err != nil {
		fn()
		return
	}
	before := data
	for _, c := range caps {
		set := &data[c/32]
		if bit := uint32(1) << (c % 32); set.Permitted&bit != 0 {
			set.Effective |= bit
		}
	}
	if data == before || unix.Capset(&hdr, &data[0]) != nil {
		fn()
		return
	}
	fn()
	_ = unix.Capset(&hdr, &before[0])
}

// withPerfmon lets fn open Intel's engine counters (cap_add: [PERFMON]).
func withPerfmon(fn func()) { withCaps(fn, capPerfmon) }

// withPtrace lets fn read other users' /proc/<pid>/fdinfo
// (cap_add: [SYS_PTRACE]).
func withPtrace(fn func()) { withCaps(fn, capSysPtrace) }
