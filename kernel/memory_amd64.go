// SPDX-License-Identifier: Unlicense OR MIT

package kernel

import (
	"math/bits"
	"reflect"
	"unsafe"
)

const (
	_EFI_MEMORY_RUNTIME = 0x8000000000000000
)

const (
	// Page sizes
	pageSize    = 1 << 12
	pageSize2MB = 1 << 21
	pageSize1GB = 1 << 30

	pageSizeRoot = 1 << 39

	pageTableSize = 512
)

// physicalMapOffset is the offset at which the physical memory
// is identity mapped. It is 0 until after initPageTables.
var physicalMapOffset virtualAddress

// The maximum physical address addressable by the processor.
const _MAXPHYADDR physicalAddress = 1 << 52

// The maximum virtual address.
var maxVirtAddress virtualAddress

type efiMemoryMap struct {
	mmap   []byte
	stride int
}

type elfImage struct {
	phdr                []byte
	phdrSize, phdrCount int
}

type elfSegmentHeader struct {
	pType   uint32
	pFlags  uint32
	pOffset uint64
	pVaddr  uint64
	pPaddr  uint64
	pFilesz uint64
	pMemsz  uint64
	pAlign  uint64
}

// efiMemoryDescriptor is a version 1 EFI_MEMORY_DESCRIPTOR.
type efiMemoryDescriptor struct {
	_type         efiMemoryType
	physicalStart physicalAddress
	virtualStart  virtualAddress
	numberOfPages uint64
	attribute     uint64
}

type efiMemoryType uint32

type pageFlags uint64

type physicalAddress uintptr

type virtualAddress uintptr

// pageTable is the hardware representation of a 4-level page table.
type pageTable [pageTableSize]pageTableEntry

// pageTableEntry is the hardware representation of a page table
// entry.
type pageTableEntry uint64

type pageVisitor func(addr virtualAddress, entry *pageTableEntry)

// memory is a simple allocator for physical memory, tracking free
// pages with a bitmap.
type memory struct {
	start physicalAddress
	// The index into bits of the last allocated block.
	word int
	// bits represent each physical memory page with one bit. 1
	// mean free, 0 means allocated or reserved.
	bits []uint64
}

// virtMemory tracks the all reserved virtual memory ranges
// and their flags.
type virtMemory struct {
	// ranges is the list of memory ranges, sorted by range.
	ranges []memoryRange
	// next is the address to start searching for a free range.
	next virtualAddress
}

type memoryRange struct {
	start virtualAddress
	end   virtualAddress
	flags pageFlags
}

var (
	hugePage1GBSupport = false
	nxSupport          = false
)

var (
	globalMem memory
	globalPT  *pageTable
	globalMap virtMemory
)

const (
	pageFlagPresent    pageFlags = 1 << 0
	pageFlagWritable   pageFlags = 1 << 1
	pageFlagNX         pageFlags = 1 << 63
	pageFlagUserAccess pageFlags = 1 << 2
	pageFlagNoCache    pageFlags = 1 << 4
	allPageFlags                 = pageFlagPresent | pageFlagWritable | pageFlagNX | pageFlagUserAccess | pageFlagNoCache

	pageSizeFlag pageFlags = 1 << 7
)

const (
	efiLoaderCode          efiMemoryType = 1
	efiLoaderData          efiMemoryType = 2
	efiBootServicesCode    efiMemoryType = 3
	efiBootServicesData    efiMemoryType = 4
	efiRuntimeServicesCode efiMemoryType = 5
	efiRuntimeServicesData efiMemoryType = 6
	efiConventionalMemory  efiMemoryType = 7
)

const (
	_ELFMagic = 0x464C457F
	_PT_LOAD  = 1
)

const virtMapSize = 1 << 30

//go:nosplit
func newELFImage(img []byte) (elfImage, error) {
	magic := *(*uint32)(unsafe.Pointer(&img[0]))
	if magic != _ELFMagic {
		return elfImage{}, kernError("kernel: invalid ELF image magic")
	}
	phdrOff := *(*uint64)(unsafe.Pointer(&img[32]))
	phdrSize := *(*uint16)(unsafe.Pointer(&img[54]))
	phdrCount := *(*uint16)(unsafe.Pointer(&img[56]))
	phdr := img[phdrOff : phdrSize*phdrCount]
	return elfImage{phdr: phdr, phdrSize: int(phdrSize), phdrCount: int(phdrCount)}, nil
}

