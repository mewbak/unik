// SPDX-License-Identifier: Unlicense OR MIT

package kernel

import (
	"time"
	"unsafe"
)

const (
	_MSR_LSTAR = 0xc0000082
	_MSR_STAR  = 0xc0000081
	_MSR_FSTAR = 0xc0000084
)

const (
	// SYSCALL numbers.
	_SYS_write          = 1
	_SYS_mmap           = 9
	_SYS_pipe           = 22
	_SYS_pipe2          = 293
	_SYS_arch_prctl     = 158
	_SYS_uname          = 63
	_SYS_rt_sigaction   = 13
	_SYS_rt_sigprocmask = 14
	_SYS_sigaltstack    = 131
	_SYS_clone          = 56
	_SYS_exit_group     = 231
	_SYS_exit           = 60
	_SYS_nanosleep      = 35
	_SYS_futex          = 202
	_SYS_epoll_create1  = 291
	_SYS_epoll_pwait    = 281
	_SYS_epoll_ctl      = 233

	// Custom syscall numbers.
	_SYS_outl = 0x80000000 + iota
	_SYS_inl
	_SYS_iomap
	_SYS_alloc
	_SYS_waitinterrupt

	_ARCH_SET_FS = 0x1002

	_AT_PAGESZ = 6
	_AT_NULL   = 0

	_MAP_ANONYMOUS = 0x20
	_MAP_PRIVATE   = 0x2
	_MAP_FIXED     = 0x10

	_PROT_WRITE = 0x2
	_PROT_EXEC  = 0x4

	_CLONE_VM      = 0x100
	_CLONE_FS      = 0x200
	_CLONE_FILES   = 0x400
	_CLONE_SIGHAND = 0x800
	_CLONE_SYSVSEM = 0x40000
	_CLONE_THREAD  = 0x10000
)

// Processor flags.
const (
	_FLAG_RESERVED = 1 << 2 // Always set.
	_FLAG_TF       = 1 << 8
	_FLAG_IF       = 1 << 9
	_FLAG_DF       = 1 << 10
	_FLAG_VM       = 1 << 17
	_FLAG_AC       = 1 << 18
)

// Errnos.
const (
	_EOK     = 0
	_ENOTSUP = ^uint64(95) + 1
	_ENOMEM  = ^uint64(0xc) + 1
	_EINVAL  = ^uint64(0x16) + 1
)

const (
	_FUTEX_WAIT         = 0
	_FUTEX_WAKE         = 1
	_FUTEX_PRIVATE_FLAG = 128
	_FUTEX_WAIT_PRIVATE = _FUTEX_WAIT | _FUTEX_PRIVATE_FLAG
	_FUTEX_WAKE_PRIVATE = _FUTEX_WAKE | _FUTEX_PRIVATE_FLAG
)

type timespec struct {
	seconds     int64
	nanoseconds int32
}

//go:nosplit
func initSYSCALL() {
	// Setup segments for SYSCALL/SYSRET.
	syscallSeg := uint64(segmentCode0<<3 | ring0)
	sysretSeg := uint64(segment32Code3<<3 | ring3)
	wrmsr(_MSR_STAR, uint64(uint64(syscallSeg)<<32|uint64(sysretSeg)<<48))
	// Clear flags on entry to SYSCALL handler.
	wrmsr(_MSR_FSTAR, _FLAG_IF|_FLAG_TF|_FLAG_AC|_FLAG_VM|_FLAG_TF|_FLAG_DF)
	// Setup SYSCALL handler.
	wrmsr(_MSR_LSTAR, uint64(funcPC(syscallTrampoline)))

	// Enable SYSCALL instruction.
	efer := rdmsr(_MSR_IA32_EFER)
	wrmsr(_MSR_IA32_EFER, efer|_EFER_SCE)
}

//go:nosplit
func sysenter(t *thread, sysno, a0, a1, a2, a3, a4, a5 uint64) {
	t.block = blockCondition{
		syscall: 1,
	}
	ret0, ret1 := sysenter0(t, sysno, a0, a1, a2, a3, a4, a5)
	// Return values are passed in AX, DX.
	t.setSyscallResult(ret0, ret1)
	if t.block.conditions == 0 {
		resumeThreadFast()
	} else {
		globalThreads.schedule(t)
	}
	fatal("sysenter: resume failed")
}

//go:nosplit
func (ts *timespec) duration() (time.Duration, bool) {
	if ts == nil || ts.seconds < 0 {
		return 0, false
	}
	dur := time.Duration(ts.seconds)*time.Second + time.Duration(ts.nanoseconds)*time.Nanosecond
	return dur, true
}

