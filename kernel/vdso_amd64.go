// SPDX-License-Identifier: Unlicense OR MIT

package kernel

import "encoding/binary"

const vdsoAddress virtualAddress = 0xffffffffff600000

//go:nosplit
func initVDSO() error {
	// We're not yet passing a full-fledged vDSO AT_SYSINFO_EHDR to
	// the Go runtime, in which case it expects an implementation of
	// gettimeofday at address 0xffffffffff600000.
	// Map a page that jumps to our implementation.
	if !globalMap.mmapFixed(vdsoAddress, pageSize, pageFlagWritable|pageFlagUserAccess) {
		return kernError("setupVDSO: failed to map vDSO page")
	}
	page := sliceForMem(vdsoAddress, pageSize)
	// MOVQ $vdsoGettimeofday(SB), R11
	page[0] = 0x49
	page[1] = 0xbb
	gettimeofday := funcPC(vdsoGettimeofday)
	binary.LittleEndian.PutUint64(page[2:10], uint64(gettimeofday))
	// JMP R11
	page[10] = 0x41
	page[11] = 0xff
	page[12] = 0xe3
	return nil
}

func vdsoGettimeofday()