//go:nosplit
func initMemory(efiMap efiMemoryMap, kernelImage []byte) error {
	if err := setupPageTable(efiMap, kernelImage); err != nil {
		return err
	}
	// Hold the virtual memory map structure and the physical identity
	// map in the upper half of the virtual address space.
	virtMapStart := maxVirtAddress >> 1
	// Addresses must be sign extended (in canonical form).
	virtMapStart |= ^(maxVirtAddress - 1)
	// Allocate 1 GB of virtual memory for the virtual memory ranges.
	virtMapEnd := virtMapStart + virtMapSize
	// Identity map physical memory in the upper half of the virtual
	// memory space.
	if err := identityMapMem(&globalMem, globalPT, efiMap, virtMapEnd); err != nil {
		return err
	}
	if err := identityMapKernel(&globalMem, globalPT, kernelImage); err != nil {
		return err
	}
	physicalMapOffset = virtMapEnd
	switchMemoryMap(&efiMap, &kernelImage)

	// Initialize virtual memory map.
	vmap, err := newVirtMemory(&globalMem, globalPT, virtMapStart, virtMapSize)
	if err != nil {
		return err
	}
	if err := addKernelRanges(&vmap, kernelImage); err != nil {
		return err
	}
	// Reserve the upper half of the virtual memory, up until the vDSO
	// starting address.
	vmap.mustAddRange(physicalMapOffset, vdsoAddress, pageFlagWritable|pageFlagNX)
	if err := mapReservedMem(&globalMem, globalPT, &vmap, efiMap); err != nil {
		return err
	}
	freeLoaderMem(&globalMem, efiMap)
	globalMap = vmap
	return nil
}

//go:nosplit
func switchMemoryMap(efiMap *efiMemoryMap, kernelImage *[]byte) {
	// Activate new memory map.
	setCR3Reg(uintptr(unsafe.Pointer(globalPT)))
	// Offset pointers allocated with 0-based offsets.
	*(*uintptr)(unsafe.Pointer(&globalPT)) += uintptr(physicalMapOffset)
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&globalMem.bits))
	hdr.Data += uintptr(physicalMapOffset)
	hdr = (*reflect.SliceHeader)(unsafe.Pointer(&efiMap.mmap))
	hdr.Data += uintptr(physicalMapOffset)
	hdr = (*reflect.SliceHeader)(unsafe.Pointer(kernelImage))
	hdr.Data += uintptr(physicalMapOffset)
}

//go:nosplit
func setupPageTable(efiMap efiMemoryMap, kernelImage []byte) error {
	initPagingFeatures()
	if nxSupport {
		// Enable no-execute bit.
		efer := rdmsr(_MSR_IA32_EFER)
		wrmsr(_MSR_IA32_EFER, efer|_EFER_NXE)
	}

	if err := initMemBitmap(&globalMem, efiMap); err != nil {
		return err
	}
	if err := reserveImageMem(&globalMem, kernelImage); err != nil {
		return err
	}
	page, _, err := globalMem.alloc(pageSize)
	if err != nil {
		return err
	}
	globalPT = (*pageTable)(unsafe.Pointer(physToVirt(page)))
	return nil
}

//go:nosplit
func newVirtMemory(mem *memory, pt *pageTable, start virtualAddress, size uint64) (virtMemory, error) {
	var vm virtMemory
	// Leave the lowest addresses unmapped.
	vm.next = 0x100000
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&vm.ranges))
	hdr.Data = uintptr(start)
	hdr.Cap = int(uintptr(size) / unsafe.Sizeof(vm.ranges[0]))
	// Eagerly allocate the first page to fit its own mapping. The
	// rest is faulted in.
	addr, _, err := mem.alloc(pageSize)
	if err != nil {
		return virtMemory{}, err
	}
	flags := pageFlagWritable | pageFlagNX
	if err := mmapAligned(mem, pt, start, start+pageSize, addr, flags); err != nil {
		return virtMemory{}, err
	}
	// Add vm's own address range.
	vm.mustAddRange(start, start+virtualAddress(size), flags)
	return vm, nil
}

