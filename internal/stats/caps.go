package stats

import (
	"runtime"

	"golang.org/x/sys/unix"
)

const (
	capSysPtrace = 19
	capPerfmon   = 38
)

// withCap runs fn with a capability in this thread's effective set, if it's
// permitted. The image gives the binary its caps as permitted only (setcap
// =p), so a non-root container with the matching cap_add can use them, and
// one without still starts. Caps are per thread, so fn runs locked to the
// raised thread, which drops it again.
func withCap(capability int, fn func()) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hdr := unix.CapUserHeader{Version: unix.LINUX_CAPABILITY_VERSION_3}
	var data [2]unix.CapUserData
	if err := unix.Capget(&hdr, &data[0]); err != nil {
		fn()
		return
	}
	set := &data[capability/32]
	bit := uint32(1) << (capability % 32)
	if set.Permitted&bit == 0 || set.Effective&bit != 0 {
		fn()
		return
	}
	set.Effective |= bit
	if err := unix.Capset(&hdr, &data[0]); err != nil {
		fn()
		return
	}
	fn()
	set.Effective &^= bit
	_ = unix.Capset(&hdr, &data[0])
}

// withPerfmon lets fn open Intel's engine counters (cap_add: [PERFMON]).
func withPerfmon(fn func()) { withCap(capPerfmon, fn) }

// withPtrace lets fn read other users' /proc/<pid>/fdinfo
// (cap_add: [SYS_PTRACE]).
func withPtrace(fn func()) { withCap(capSysPtrace, fn) }
