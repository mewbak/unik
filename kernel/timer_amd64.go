// SPDX-License-Identifier: Unlicense OR MIT

package kernel

import (
	"sync/atomic"
	"time"
	"unsafe"
)

const (
	// Assume a fixed address for the HPET.
	// TODO: Detect HPET presence and address from ACPI.
	hpetBase virtualAddress = 0xfed00000
)

const (
	_TN_FSB_INT_DEL_CAP = 1 << 15
	_Tn_FSB_EN_CNF      = 1 << 14
	_Tn_INT_ENB_CNF     = 1 << 2
	_Tn_INT_TYPE_CNF    = 1 << 1
	_Tn_32MODE_CNF      = 1 << 8
	_LEG_ROUTE_CAP      = 1 << 15
	_ENABLE_CNF         = 1 << 0
)

// HPET is the hardware layout of the HPET timer device.
// See the IA-PC HPET specification:
// https://www.intel.com/content/dam/www/public/us/en/documents/technical-specifications/software-developers-hpet-spec-1-0a.pdf
type HPET struct {
	capID   uint64
	_       uint64
	conf    uint64
	_       uint64
	status  uint64
	_       [25]uint64
	counter uint64
	_       uint64
	timers  [32]HPETTimer
}

type HPETTimer struct {
	confCap    uint64
	comparator uint64
	fsbInt     uint64
	_          uint64
}

// A clock keeps track of the current time, and allow multiple
// concurrent readers and a single writer. The writer never blocks
// and the readers are lock-free.
//
// Uses a similar algorithm as the gettimeofday implementation in
// Linux.
//
// Note that vdsoGettimeofday depends on the field offsets.
type clock struct {
	// seq is the sequence number of the clock, as is incremented
	// before and after a write. An odd seq indicates a write is in
	// progress.
	seq uint64

	time instant

	monotoneTime instant
}

type instant struct {
	seconds     int64
	nanoseconds uint32
}

var hpetDev struct {
	device *HPET
	timers []HPETTimer
	period uint32
	last   uint32
	accum  uint64
}

var unixClock clock

//go:nosplit
func initClock() error {
	unixClock.init(readCMOSTime())

	globalIDT.install(intTimer, ring0, istGeneric, timerTrampoline)

	// Map the HPET address range.
	flags := pageFlagWritable | pageFlagNX | pageFlagNoCache
	globalMap.mustAddRange(hpetBase, hpetBase+pageSize, flags)
	if err := mmapAligned(&globalMem, globalPT, hpetBase, hpetBase+pageSize, physicalAddress(hpetBase), flags); err != nil {
		return err
	}
	hpetDev.device = (*HPET)(unsafe.Pointer(hpetBase))
	hpetDev.period = hpetDev.device.clockPeriod()
	hpetDev.timers = hpetDev.device.timers[:hpetDev.device.numTimers()]

	// Sanity checks.

	// The maximum HPET period is 100 ns.
	if hpetDev.period == 0 || hpetDev.period > 1e8 {
		return kernError("setupClock: invalid clock period")
	}

	t := &hpetDev.timers[0]
	if !t.supportsFSB() {
		// TODO: The Qemu HPET device doesn't announce FSB support,
		// but supports it anyway.
	}

	// Configure timer.
	t.fsbInt = uint64(0xfee<<20)<<32 | uint64(intTimer)
	t.confCap = _Tn_FSB_EN_CNF | // Enable FSB.
		_Tn_INT_ENB_CNF | // Enable interrupts.
		_Tn_32MODE_CNF // 32-bit mode.

	// Reset and enable the HPET.
	hpetDev.device.setCounter(0)
	hpetDev.device.enable()
	return nil
}

// clockPeriod returns the clock period in femtoseconds (10⁻¹⁵).
//go:nosplit
func (h *HPET) clockPeriod() uint32 {
	return uint32(h.capID >> 32)
}

// numClocks returns the number of timers.
//go:nosplit
func (h *HPET) numTimers() int {
	return int((h.capID>>8)&0xf) + 1
}