// faultPage is called from the page fault interrupt handler.
//go:nosplit
func faultPage(addr virtualAddress) error {
	addr = addr & ^virtualAddress(pageSize-1)
	r, ok := globalMap.rangeForAddress(addr, pageSize)
	if !ok {
		return kernError("faultPage: page fault for unmapped address")
	}
	flags := r.flags
	if flags == pageFlagNX {
		return kernError("faultPage: page fault for PROT_NONE address")
	}
	paddr, _, err := globalMem.alloc(pageSize)
	if err != nil {
		return err
	}
	return mmapAligned(&globalMem, globalPT, addr, addr+pageSize, paddr, flags)
}

// identityMapMem makes the physical memory directly addressable for
// purposes such as page tables.
//go:nosplit
func identityMapMem(mem *memory, pt *pageTable, efiMap efiMemoryMap, offset virtualAddress) error {
	start := ^physicalAddress(0)
	end := physicalAddress(0)
	for i := 0; i < efiMap.len(); i++ {
		desc := efiMap.entry(i)
		if !desc.isUsable() {
			continue
		}
		if desc.physicalStart < start {
			start = desc.physicalStart
		}
		dend := desc.physicalStart + physicalAddress(desc.numberOfPages*pageSize)
		if dend > end {
			end = dend
		}
	}
	if start > end {
		return kernError("identityMapMem: start > end")
	}
	size := uint64(end - start)
	vaddr := offset + virtualAddress(start)
	return mmapAligned(mem, pt, vaddr, vaddr+virtualAddress(size), start, pageFlagWritable|pageFlagNX)
}

//go:nosplit
func freeLoaderMem(mem *memory, efiMap efiMemoryMap) {
	for i := 0; i < efiMap.len(); i++ {
		desc := efiMap.entry(i)
		if desc._type == efiLoaderData {
			start := desc.physicalStart
			end := start + physicalAddress(desc.numberOfPages*pageSize)
			mem.setFree(true, start, end)
		}
	}
}

//go:nosplit
func mapReservedMem(mem *memory, pt *pageTable, vmap *virtMemory, efiMap efiMemoryMap) error {
	for i := 0; i < efiMap.len(); i++ {
		desc := efiMap.entry(i)
		if !desc.isRuntime() {
			continue
		}
		// Identity map UEFI runtime addresses.
		addr := desc.physicalStart
		vaddr := virtualAddress(addr)
		end := vaddr + virtualAddress(desc.numberOfPages*pageSize)
		flags := pageFlagWritable
		vmap.mustAddRange(vaddr, end, flags)
		if err := mmapAligned(mem, pt, vaddr, end, addr, flags); err != nil {
			return err
		}
	}
	return nil
}

//go:nosplit
func addKernelRanges(vmap *virtMemory, image []byte) error {
	elfImg, err := newELFImage(image)
	if err != nil {
		return err
	}
	for i := 0; i < elfImg.phdrCount; i++ {
		seg := elfImg.readSegHeader(i)
		if seg.pType != _PT_LOAD {
			continue
		}
		start := seg.start()
		end := seg.end()
		flags := seg.flags() | pageFlagUserAccess
		vmap.mustAddRange(start, end, flags)
	}
	return nil
}

//go:nosplit
func identityMapKernel(mem *memory, pt *pageTable, image []byte) error {
	elfImg, err := newELFImage(image)
	if err != nil {
		return err
	}
	mmapAligned := mmapAligned // Cheat the nosplit checks.
	for i := 0; i < elfImg.phdrCount; i++ {
		seg := elfImg.readSegHeader(i)
		if seg.pType != _PT_LOAD {
			continue
		}
		start := seg.start()
		end := seg.end()
		flags := seg.flags() | pageFlagUserAccess
		err = mmapAligned(mem, pt, start, end, physicalAddress(start), flags)
		if err != nil {
			return err
		}
	}
	return nil
}

