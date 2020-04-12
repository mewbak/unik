// SPDX-License-Identifier: Unlicense OR MIT

// User space interface to the kernel.

package kernel

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"syscall"
	"unsafe"
)

// InterruptMessage describes a message signalled interrupt (MSI)
// address and data.
type InterruptMessage struct {
	Addr uint64
	Data uint32
}

type interruptMessageAndError struct {
	msg InterruptMessage
	err error
}

type interruptHandler struct {
	once      sync.Once
	allocIntr chan chan<- struct{}
	allocResp chan interruptMessageAndError

	handlers []chan<- struct{}
}

type interrupt int

var userHandler interruptHandler

// Outl executes an outl instruction.
func Outl(port uint16, val uint32) {
	syscall.RawSyscall(_SYS_outl, uintptr(port), uintptr(val), 0)
}

// Inl executes an inl instruction.
func Inl(port uint16) uint32 {
	r, _, _ := syscall.RawSyscall(_SYS_inl, uintptr(port), 0, 0)
	return uint32(r)
}

// Alloc allocates at most maxSize bytes of contiguous physical memory. It
// returns the physical address and size of the block or an error.
func Alloc(maxSize int) (uintptr, int, error) {
	r, size, errno := syscall.Syscall(_SYS_alloc, uintptr(maxSize), 0, 0)
	if errno != 0 {
		return 0, 0, errno
	}
	return r, int(size), nil
}

// AllocInterrupt reserves and sets up an MSI interrupt.
func AllocInterrupt(ch chan<- struct{}) (InterruptMessage, error) {
	return userHandler.alloc(ch)
}

func (h *interruptHandler) alloc(ch chan<- struct{}) (InterruptMessage, error) {
	h.once.Do(func() {
		h.allocIntr = make(chan chan<- struct{})
		h.allocResp = make(chan interruptMessageAndError)
		go h.run()
	})
	h.allocIntr <- ch
	ret := <-h.allocResp
	return ret.msg, ret.err
}

func (h *interruptHandler) run() {
	interrupts := make(chan interrupt)
	go func() {
		for {
			_, intno, errno := syscall.Syscall(_SYS_waitinterrupt, 0, 0, 0)
			if errno != 0 {
				var err error = errno
				panic(err)
			}
			interrupts <- interrupt(intno)
		}
	}()
	for {
		select {
		case intr := <-interrupts:
			select {
			case h.handlers[intr] <- struct{}{}:
			default:
			}
		case ch := <-h.allocIntr:
			msg, err := h.alloc0(ch)
			h.allocResp <- interruptMessageAndError{msg, err}
		}
	}
}

func (h *interruptHandler) alloc0(ch chan<- struct{}) (InterruptMessage, error) {
	intno := len(h.handlers)
	max := int(intLastUser - intFirstUser)
	if intno >= max {
		return InterruptMessage{}, errors.New("kernel: no interrupt available")
	}
	h.handlers = append(h.handlers, ch)
	return InterruptMessage{
		Addr: 0xfee << 20,
		Data: uint32(intFirstUser) + uint32(intno),
	}, nil
}

// Map the physical address and the following size bytes into virtual
// memory.
func Map(addr uintptr, size int) ([]byte, error) {
	addrAlign := addr &^ (pageSize - 1)
	off := int(addr - addrAlign)
	sizeAlign := size + off
	sizeAlign = (size + pageSize - 1) &^ (pageSize - 1)
	vmem, err := syscall.Mmap(0, 0, int(sizeAlign), syscall.PROT_WRITE|syscall.PROT_READ, syscall.MAP_ANONYMOUS)
	if err != nil {
		return nil, fmt.Errorf("kernel: Map failed to allocate virtual memory: %v", err)
	}
	vaddr := ((*reflect.SliceHeader)(unsafe.Pointer(&vmem))).Data
	if err := IOMap(vaddr, uintptr(addrAlign), sizeAlign); err != nil {
		syscall.Munmap(vmem)
		return nil, fmt.Errorf("kernel: Map failed to map physical memory %#x: %v", addr, err)
	}
	return vmem[off : off+size : off+size], nil
}

// IOMap maps the already mmap'ed memory range [vaddr; vaddr+size[ to
// the contiguous physical memory range starting at addr. Both vaddr,
// addr and size must be page-aligned.
func IOMap(vaddr, addr uintptr, size int) error {
	_, _, errno := syscall.Syscall(_SYS_iomap, vaddr, addr, uintptr(size))
	if errno != 0 {
		return errno
	}
	return nil
}