// enable enables the HPET.
//go:nosplit
func (h *HPET) enable() {
	h.conf |= _ENABLE_CNF
}

// disable disables the HPET.
//go:nosplit
func (h *HPET) disable() {
	h.conf &^= _ENABLE_CNF
}

// setCounter sets the hpet counter.
//go:nosplit
func (h *HPET) setCounter(c uint32) {
	atomic.StoreUint64(&hpetDev.device.counter, uint64(c))
}

// readCounter reads the hpet counter.
//go:nosplit
func (h *HPET) readCounter() uint32 {
	return uint32(atomic.LoadUint64(&hpetDev.device.counter))
}

// supportsFSB reports whether the timer supports direct
// interrupt delivery.
//go:nosplit
func (t *HPETTimer) supportsFSB() bool {
	return t.confCap&_TN_FSB_INT_DEL_CAP != 0
}

//go:nosplit
func (t *HPETTimer) oneshot(counter uint32) {
	atomic.StoreUint64(&t.comparator, uint64(counter))
}

//go:nosplit
func updateClock() {
	counter := hpetDev.device.readCounter()
	updateClockWithCounter(counter)
}

func updateClockWithCounter(counter uint32) {
	// Compute the elapsed periods since last interrupt.
	// The subtraction is correct even if the counter wrapped around
	// once.
	fsPrPeriod := uint64(hpetDev.period)
	periods := uint64(counter - hpetDev.last)
	hpetDev.last = counter
	femtos := periods * fsPrPeriod
	acc := hpetDev.accum + femtos
	nanos := acc / 1e6
	hpetDev.accum = acc % 1e6
	unixClock.advance(nanos)
}

//go:nosplit
func setTimer(dur time.Duration) {
	if max := 2 * time.Second; dur > max {
		// Make sure that the current time is updated regularly,
		// and that no time is lost because of the HPET wrapping
		// around.
		dur = max
	}
	fsPrPeriod := uint64(hpetDev.period)
	counter := hpetDev.last
	// Convert to periods.
	schedulePeriods := uint32(uint64(dur) * 1e6 / fsPrPeriod)
	// Schedule next timer interrupt. HPET timers compare with equal,
	// not equal to or larger than, and because we're unwilling to
	// stop the HPET counter, the counter may pass the scheduled time
	// before we set it. Check the counter after arming and try again
	// if so.
	t := &hpetDev.timers[0]
	for {
		end := counter + schedulePeriods
		t.oneshot(end)
		counter = hpetDev.device.readCounter()
		if dur := end - counter; dur < schedulePeriods {
			break
		}
	}
	updateClockWithCounter(counter)
}

//go:nosplit
func (c *clock) init(seconds int64) {
	// Verify field offsets.
	if unsafe.Offsetof(clock{}.seq) != 0 ||
		unsafe.Offsetof(clock{}.time) != 8 ||
		unsafe.Offsetof(clock{}.time.seconds) != 0 ||
		unsafe.Offsetof(clock{}.time.nanoseconds) != 8 {
		fatal("clock.init: unexpected field offset")
	}
	c.time.seconds = seconds
}

//go:nosplit
func (c *clock) advance(nanoseconds uint64) {
	atomic.AddUint64(&c.seq, 1)
	c.time.advance(nanoseconds)
	c.monotoneTime.advance(nanoseconds)
	atomic.AddUint64(&c.seq, 1)
}

//go:nosplit
func (t *instant) advance(nanoseconds uint64) {
	nanoseconds += uint64(t.nanoseconds)
	t.seconds += int64(nanoseconds / 1e9)
	t.nanoseconds = uint32(nanoseconds % 1e9)
}

// monotoneMillis reports the monotone time in milliseconds.
//go:nosplit
func (c *clock) monotoneMillis() uint64 {
	return uint64(c.monotoneTime.seconds)*1e9 + uint64(c.monotoneTime.nanoseconds)
}

func timerTrampoline()