//go:nosplit
func (e *elfImage) readSegHeader(idx int) *elfSegmentHeader {
	off := idx * e.phdrSize
	hdr := e.phdr[off : off+int(unsafe.Sizeof(elfSegmentHeader{}))]
	return (*elfSegmentHeader)(unsafe.Pointer(&hdr[0]))
}

//go:nosplit
func reserveImageMem(mem *memory, image []byte) error {
	elfImg, err := newELFImage(image)
	if err != nil {
		return err
	}
	for i := 0; i < elfImg.phdrCount; i++ {
		seg := elfImg.readSegHeader(i)
		if seg.pType != _PT_LOAD {
			continue
		}
		mem.setFree(false, physicalAddress(seg.start()), physicalAddress(seg.end()))
	}
	return nil
}

//go:nosplit
func (e *elfSegmentHeader) start() virtualAddress {
	return virtualAddress(e.pVaddr)
}

//go:nosplit
func (e *elfSegmentHeader) end() virtualAddress {
	sz := (e.pMemsz + e.pAlign - 1) &^ uint64(e.pAlign-1)
	return e.start() + virtualAddress(sz)
}

//go:nosplit
func (e *elfSegmentHeader) flags() pageFlags {
	flags := pageFlagNX
	const (
		PF_X = 0x1
		PF_W = 0x2
	)
	if e.pFlags&PF_X != 0 {
		flags &^= pageFlagNX
	}
	if e.pFlags&PF_W != 0 {
		flags |= pageFlagWritable
	}
	return flags
}

// initMem initializes a memory allocator from an EFI memory map.
//go:nosplit
func initMemBitmap(mem *memory, efiMap efiMemoryMap) error {
	// Determine the highest usable physical memory address and
	// largest memory region.
	var maxAddr physicalAddress
	minAddr := ^physicalAddress(0)
	var largestDesc *efiMemoryDescriptor
	for i := 0; i < efiMap.len(); i++ {
		desc := efiMap.entry(i)
		if !desc.isUsable() {
			continue
		}
		min := desc.physicalStart
		max := min + physicalAddress(desc.numberOfPages*pageSize)
		if min < minAddr {
			minAddr = min
		}
		if max > maxAddr {
			maxAddr = max
		}
		// The EFI memory map is itself located in an
		// EFILoaderData region. Don't re-use it before we're
		// done with it.
		if desc._type == efiLoaderData {
			continue
		}
		if largestDesc == nil || desc.numberOfPages > largestDesc.numberOfPages {
			largestDesc = desc
		}
	}
	if largestDesc == nil {
		return kernError("initMem: no initial memory")
	}
	// Compute the number of pages the memory bitmap takes up.
	rng := uint64(maxAddr - minAddr)
	nbits := (rng + pageSize - 1) / pageSize
	nbytes := (nbits + 8 - 1) / 8
	npages := (nbytes + pageSize - 1) / pageSize
	if npages > largestDesc.numberOfPages {
		return kernError("initMem: memory bitmap doesn't fit in available memory")
	}
	mem.start = minAddr
	nwords := (nbytes + 8 - 1) / 8
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&mem.bits))
	hdr.Data = uintptr(largestDesc.physicalStart)
	hdr.Len = int(nwords)
	hdr.Cap = int(nwords)
	// Clear bitmap.
	for i := range mem.bits {
		mem.bits[i] = 0
	}
	// Mark free memory.
	for i := 0; i < efiMap.len(); i++ {
		desc := efiMap.entry(i)
		if !desc.isUsable() || desc._type == efiLoaderData {
			continue
		}
		start := desc.physicalStart
		end := start + physicalAddress(desc.numberOfPages*pageSize)
		if start < end {
			mem.setFree(true, start, end)
		}
	}
	// Reserve memory for the allocator itself.
	mem.setFree(false, largestDesc.physicalStart, largestDesc.physicalStart+physicalAddress(npages*pageSize))
	return nil
}

