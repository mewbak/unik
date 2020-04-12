// SPDX-License-Identifier: Unlicense OR MIT

package kernel

//go:noescape
func OrUint8(addr *byte, val byte)

//go:noescape
func StoreUint8(addr *byte, val byte)

//go:noescape
func StoreUint16(addr *uint16, val uint16)

//go:nosplit
func LoadUint16(addr *uint16) uint16 {
	return *addr
}

//go:nosplit
func LoadUint8(addr *byte) byte {
	return *addr
}
