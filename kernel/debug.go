// SPDX-License-Identifier: Unlicense OR MIT

package kernel

import (
	"fmt"
	"sort"
)

type pageTableRange struct {
	vaddr virtualAddress
	paddr physicalAddress
	size  int
}

func Verify() {
	entries := dumpPageTable(globalPT)
	verifyPageTable(entries)
}

func verifyPageTable(entries []pageTableRange) {
	type addrRange struct {
		start uint64
		end   uint64
		r     pageTableRange
	}
	var pranges []addrRange
	for _, e := range entries {
		if e.vaddr >= physicalMapOffset {
			// Ignore the identity mapped memory which overlaps
			// existing mappings per design.
			continue
		}
		pranges = append(pranges, addrRange{
			start: uint64(e.paddr),
			end:   uint64(e.paddr) + uint64(e.size),
			r:     e,
		})
	}
	overlap := false
	sort.Slice(pranges, func(i, j int) bool {
		r1, r2 := pranges[i], pranges[j]
		if r1.start < r2.start {
			return true
		}
		if r1.start == r2.start && r1.end < r2.end {
			return true
		}
		if r1.start == r2.start && r1.end == r2.end {
			overlap = true
			fmt.Printf("overlapping range: %#x %#x\n", r1.r, r2.r)
		}
		return false
	})
	for i := 0; i < len(pranges)-1; i++ {
		r1 := pranges[i]
		r2 := pranges[i+1]
		if r1.end > r2.start {
			overlap = true
			fmt.Printf("overlapping range: %#x %#x\n", r1.r, r2.r)
		}
	}
	if overlap {
		/*for _, e := range pranges {
			fmt.Printf("range: vaddr: %#x paddr: %#x size: %#x\n", e.r.vaddr, e.r.paddr, e.r.size)
		}*/
		panic("overlapping ranges")
	}
}

func dumpPageTable(pml4 *pageTable) []pageTableRange {
	var entries []pageTableRange
	for i, pml4e := range pml4 {
		if !pml4e.present() {
			continue
		}
		vaddr := virtualAddress(i * pageSizeRoot)
		// Sign extend.
		if vaddr&(maxVirtAddress>>1) != 0 {
			vaddr |= ^(maxVirtAddress - 1)
		}
		pdpt := pml4e.getPageTable()
		for i, pdpte := range pdpt {
			if !pdpte.present() {
				continue
			}
			vaddr := vaddr + virtualAddress(i)*pageSize1GB
			if pageFlags(pdpte)&pageSizeFlag != 0 {
				// 1GB page.
				paddr := physicalAddress(pdpte) & (_MAXPHYADDR - 1) &^ (pageSize1GB - 1)
				entries = append(entries, pageTableRange{vaddr, paddr, pageSize1GB})
			} else {
				pd := pdpte.getPageTable()
				for i, pde := range pd {
					if !pde.present() {
						continue
					}
					vaddr := vaddr + virtualAddress(i)*pageSize2MB
					if pageFlags(pde)&pageSizeFlag != 0 {
						// 2MB page.
						paddr := physicalAddress(pde) & (_MAXPHYADDR - 1) &^ (pageSize2MB - 1)
						entries = append(entries, pageTableRange{vaddr, paddr, pageSize2MB})
					} else {
						pt := pde.getPageTable()
						for i, e := range pt {
							if !e.present() {
								continue
							}
							vaddr := vaddr + virtualAddress(i)*pageSize
							// 4kb page.
							paddr := physicalAddress(e) & (_MAXPHYADDR - 1) &^ (pageSize - 1)
							entries = append(entries, pageTableRange{vaddr, paddr, pageSize})
						}
					}
				}
			}
		}
	}
	return entries
}

func dumpPageEntry(vaddr virtualAddress, paddr physicalAddress, size int) {
	fmt.Printf("mapping vaddr: %#x paddr: %#x size %#x\n", vaddr, paddr, size)
}
