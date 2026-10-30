package stats

import (
	"runtime"

	"golang.org/x/sys/unix"
)

const capPerfmon = 38

// withPerfmon runs fn with CAP_PERFMON in this thread's effective set, if
// it's permitted. The image gives the binary cap_perfmon as permitted only
// (setcap cap_perfmon=p), so a non-root container with cap_add: [PERFMON]
// can read Intel's counters, and one without still starts. Caps are per
// thread, so fn runs locked to the raised thread, which drops it again.
func withPerfmon(fn func()) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hdr := unix.CapUserHeader{Version: unix.LINUX_CAPABILITY_VERSION_3}
	var data [2]unix.CapUserData
	if err := unix.Capget(&hdr, &data[0]); err != nil {
		fn()
		return
	}
	bit := uint32(1) << (capPerfmon - 32)
	if data[1].Permitted&bit == 0 || data[1].Effective&bit != 0 {
		fn()
		return
	}
	data[1].Effective |= bit
	if err := unix.Capset(&hdr, &data[0]); err != nil {
		fn()
		return
	}
	fn()
	data[1].Effective &^= bit
	_ = unix.Capset(&hdr, &data[0])
}
