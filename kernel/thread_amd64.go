// SPDX-License-Identifier: Unlicense OR MIT

package kernel

import (
	"reflect"
	"time"
	"unsafe"
)

const maxThreads = 100

const (
	_IA32_KERNEL_GS_BASE = 0xc0000102
	_IA32_GS_BASE        = 0xc0000101
)

var globalThreads threads

type tid uint64

type threads struct {
	threads []thread
}

// thread represents per-thread context and bookkeeping. Must be
// 8 byte aligned so its fields must have known sizes.
type thread struct {
	self *thread
	context

	id tid

	block blockCondition
}

type blockCondition struct {
	syscall    uint32
	conditions waitConditions

	// For sleepCondition.
	sleep struct {
		monotoneTime uint64
		duration     time.Duration
	}

	// For futexCondition.
	futex uint64

	_ uint32
}

// waitConditions is a set of potential conditions that will wake up a
// thread.
type waitConditions uint32

const (
	interruptCondition waitConditions = 1 << iota
	sleepCondition
	futexCondition
	deadCondition
)

const scheduleTimeSlice = 10 * time.Millisecond

// Thread state for early initialization.
var thread0 thread

// Kernel thread for yielding.
var kernelThread thread

// context represent a thread's CPU state. The exact layout of context
// is known to the thread assembly functions.
type context struct {
	ip    uint64
	sp    uint64
	flags uint64
	bp    uint64
	ax    uint64
	bx    uint64
	cx    uint64
	dx    uint64
	si    uint64
	di    uint64
	r8    uint64
	r9    uint64
	r10   uint64
	r11   uint64
	r12   uint64
	r13   uint64
	r14   uint64
	r15   uint64

	fsbase uint64

	// fpState is space for the floating point context, including
	// alignment. FXSAVE/FXRSTOR needs 512 bytes.
	fpState [512]byte
}

//go:nosplit
func initThreads() error {
	// assembly expects the thread context after self.
	if unsafe.Offsetof(thread{}.self) != 0 {
		fatal("initThreads: invalid thread.self field alignment")
	}
	if unsafe.Offsetof(thread{}.context) != 8 {
		fatal("initThreads: invalid thread.context field alignment")
	}
	if unsafe.Offsetof(thread{}.fpState)%16 != 0 {
		fatal("initThreads: invalid thread.context field alignment")
	}
	return globalThreads.init()
}

//go:nosplit
func (ts *threads) init() error {
	size := unsafe.Sizeof(ts.threads[0]) * maxThreads
	addr, err := globalMap.mmap(0, uint64(size), pageFlagNX|pageFlagWritable)
	if err != nil {
		return err
	}
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&ts.threads))
	hdr.Data = uintptr(addr)
	hdr.Cap = int(size / unsafe.Sizeof(ts.threads[0]))
	return nil
}

//go:nosplit
func (ts *threads) newThread() (*thread, error) {
	if len(ts.threads) == cap(ts.threads) {
		return nil, kernError("newThread: too many threads")
	}
	tid := tid(len(ts.threads))
	ts.threads = ts.threads[:tid+1]
	newt := &ts.threads[tid]
	*newt = thread{
		id: tid,
	}
	newt.self = newt
	return newt, nil
}

// Schedule selects an appropriate thread to resume and makes it
// current.
//go:nosplit
func (ts *threads) schedule(t *thread) {
	for {
		updateClock()
		maxDur := 24 * time.Hour
		monotoneTime := unixClock.monotoneMillis()
		for i := 0; i < len(ts.threads); i++ {
			// Round-robin scheduling.
			tid := (int(t.id) + 1) % len(ts.threads)
			t = &ts.threads[tid]
			if dur, ok := t.runnable(monotoneTime); !ok {
				if dur > 0 && dur < maxDur {
					maxDur = dur
				}
				continue
			}
			t.block.conditions = 0
			t.makeCurrent()
			setTimer(scheduleTimeSlice)
			if t.block.syscall != 0 {
				resumeThreadFast()
			} else {
				resumeThread()
			}
			fatal("schedule: resume failed")
		}
		setTimer(maxDur)
		yield()
	}
}

