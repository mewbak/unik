// SPDX-License-Identifier: Unlicense OR MIT

package kernel

import (
	"sync/atomic"
	"unsafe"
)

const (
	_IA32_APIC_BASE = 0x1b
)

const (
	intDivideError            intVector = 0x0
	intGeneralProtectionFault intVector = 0xd
	intPageFault              intVector = 0xe
	intSSE                    intVector = 0x13

	apic
)

const (
	firstAvailableInterrupt intVector = 0x20 + iota
	intAPICError
	intTimer
	intFirstUser

	intLastUser           = intFirstUser + 10
	intSpurious intVector = 0xff
)

var pendingInterrupts [intLastUser - intFirstUser]bool

var (
	apicBase virtualAddress
	// Interrupt handlers write 0 to apicEOI to
	// signal end of interrupt handling.
	apicEOI *uint32
)

//go:nosplit
func initAPIC() error {
	_, _, _, edx := cpuid(0x1, 0)
	if edx&(1<<9) == 0 {
		return kernError("initAPIC: no APIC available")
	}
	maskPIC()
	apicBaseMSR := rdmsr(_IA32_APIC_BASE)
	if apicBaseMSR&(1<<8) == 0 {
		return kernError("initAPIC: not running on the boot CPU")
	}
	apicBase = virtualAddress(apicBaseMSR &^ 0xfff)
	// Enable APIC.
	wrmsr(_IA32_APIC_BASE, apicBaseMSR|1<<11)

	apicEOI = (*uint32)(unsafe.Pointer(apicBase + 0xb0))

	// Map the APIC page.
	flags := pageFlagWritable | pageFlagNX | pageFlagNoCache
	globalMap.mustAddRange(apicBase, apicBase+pageSize, flags)
	if err := mmapAligned(&globalMem, globalPT, apicBase, apicBase+pageSize, physicalAddress(apicBase), flags); err != nil {
		return err
	}

	globalIDT.install(intDivideError, ring0, istGeneric, divFault)
	globalIDT.install(intGeneralProtectionFault, ring0, istGeneric, generalProtectionFaultTrampoline)
	globalIDT.install(intSSE, ring0, istGeneric, sseException)
	globalIDT.install(intPageFault, ring0, istPageFault, pageFaultTrampoline)

	globalIDT.install(intAPICError, ring0, istGeneric, unknownInterruptTrampoline)
	globalIDT.install(intSpurious, ring0, istGeneric, unknownInterruptTrampoline)
	installUserHandlers()

	reloadIDT()
	// Mask LINT0-1.
	const apicIntMasked = 1 << 17
	apicWrite(0x350 /* LVT LINT0*/, apicIntMasked)
	apicWrite(0x360 /* LVT LINT1*/, apicIntMasked)
	// Setup error interrupt.
	apicWrite(0x370 /* LVT Error*/, uint32(intAPICError))
	// Setup spurious interrupt handler and enable interrupts.
	apicWrite(0x0f0, 0x100|uint32(intSpurious))

	return nil
}

//go:nosplit
func divFault() {
	fatal("division by 0")
}

//go:nosplit
func gpFault(addr uint64) {
	outputString("fault address: ")
	outputUint64(addr)
	outputString("\n")
	fatal("general protection fault")
}

//go:nosplit
func sseException() {
	fatal("SSE exception")
}

//go:nosplit
func installUserHandlers() {
	vector := intFirstUser
	globalIDT.install(vector, ring0, istGeneric, userInterruptTrampoline0)
	vector++
	globalIDT.install(vector, ring0, istGeneric, userInterruptTrampoline1)
	vector++
	globalIDT.install(vector, ring0, istGeneric, userInterruptTrampoline2)
	vector++
	globalIDT.install(vector, ring0, istGeneric, userInterruptTrampoline3)
	vector++
	globalIDT.install(vector, ring0, istGeneric, userInterruptTrampoline4)
	vector++
	globalIDT.install(vector, ring0, istGeneric, userInterruptTrampoline5)
	vector++
	globalIDT.install(vector, ring0, istGeneric, userInterruptTrampoline6)
	vector++
	globalIDT.install(vector, ring0, istGeneric, userInterruptTrampoline7)
	vector++
	globalIDT.install(vector, ring0, istGeneric, userInterruptTrampoline8)
	vector++
	globalIDT.install(vector, ring0, istGeneric, userInterruptTrampoline9)
	vector++
	if vector != intLastUser {
		fatal("not enough users interrupt handlers declared")
	}
}

//go:nosplit
func apicWrite(reg int, val uint32) {
	atomic.StoreUint32((*uint32)(unsafe.Pointer(apicBase+virtualAddress(reg))), val)
}

//go:nosplit
func apicRead(reg int) uint32 {
	return atomic.LoadUint32((*uint32)(unsafe.Pointer(apicBase + virtualAddress(reg))))
}

//go:nosplit
func maskPIC() {
	const (
		PIC1_DATA = 0x21
		PIC2_DATA = 0xa1
	)
	outb(PIC1_DATA, 0xff)
	outb(PIC2_DATA, 0xff)
}

//go:nosplit
func userInterrupt(vector uint64) {
	pendingInterrupts[vector] = true
}

//go:nosplit
func unknownInterrupt() {
	fatal("unexpected interrupt")
}

func unknownInterruptTrampoline()

func userInterruptTrampoline0()
func userInterruptTrampoline1()
func userInterruptTrampoline2()
func userInterruptTrampoline3()
func userInterruptTrampoline4()
func userInterruptTrampoline5()
func userInterruptTrampoline6()
func userInterruptTrampoline7()
func userInterruptTrampoline8()
func userInterruptTrampoline9()

func generalProtectionFaultTrampoline()
