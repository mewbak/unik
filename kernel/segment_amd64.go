// SPDX-License-Identifier: Unlicense OR MIT

package kernel

import (
	"encoding/binary"
	"unsafe"
)

// Types and code for setting up processor segments and task
// state structure. Segmenting and task switching is largely
// disabled in 64-bit mode, but a GDT and a TSS is nevertheless
// required.

// segmentDescriptor represents a 64-bit segment descriptor.
// Uses uint64 type to force 8-byte alignment.
type segmentDescriptor uint64

// TSS structure for amd64. Hardware task switching is not available
// in 64-bit mode, but a TSS structure must be defined to specify
// interrupt and ring 0 stacks.
type tss [25]uint32

// Global interrupt descriptor table, never touched after
// initialization.
var globalIDT idt

// Golbal task state structure, never touched after initialization.
var globalTSS tss

// The global descriptor table, never touched after initialization.
var globalGDT [segmentEnd]segmentDescriptor

// Interrupt and SYSCALL stack.
var (
	istack         stack
	pageFaultStack stack
)

// Segment selectors. Note that the SYSCALL/SYSRET
// instructions force the particular positions of the selectors.
// See the Intel Architectures Manual Vol 3., 5.8.8 ("Fast System
// Calls in 64-bit Mode"). Additionally, assembly constructs IRETQ
// stack frames with hardcoded segments.
const (
	// Mandatory null selector.
	_ = iota
	// Ring 0 code (64-bit).
	segmentCode0
	// Ring 0 data.
	segmentData0
	// Ring 3 code (32-bit).
	segment32Code3
	// Ring 3 data.
	segmentData3
	// Ring 3 code (64-bit).
	segment64Code3
	// TSS.
	segmentTSS0
	// TSS high address.
	segmentTSS0High
	// End sentinal for determining limit.
	segmentEnd
)

// There are 256 interrupts available.
type idt [256]idtDescriptor

// IDT descriptor. Uses uint64 to force 8-byte alignment.
type idtDescriptor [2]uint64

type segmentFlags uint32
type privLevel uint32
type intVector uint8

const (
	ring0 privLevel = 0
	ring3 privLevel = 3
)

const (
	segFlagAccess  segmentFlags = 1 << 8
	segFlagWrite                = 1 << 9
	segFlagCode                 = 1 << 11
	segFlagSystem               = 1 << 12
	segFlagPresent              = 1 << 15
	segFlagLong                 = 1 << 21
)

const (
	istGeneric = 1
	// Use a separate stack for page faults to handle faults
	// that occur during interrupts.
	istPageFault = 2
)

//go:nosplit
func loadGDT() {
	globalTSS.setISP(istGeneric, uint64(istack.top()))
	globalTSS.setISP(istPageFault, uint64(pageFaultStack.top()))
	globalTSS.setRSP(0, uint64(istack.top()))
	tssAddr := uintptr(unsafe.Pointer(&globalTSS))
	tssLimit := uint32(unsafe.Sizeof(globalTSS) - 1)
	// Block all I/O ports.
	globalTSS.setIOPerm(uint16(tssLimit + 1))
	globalGDT[segmentCode0] = newSegmentDescriptor(0, 0, segFlagSystem|segFlagCode|segFlagLong, ring0)
	globalGDT[segmentData0] = newSegmentDescriptor(0, 0, segFlagSystem|segFlagWrite, ring0)
	globalGDT[segment32Code3] = newSegmentDescriptor(0, 0, segFlagSystem|segFlagCode|segFlagLong, ring3)
	globalGDT[segmentData3] = newSegmentDescriptor(0, 0, segFlagSystem|segFlagWrite, ring3)
	globalGDT[segment64Code3] = newSegmentDescriptor(0, 0, segFlagSystem|segFlagCode|segFlagLong, ring3)
	// The 64-bit TSS structure spans two descriptor entries,
	// with the high 32-bit address in the second entry.
	globalGDT[segmentTSS0] = newSegmentDescriptor(uint32(tssAddr), tssLimit, segFlagAccess|segFlagCode, ring0)
	globalGDT[segmentTSS0High] = segmentDescriptor(tssAddr >> 32)
	// The GDT register is a 10 byte value: a 16-bit limit followed by
	// the 64-bit address.
	var gdtAddr [10]uint8
	addr := uintptr(unsafe.Pointer(&globalGDT))
	// GDT should be 8-byte aligned for best performance.
	if addr%8 != 0 {
		fatal("loadGDT: bad GDT alignment")
	}
	limit := unsafe.Sizeof(globalGDT) - 1
	binary.LittleEndian.PutUint64(gdtAddr[2:], uint64(addr))
	binary.LittleEndian.PutUint16(gdtAddr[:2], uint16(limit))
	lgdt(uint64(uintptr(unsafe.Pointer(&gdtAddr))))
	data0 := uint16(segmentData0<<3 | ring0)
	code0 := uint16(segmentCode0<<3 | ring0)
	tss0 := uint16(segmentTSS0<<3 | ring0)
	setCSReg(code0)
	setSSReg(data0)
	setDSReg(data0)
	setESReg(data0)
	setFSReg(data0)
	setGSReg(data0)
	ltr(tss0)
}