//go:nosplit
func (ts *threads) futexWakeup(addr uint64, nwaiters int) {
	for i := 0; i < len(ts.threads); i++ {
		if nwaiters == 0 {
			break
		}
		t := &ts.threads[i]
		if t.block.conditions&futexCondition == 0 {
			continue
		}
		if t.block.futex != addr {
			continue
		}
		t.block.conditions = 0
		nwaiters--
	}
}

//go:nosplit
func (t *thread) runnable(monotoneTime uint64) (time.Duration, bool) {
	w := &t.block
	if w.conditions == 0 {
		return 0, true
	}
	cond := w.conditions
	if cond&interruptCondition != 0 {
		for i, intr := range pendingInterrupts {
			if intr {
				pendingInterrupts[i] = false
				t.setSyscallResult(_EOK, uint64(i))
				return 0, true
			}
		}
	}
	if cond&sleepCondition != 0 {
		dur := time.Duration(monotoneTime-t.block.sleep.monotoneTime) * time.Millisecond
		rem := t.block.sleep.duration - dur
		return rem, rem <= 0
	}
	return 0, false
}

//go:nosplit
func (t *thread) setSyscallResult(ret0, ret1 uint64) {
	t.ax = ret0
	t.dx = ret1
}

//go:nosplit
func (t *thread) makeCurrent() {
	v := uint64(uintptr(unsafe.Pointer(t)))
	wrmsr(_IA32_GS_BASE, v)
}

//go:nosplit
func (t *thread) sleepFor(duration time.Duration) {
	t.block.conditions |= sleepCondition
	t.block.sleep.monotoneTime = unixClock.monotoneMillis()
	t.block.sleep.duration = duration
}

//go:nosplit
func (t *thread) dump() {
	fields := []struct {
		field string
		value uint64
	}{
		{"addr", uint64(uintptr(unsafe.Pointer(t)))},
		{"id", uint64(t.id)},
		{"ip", t.ip},
		{"sp", t.sp},
		{"bp", t.bp},
		{"ax", t.ax},
		{"bx", t.bx},
		{"cx", t.cx},
		{"dx", t.dx},
		{"si", t.si},
		{"di", t.di},
		{"r8", t.r8},
		{"r9", t.r9},
		{"r10", t.r10},
		{"r11", t.r11},
		{"r12", t.r12},
		{"r13", t.r13},
		{"r14", t.r14},
		{"r15", t.r15},
		{"fsbase", t.fsbase},
	}
	for _, f := range fields {
		outputString(f.field)
		outputString(": ")
		outputUint64(f.value)
		outputString(" ")
	}
	nstate := len(t.fpState) / int(unsafe.Sizeof(uint64(0)))
	fpState := (*(*[1 << 30]uint64)(unsafe.Pointer(&t.fpState[0])))[:nstate:nstate]
	for i, s := range fpState {
		outputString("f")
		outputUint64(uint64(i))
		outputString(": ")
		outputUint64(s)
		outputString(" ")
	}
	outputString("\n")
}

// interruptSchedule is called to schedule a thread after a
// timer interrupt.
//go:nosplit
func interruptSchedule(t *thread) {
	t.block = blockCondition{}
	globalThreads.schedule(t)
}

// yield halts the processor until an interrupt arrives.
//go:nosplit
func yield() {
	swapgs()
	kernelThread.makeCurrent()
	swapgs()
	yield0()
}

func currentThread() *thread
func swapgs()

// saveThread stores the current thread state except IP, SP, CX, R11,
// and flags.
func saveThread()

// resumeThread restores thread context and resumes the thread. The
// context is stored in GS; see type context for the layout.
// resumeThread never returns.
func resumeThread()

// resumeThreadFast is like resumeThread, but clobbers CX and R11.
func resumeThreadFast()

// restoreThread restores thread context stored by saveThread.
func restoreThread()

func yield0()
