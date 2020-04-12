// SPDX-License-Identifier: Unlicense OR MIT

package pci

import (
	"errors"
	"sync/atomic"
	"unsafe"

	"eliasnaur.com/unik/kernel"
)

// Address represents a PCI device.
type Address struct {
	Bus, Device, Function uint8
}

type InterruptTable struct {
	NumInterrupts int
	msixTable     []msixTableEntry
}

// msixTableEntry is the hardware representation of
// an MSI-X table entry.
type msixTableEntry struct {
	addrLo  uint32
	addrHi  uint32
	data    uint32
	control uint32
}

const (
	pciConfigAddrPort = 0xcf8
	pciConfigDataPort = 0xcfc
)

const (
	_PCI_CAP_ID_MSI  = 0x05
	_PCI_CAP_ID_MSIX = 0x11
)

func Detect() ([]Address, error) {
	var addrs []Address
	// Run through all possible PCI host controllers.
	for function := uint8(0); function <= 7; function++ {
		if (Address{Function: function}).ReadVendorID() == 0xFFFF {
			break
		}
		if err := searchPCIBus(&addrs, function); err != nil {
			return addrs, err
		}
	}
	return addrs, nil
}

func searchPCIBus(addrs *[]Address, bus uint8) error {
	for device := uint8(0); device <= 31; device++ {
		if err := searchPCIDevice(addrs, bus, device); err != nil {
			return err
		}
	}
	return nil
}

func searchPCIDevice(addrs *[]Address, bus, device uint8) error {
	addr := Address{Bus: bus, Device: device}
	if vendorID := addr.ReadVendorID(); vendorID == 0xFFFF {
		return nil
	}
	maxFunc := uint8(0)
	if headerType := addr.readHeaderType(); headerType&0x80 != 0 {
		// Multi-function device.
		maxFunc = 7
	}
	for function := uint8(0); function <= maxFunc; function++ {
		addr := addr
		addr.Function = function
		if addr.ReadVendorID() == 0xFFFF {
			continue
		}
		if err := searchPCIFunction(addrs, addr); err != nil {
			return err
		}
	}
	return nil
}

func searchPCIFunction(addrs *[]Address, addr Address) error {
	header := addr.readHeaderType()
	switch header & 0x7f {
	case 0x00:
		// Standard device.
		*addrs = append(*addrs, addr)
	case 0x01:
		// PCI-to-PCI bridge.
		secondaryBus := addr.readSecondaryBus()
		return searchPCIBus(addrs, secondaryBus)
	}
	return nil
}

func (a Address) ReadBAR(bar uint8) (addr uint64, prefetch, isMem bool) {
	if bar > 0x5 {
		panic("invalid BAR")
	}
	addr0 := a.ReadPCIRegister(0x10 + bar*4)
	if addr0&1 != 0 {
		// I/O address.
		return uint64(addr0 &^ 0b11), false, false
	}
	// Mask off flags.
	addr = uint64(addr0 &^ 0xf)
	switch (addr0 >> 1) & 0b11 {
	case 0b01:
		// 16-bit address. Not used.
		return addr, false, false
	case 0b00:
	case 0b10:
		// 64-bit address.
		addr1 := a.ReadPCIRegister(0x10 + (bar+1)*4)
		addr |= uint64(addr1) << 32
	}
	prefetch = addr0&0b1000 != 0
	return addr, prefetch, true
}

func (a Address) ReadCapOffset() uint8 {
	return uint8(a.ReadPCIRegister(0x34)) &^ 0x3
}

func (a Address) ReadStatus() uint16 {
	return uint16(a.ReadPCIRegister(0x4) >> 16)
}

func (a Address) ReadDeviceID() uint16 {
	return uint16(a.ReadPCIRegister(0x0) >> 16)
}

func (a Address) ReadVendorID() uint16 {
	return uint16(a.ReadPCIRegister(0x0))
}

func (a Address) readHeaderType() uint8 {
	return uint8(a.ReadPCIRegister(0xc) >> 16)
}

func (a Address) readPCIClass() uint16 {
	return uint16(a.ReadPCIRegister(0x8) >> 16)
}

func (a Address) readSecondaryBus() uint8 {
	return uint8(a.ReadPCIRegister(0x18) >> 8)
}

func (a Address) ReadPCIRegister(reg uint8) uint32 {
	if reg&0x3 != 0 {
		panic("unaligned PCI register access")
	}
	addr := 0x80000000 | uint32(a.Bus)<<16 | uint32(a.Device)<<11 | uint32(a.Function)<<8 | uint32(reg)
	kernel.Outl(pciConfigAddrPort, addr)
	return kernel.Inl(pciConfigDataPort)
}

func (a Address) writePCIRegister(reg uint8, val uint32) {
	if reg&0x3 != 0 {
		panic("unaligned PCI register access")
	}
	addr := 0x80000000 | uint32(a.Bus)<<16 | uint32(a.Device)<<11 | uint32(a.Function)<<8 | uint32(reg)
	kernel.Outl(pciConfigAddrPort, addr)
	kernel.Outl(pciConfigDataPort, val)
}

func (t *InterruptTable) SetupInterrupt(intr int, enable bool, addr uint64, data uint32) {
	if addr&0x2 != 0 {
		panic("pci: unaligned message address")
	}
	if !enable {
		addr |= 0b10
	}
	if intr < 0 || intr >= t.NumInterrupts {
		panic("pci: interrupt number out of range")
	}
	atomic.StoreUint32(&t.msixTable[intr].addrLo, uint32(addr))
	atomic.StoreUint32(&t.msixTable[intr].addrHi, uint32(addr>>32))
	atomic.StoreUint32(&t.msixTable[intr].data, data)
	ctrl := atomic.LoadUint32(&t.msixTable[intr].control)
	// Unmask.
	ctrl &^= 0b1
	atomic.StoreUint32(&t.msixTable[intr].control, ctrl)
}

func (a Address) InitInterrupts() (*InterruptTable, error) {
	nextCap := a.ReadCapOffset()
	for nextCap != 0 {
		capOff := nextCap
		w0 := a.ReadPCIRegister(capOff)
		nextCap = uint8(w0 >> 8)
		capID := uint8(w0)
		if capID != _PCI_CAP_ID_MSIX {
			continue
		}
		offBAR := a.ReadPCIRegister(capOff + 4)
		barOff := offBAR &^ 0b111
		BAR := uint8(offBAR & 0b111)
		barAddr, _, isMem := a.ReadBAR(BAR)
		if !isMem {
			continue
		}
		ninterrupts := int((w0>>16)&0x7ff) + 1
		tbl := new(InterruptTable)
		tbl.NumInterrupts = ninterrupts
		base := uintptr(barAddr) + uintptr(barOff)
		intTable, err := kernel.Map(base, int(unsafe.Sizeof(msixTableEntry{}))*ninterrupts)
		if err != nil {
			return nil, err
		}
		cap := len(intTable) / int(unsafe.Sizeof(msixTableEntry{}))
		tbl.msixTable = (*(*[2 << 30]msixTableEntry)(unsafe.Pointer(&intTable[0])))[:cap:cap]
		// Enable interrupts.
		a.writePCIRegister(capOff, w0|1<<31)
		return tbl, nil
	}
	return nil, errors.New("pci: MSI-X is not supported")
}