//go:nosplit
func reloadIDT() {
	// The GDT register is a 10 byte value: a 16-bit limit followed by
	// the 64-bit address.
	var idtAddr [10]uint8
	addr := uintptr(unsafe.Pointer(&globalIDT))
	// GDT should be 8-byte aligned for best performance.
	if addr%8 != 0 {
		fatal("reloadIDT: bad GDT alignment")
	}
	limit := unsafe.Sizeof(globalIDT) - 1
	binary.LittleEndian.PutUint64(idtAddr[2:], uint64(addr))
	binary.LittleEndian.PutUint16(idtAddr[:2], uint16(limit))
	lidt(uint64(uintptr(unsafe.Pointer(&idtAddr))))
}

// install an interrupt handler.
//go:nosplit
func (t *idt) install(interrupt intVector, level privLevel, ist uint8, trampoline func()) {
	sel := uint32(segmentCode0<<3 | ring0)
	pc := funcPC(trampoline)
	flags := uint32(segFlagPresent)
	// Use a trap gate, which does not affect the IF flag on entry.
	const trapGate = 0xe
	w0 := sel<<16 | uint32(pc&0xffff)
	w1 := uint32(pc&0xffff0000) | flags | uint32(level)<<13 | trapGate<<8 | uint32(ist)
	w2 := uint32(pc >> 32)
	t[interrupt][0] = uint64(w1)<<32 | uint64(w0)
	t[interrupt][1] = uint64(w2)
}

// setRSP sets the address for the kernel stack
// number idx.
//go:nosplit
func (t *tss) setRSP(idx int, rsp uint64) {
	if idx < 0 || idx > 2 {
		fatal("setRSP: stack index out of range")
	}
	t[1+idx*16] = uint32(rsp)
	t[1+idx*16+1] = uint32(rsp >> 32)
}

// setISP sets the address for the interrupt stack
// number idx (1-based).
//go:nosplit
func (t *tss) setISP(idx int, rsp uint64) {
	if idx < 1 || idx > 7 {
		fatal("setRSP: stack index out of range")
	}
	t[7+idx*2] = uint32(rsp)
	t[7+idx*2+1] = uint32(rsp >> 32)
}

//go:nosplit
func (t *tss) setIOPerm(addr uint16) {
	t[24] = uint32(addr) << 16
}

//go:nosplit
func newSegmentDescriptor(base uint32, limit uint32, flags segmentFlags, level privLevel) segmentDescriptor {
	if limit > 0xfffff {
		fatal("newSegmentDesciptor: limit too high")
	}
	flags |= segFlagPresent
	w0 := base<<16 | limit&0xffff
	w1 := base&0xff000000 | uint32(limit&0xf0000) | uint32(flags) | uint32(level)<<13 | (base>>16)&0xff
	return segmentDescriptor(uint64(w1)<<32 | uint64(w0))
}

func lgdt(addr uint64)
func lidt(addr uint64)
func setCSReg(seg uint16)
func setDSReg(seg uint16)
func setSSReg(seg uint16)
func setESReg(seg uint16)
func setFSReg(seg uint16)
func setGSReg(seg uint16)
func ltr(seg uint16)