//go:nosplit
func (m *memory) setFree(free bool, start, end physicalAddress) {
	if start&^(pageSize-1) != start || end&^(pageSize-1) != end {
		fatal("markFree: unaligned memory range")
	}
	if start > end {
		fatal("markFree: start > end")
	}
	if start < m.start {
		fatal("markFree: start > m.start")
	}
	start -= m.start
	end -= m.start
	startBit := uint64(start / pageSize)
	endBit := uint64(end / pageSize)
	startWord := startBit / 64
	endWord := endBit / 64
	// Set the bits of the first and last word(s).
	startPattern := uint64(1)<<(64-startBit%64) - 1
	endPattern := ^(uint64(1)<<(64-endBit%64) - 1)
	if startWord == endWord {
		startPattern &= endPattern
		endPattern = startPattern
	}
	var pattern uint64
	if free {
		pattern = ^uint64(0)
		m.bits[startWord] |= startPattern
		m.bits[endWord] |= endPattern
	} else {
		pattern = 0
		m.bits[startWord] &^= startPattern
		m.bits[endWord] &^= endPattern
	}
	// Mark the middle bits.
	for i := startWord + 1; i < endWord; i++ {
		m.bits[i] = pattern
	}
}

// alloc allocates at most maxSize bytes of contiguous memory, rounded
// up to the page size. alloc returns at least a page of memory.
//go:nosplit
func (m *memory) alloc(maxSize int) (physicalAddress, int, error) {
	pageIdx, ok := m.nextFreePage()
	if !ok {
		return 0, 0, kernError("alloc: out of memory")
	}
	addr := physicalAddress(pageIdx * pageSize)
	var size int
	for maxSize > 0 {
		if !m.mark(pageIdx) {
			break
		}
		pageIdx++
		size += pageSize
		maxSize -= pageSize
	}
	mem := sliceForMem(physToVirt(addr), size)
	for i := range mem {
		mem[i] = 0
	}
	return addr, size, nil
}

//go:nosplit
func (m *memory) mark(pageIdx int) bool {
	wordIdx := pageIdx / 64
	bit := pageIdx % 64
	mask := uint64(1 << (64 - bit - 1))
	word := m.bits[wordIdx]
	if word&mask == 0 {
		return false
	}
	m.bits[wordIdx] = word &^ mask
	return true
}

//go:nosplit
func (m *memory) nextFreePage() (int, bool) {
	for i := 0; i < len(m.bits); i++ {
		idx := (i + m.word) % len(m.bits)
		w := m.bits[idx]
		b := bits.LeadingZeros64(w)
		if b == 64 {
			continue
		}
		m.word = idx
		return idx*64 + b, true
	}
	return 0, false
}

//go:nosplit
func (m *efiMemoryMap) entry(i int) *efiMemoryDescriptor {
	off := i * m.stride
	return (*efiMemoryDescriptor)(unsafe.Pointer(&m.mmap[off]))
}

//go:nosplit
func (m *efiMemoryMap) len() int {
	return len(m.mmap) / m.stride
}

// mmapAligned maps the virtual address range to a physical address
// range.
//go:nosplit
func mmapAligned(mem *memory, pml4 *pageTable, start, end virtualAddress, paddr physicalAddress, flags pageFlags) error {
	if paddr%pageSize != 0 {
		fatal("mmap: pagetable entry not aligned")
	}
	for start < end {
		size := end - start
		// Look up PML4 entry.
		pml4e := (start / pageSizeRoot) % pageTableSize
		pdpt, err := pml4.lookupOrCreatePageTable(mem, int(pml4e))
		if err != nil {
			return err
		}
		pdpte := (start / pageSize1GB) % pageTableSize
		if size >= pageSize1GB && start%pageSize2MB == 0 && paddr%pageSize1GB == 0 && hugePage1GBSupport {
			// Map a 1 GB page.
			pdpt[pdpte].mmap(paddr, flags|pageSizeFlag)
			paddr += pageSize1GB
			start += pageSize1GB
			continue
		}
		pd, err := pdpt.lookupOrCreatePageTable(mem, int(pdpte))
		if err != nil {
			return err
		}
		pde := (start / pageSize2MB) % pageTableSize
		if size >= pageSize2MB && start%pageSize2MB == 0 && paddr%pageSize2MB == 0 {
			// Map a 2MB page.
			pd[pde].mmap(paddr, flags|pageSizeFlag)
			paddr += pageSize2MB
			start += pageSize2MB
			continue
		}
		pt, err := pd.lookupOrCreatePageTable(mem, int(pde))
		if err != nil {
			return err
		}
		e := (start / pageSize) % pageTableSize
		pt[e].mmap(paddr, flags)
		paddr += pageSize
		start += pageSize
	}
	return nil
}