//go:nosplit
func sysenter0(t *thread, sysno, a0, a1, a2, a3, a4, a5 uint64) (uint64, uint64) {
	switch sysno {
	case _SYS_write:
		fd := a0
		p := virtualAddress(a1)
		n := uint32(a2)
		const dummyFd = 0
		if fd == dummyFd {
			return uint64(n), 0
		}
		if fd != 1 && fd != 2 {
			return _ENOTSUP, 0
		}
		bytes := sliceForMem(p, int(n))
		output(bytes)
		return uint64(len(bytes)), 0
	case _SYS_mmap:
		addr := virtualAddress(a0)
		n := a1
		// prot := a2
		flags := a3
		// fd := a4
		// off := a5
		supported := _MAP_ANONYMOUS | _MAP_PRIVATE | _MAP_FIXED
		if flags & ^uint64(supported) != 0 {
			return _ENOTSUP, 0
		}
		/*var pf pageFlags
		if prot&_PROT_WRITE != 0 {
			pf |= pageWritable
		}
		if prot&_PROT_EXEC == 0 {
			pf |= pageNotExecutable
		}*/
		// Always use the most lenient flags for now.
		pf := pageFlagWritable | pageFlagUserAccess
		if flags&_MAP_FIXED != 0 {
			if !globalMap.mmapFixed(addr, n, pf) {
				// Ignore error and assume the range is
				// already mapped.
			}
			return uint64(addr), 0
		} else {
			addr, err := globalMap.mmap(addr, n, pf)
			if err != nil {
				return _ENOMEM, 0
			}
			return uint64(addr), 0
		}
	case _SYS_clone:
		flags := a0
		// Support only the particular set of flags used by Go.
		const expFlags = _CLONE_VM |
			_CLONE_FS |
			_CLONE_FILES |
			_CLONE_SIGHAND |
			_CLONE_SYSVSEM |
			_CLONE_THREAD
		if flags != expFlags {
			return _ENOTSUP, 0
		}
		stack := a1
		clone, err := globalThreads.newThread()
		if err != nil {
			return _ENOMEM, 0
		}
		clone.context = t.context
		clone.sp = stack
		clone.ax = 0 // Return 0 from the cloned thread.
		return uint64(clone.id), 0
	case _SYS_exit_group:
		t.block.conditions = deadCondition
		return _EOK, 0
	case _SYS_arch_prctl:
		switch code := a0; code {
		case _ARCH_SET_FS:
			addr := a1
			t.fsbase = addr
			wrmsr(_MSR_FS_BASE, addr)
			return _EOK, 0
		}
	case _SYS_uname:
		// Ignore for now; the Go runtime only uses uname to detect buggy
		// Linux kernel versions.
		return _EOK, 0
	case _SYS_futex:
		addr := a0
		val := a2
		switch op := a1; op {
		case _FUTEX_WAIT, _FUTEX_WAIT_PRIVATE:
			ts := (*timespec)(unsafe.Pointer(uintptr(a3)))
			if d, ok := ts.duration(); ok {
				t.sleepFor(d)
			}
			t.block.conditions |= futexCondition
			t.block.futex = addr
			return 0, 0
		case _FUTEX_WAKE, _FUTEX_WAKE_PRIVATE:
			globalThreads.futexWakeup(addr, int(val))
			return _EOK, 0
		}
	case _SYS_rt_sigprocmask, _SYS_sigaltstack, _SYS_rt_sigaction:
		// Ignore signals.
		return _EOK, 0
	case _SYS_nanosleep:
		ts := (*timespec)(unsafe.Pointer(uintptr(a0)))
		if d, ok := ts.duration(); ok {
			t.sleepFor(d)
		}
		return _EOK, 0
	case _SYS_epoll_create1:
		return _EOK, 0
	case _SYS_epoll_ctl:
		return _EOK, 0
	case _SYS_pipe2:
		return _EOK, 0
	case _SYS_epoll_pwait:
		timeout := time.Duration(a3) * time.Millisecond
		if timeout >= 0 {
			t.sleepFor(timeout)
		} else {
			t.block.conditions = deadCondition
		}
		return _EOK, 0
	case _SYS_outl:
		port := uint16(a0)
		val := uint32(a1)
		outl(port, val)
		return _EOK, 0
	case _SYS_inl:
		port := uint16(a0)
		val := inl(port)
		return uint64(val), 0
	case _SYS_iomap:
		vaddr := virtualAddress(a0)
		addr := physicalAddress(a1)
		size := a2
		if vaddr&(pageSize-1) != 0 {
			return _EINVAL, 0
		}
		if addr&(pageSize-1) != 0 {
			return _EINVAL, 0
		}
		size = (size + pageSize - 1) &^ (pageSize - 1)
		r, ok := globalMap.rangeForAddress(vaddr, int(size))
		if !ok {
			return _EINVAL, 0
		}
		err := mmapAligned(&globalMem, globalPT, vaddr, vaddr+virtualAddress(size), addr, r.flags)
		if err != nil {
			// TODO: free virtual map.
			return _ENOMEM, 0
		}
		return _EOK, 0
	case _SYS_alloc:
		maxSize := a0
		addr, size, err := globalMem.alloc(int(maxSize))
		if err != nil {
			return _ENOMEM, 0
		}
		return uint64(addr), uint64(size)
	case _SYS_waitinterrupt:
		t.block.conditions = interruptCondition
		return 0, 0
	}
	return _ENOTSUP, 0
}

const COM1 = 0x3f8

//go:nosplit
func output(b []byte) {
	for i := 0; i < len(b); i++ {
		outb(COM1, b[i])
	}
}

//go:nosplit
func outputString(b string) {
	for i := 0; i < len(b); i++ {
		outb(COM1, b[i])
	}
}

//go:nosplit
func outputUint64(v uint64) {
	onlyZero := true
	outputString("0x")
	for i := 15; i >= 0; i-- {
		// Extract the ith nibble.
		nib := byte((v >> (i * 4)) & 0xf)
		if onlyZero && i > 0 && nib == 0 {
			// Skip leading zeros.
			continue
		}
		onlyZero = false
		switch {
		case 0 <= nib && nib <= 9:
			outb(COM1, nib+'0')
		default:
			outb(COM1, nib-10+'a')
		}
	}
}

func syscallTrampoline()