//go:nosplit
func (p *pageTable) lookupOrCreatePageTable(mem *memory, index int) (*pageTable, error) {
	entry := &p[index]
	if entry.present() {
		return entry.getPageTable(), nil
	}
	page, _, err := mem.alloc(pageSize)
	if err != nil {
		return nil, err
	}
	entry.setPageTable(page)
	return (*pageTable)(unsafe.Pointer(physToVirt(page))), nil
}

// setPageTable points the entry to a page table.
//go:nosplit
func (e *pageTableEntry) setPageTable(addr physicalAddress) {
	*e = pageTableEntry(addr) | pageTableEntry(pageFlagPresent|pageFlagWritable|pageFlagUserAccess)
}

// getPageTable reads a page table reference from the entry.
//go:nosplit
func (e *pageTableEntry) getPageTable() *pageTable {
	if pageFlags(*e)&pageSizeFlag != 0 {
		fatal("getPageTable: not a page table")
	}
	addr := physicalAddress(*e) & (_MAXPHYADDR - 1)
	// The address is page-aligned.
	addr = addr & ^(physicalAddress(pageSize) - 1)
	return (*pageTable)(unsafe.Pointer(physToVirt(addr)))
}

//go:nosplit
func (e *pageTableEntry) present() bool {
	return pageFlags(*e)&pageFlagPresent != 0
}

//go:nosplit
func (e *pageTableEntry) mmap(addr physicalAddress, flags pageFlags) {
	if !nxSupport {
		flags &= ^pageFlagNX
	}
	flags |= pageFlagPresent
	*e = pageTableEntry(addr) | pageTableEntry(flags)
}

//go:nosplit
func (e *pageTableEntry) setFlags(flags pageFlags) {
	*e &= ^pageTableEntry(allPageFlags)
	*e |= pageTableEntry(flags)
}

// isRuntime reports whether the memory region is used for the UEFI
// runtime.
//go:nosplit
func (e *efiMemoryDescriptor) isRuntime() bool {
	return e.attribute&_EFI_MEMORY_RUNTIME != 0
}

// isUsable reports whether the memory region is available for use.
//go:nosplit
func (e *efiMemoryDescriptor) isUsable() bool {
	if e.isRuntime() {
		return false
	}
	switch e._type {
	case efiLoaderCode, efiLoaderData, efiBootServicesCode, efiBootServicesData, efiConventionalMemory:
		return true
	default:
		return false
	}
}

//go:nosplit
func physToVirt(addr physicalAddress) virtualAddress {
	return physicalMapOffset + virtualAddress(addr)
}

//go:nosplit
func sliceForMem(addr virtualAddress, size int) []byte {
	var slice []byte
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&slice))
	hdr.Len = size
	hdr.Cap = size
	hdr.Data = uintptr(addr)
	return slice
}

//go:nosplit
func initPagingFeatures() {
	maxExt := cpuidMaxExt()
	if maxExt < 0x80000001 {
		return
	}
	_, _, _, edx := cpuid(0x80000001, 0)
	nxSupport = edx&(1<<20) != 0
	hugePage1GBSupport = edx&(1<<26) != 0
	maxVirtAddress = 1 << 32
	if edx&(1<<29) != 0 {
		maxVirtAddress = 1 << 48
	}
	if maxExt < 0x80000008 {
		return
	}
	eax, _, _, _ := cpuid(0x80000008, 0)
	virtWidth := (eax >> 8) & 0xff
	maxVirtAddress = 1 << virtWidth
}

// mmap reserves a virtual memory range size bytes big, preferring
// addr as starting address.
//go:nosplit
func (vm *virtMemory) mmap(addr virtualAddress, size uint64, flags pageFlags) (virtualAddress, error) {
	if addr == 0 {
		addr = vm.next
	}
	start := addr.Align()
	end := (addr + virtualAddress(size)).AlignUp()
	if vm.addRange(start, end, flags) {
		return start, nil
	}
	// Forward search for a starting address where the range fits.
	idx := vm.closestRange(vm.next)
	for ; idx < len(vm.ranges); idx++ {
		start := vm.ranges[idx].end
		end := (start + virtualAddress(size)).AlignUp()
		if vm.addRange(start, end, flags) {
			vm.next = end
			return start, nil
		}
	}
	return 0, kernError("mmap: failed to allocate memory")
}

// mmapFixed reserves a virtual memory range size bytes big at the
// page aligned address addr.
//go:nosplit
func (vm *virtMemory) mmapFixed(addr virtualAddress, size uint64, flags pageFlags) bool {
	if addr != addr.Align() {
		return false
	}
	end := (addr + virtualAddress(size)).AlignUp()
	return vm.addRange(addr, end, flags)
}

// mustAddRange is like addRange but calls fatal if the range
// overlaps.
//go:nosplit
func (vm *virtMemory) mustAddRange(start, end virtualAddress, flags pageFlags) {
	if !vm.addRange(start, end, flags) {
		fatal("mustAddRange: adding overlapping range")
	}
}

// rangeGorAddress returns the range that contains the
// address range or false if such range exists.
//go:nosplit
func (vm *virtMemory) rangeForAddress(addr virtualAddress, size int) (memoryRange, bool) {
	i := vm.closestRange(addr)
	if i >= len(vm.ranges) {
		return memoryRange{}, false
	}
	r := vm.ranges[i]
	if !r.containsRange(addr, size) {
		return memoryRange{}, false
	}
	return r, true
}

// addRange adds a memory range to the map. If the range overlaps
// an existing range, addRange does nothing and returns false.
//go:nosplit
func (vm *virtMemory) addRange(start, end virtualAddress, flags pageFlags) bool {
	if start > end {
		fatal("addRange: invalid range")
	}
	i := vm.closestRange(start)
	r := memoryRange{start: start, end: end, flags: flags}
	if i < len(vm.ranges) {
		if vm.ranges[i].overlaps(r) {
			return false
		}
	}
	// Expand.
	vm.ranges = vm.ranges[:len(vm.ranges)+1]
	copy(vm.ranges[i+1:], vm.ranges[i:])
	vm.ranges[i] = r
	return true
}

// closestRange finds the lowest index i where vm.ranges[i].end > addr.
//go:nosplit
func (vm *virtMemory) closestRange(addr virtualAddress) int {
	i, j := 0, len(vm.ranges)
	for i < j {
		h := int(uint(i+j) >> 1)
		if vm.ranges[h].end <= addr {
			i = h + 1
		} else {
			j = h
		}
	}
	return i
}

//go:nosplit
func (r memoryRange) containsRange(addr virtualAddress, size int) bool {
	return r.start <= addr && addr+virtualAddress(size) <= r.end
}

//go:nosplit
func (r memoryRange) contains(addr virtualAddress) bool {
	return r.start <= addr && addr < r.end
}

//go:nosplit
func (r memoryRange) overlaps(r2 memoryRange) bool {
	return r.start <= r2.start && r.end > r2.start ||
		r2.start <= r.start && r2.end > r.start
}

// Align the address downwards to the page size.
func (a virtualAddress) Align() virtualAddress {
	return a &^ virtualAddress(pageSize-1)
}

// Align the address upwards to the page size.
func (a virtualAddress) AlignUp() virtualAddress {
	return (a + pageSize - 1) & ^virtualAddress(pageSize-1)
}

//go:nosplit
func handlePageFault(errCode uint64, addr virtualAddress) {
	const (
		faultFlagPresent = 1 << 0
	)
	if errCode&faultFlagPresent != 0 {
		outputString("page fault address: ")
		outputUint64(uint64(addr))
		outputString("\n")
		fatal("handlePageFault: page protection fault")
	}
	if err := faultPage(addr); err != nil {
		outputString("page fault address: ")
		outputUint64(uint64(addr))
		outputString("\n")
		fatalError(err)
	}
}

func pageFaultTrampoline()

func setCR3Reg(addr uintptr)
