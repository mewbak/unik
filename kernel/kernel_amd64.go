// SPDX-License-Identifier: Unlicense OR MIT

package kernel

import (
	"encoding/binary"
	"unsafe"
)

// kernError is an error type usable in kernel code.
type kernError string

const (
	_MSR_IA32_EFER = 0xc0000080
	_MSR_FS_BASE   = 0xc0000100

	_EFER_SCE = 1 << 0  // Enable SYSCALL.
	_EFER_NXE = 1 << 11 // Enable no-execute page bit.

	_XCR0_FPU = 1 << 0
	_XCR0_SSE = 1 << 1
	_XCR0_AVX = 1 << 2

	_CR4_DE         = 1 << 3
	_CR4_PSE        = 1 << 4
	_CR4_PAE        = 1 << 5
	_CR4_FXSTOR     = 1 << 9
	_CR4_OSXMMEXCPT = 1 << 10
	_CR4_FSGSBASE   = 1 << 16
	_CR4_OSXSAVE    = 1 << 18
)

type stack [10 * pageSize]byte

var (
	// Kernel stack.
	kstack stack

	fpuContextSize uint64
	kstackTop      uint64
)

//go:nosplit
func kernelStackTop() uint64 {
	return uint64(kstack.top())
}

//go:nosplit
func runKernel(mmapSize, descSize, kernelImageSize uint64, mmapAddr, kernelImage *byte) {
	mmap := (*(*[1 << 30]byte)(unsafe.Pointer(mmapAddr)))[:mmapSize:mmapSize]
	img := (*(*[1 << 30]byte)(unsafe.Pointer(kernelImage)))[:kernelImageSize:kernelImageSize]
	if err := initKernel(descSize, mmap, img); err != nil {
		fatalError(err)
	}
	if err := runGo(); err != nil {
		fatalError(err)
	}
	fatal("runKernel: runGo returned")
}

//go:nosplit
func initKernel(descSize uint64, mmap, kernelImage []byte) error {
	kstackTop = kernelStackTop()
	efiMap := efiMemoryMap{mmap: mmap, stride: int(descSize)}
	setCR4Reg(_CR4_PAE | _CR4_PSE | _CR4_DE | _CR4_FXSTOR | _CR4_OSXMMEXCPT)
	loadGDT()
	kernelThread.self = &kernelThread
	thread0.self = &thread0
	thread0.makeCurrent()
	if err := initMemory(efiMap, kernelImage); err != nil {
		return err
	}
	if err := initAPIC(); err != nil {
		return err
	}
	initSYSCALL()
	if err := initVDSO(); err != nil {
		return err
	}
	if err := initThreads(); err != nil {
		return err
	}
	if err := initClock(); err != nil {
		return err
	}
	return nil
}

//go:nosplit
func runGo() error {
	// Allocate initial stack.
	ssize := uint64(unsafe.Sizeof(stack{}))
	addr, err := globalMap.mmap(0, ssize, pageFlagNX|pageFlagWritable|pageFlagUserAccess)
	if err != nil {
		return err
	}
	stack := (*stack)(unsafe.Pointer(addr))

	// Allocate initial thread.
	t, err := globalThreads.newThread()
	if err != nil {
		return err
	}
	t.sp = uint64(stack.top())
	t.makeCurrent()

	// Set up sane initial state, in particular the MXCSR flags.
	saveThread()
	// Prepare program environment on stack.
	const envSize = 256
	setupEnv(stack[len(stack)-envSize:])
	t.sp -= envSize
	t.flags = _FLAG_RESERVED | _FLAG_IF
	// Jump to Go runtime start.
	t.ip = uint64(funcPC(jumpToGo))
	resumeThread()
	return nil
}

// setupEnv sets up the argv, auxv and env on the stack, mimicing
// the Linux kernel.
//go:nosplit
func setupEnv(stack []byte) {
	args := stack
	bo := binary.LittleEndian
	bo.PutUint64(args, 1) // 1 argument, the process name.
	args = args[8:]
	// First argument, address of binary name.
	binAddr := args[:8]
	args = args[8:]
	bo.PutUint64(args, 0) // NULL separator.
	args = args[8:]
	bo.PutUint64(args, 0) // No envp.
	args = args[8:]
	// Build auxillary vector.
	// Page size.
	bo.PutUint64(args, _AT_PAGESZ)
	args = args[8:]
	bo.PutUint64(args, pageSize)
	args = args[8:]
	// End of auxv.
	bo.PutUint64(args, _AT_NULL)
	args = args[8:]
	bo.PutUint64(args, 0)
	// Binary name.
	bo.PutUint64(binAddr, uint64(uintptr(unsafe.Pointer(&args[0]))))
	n := copy(args, []byte("kernel\x00"))
	args = args[n:]
}

//go:nosplit
func funcPC(f func()) uintptr {
	return **(**uintptr)(unsafe.Pointer(&f))
}

//go:nosplit
func wrmsr(register uint32, value uint64) {
	wrmsr0(register, uint32(value), uint32(value>>32))
}

//go:nosplit
func rdmsr(register uint32) uint64 {
	lo, hi := rdmsr0(register)
	return uint64(hi)<<32 | uint64(lo)
}

//go:nosplit
func fatalError(err error) {
	// The only error type supported is kernError,
	// but we can't call its Error method directly,
	// because the compiler wrapper is not nosplit.
	switch err := err.(type) {
	case kernError:
		fatal(err.Error())
	default:
		fatal("unsupported error")
	}
}

//go:nosplit
func fatal(msg string) {
	outputString("fatal error: ")
	outputString(msg)
	outputString("\n")
	halt()
}

// cpuidMaxExt returns the highest supported CPUID extended
// function number.
//go:nosplit
func cpuidMaxExt() uint32 {
	eax, _, _, _ := cpuid(0x80000000, 0)
	return eax
}

//go:nosplit
func hasInvariantTSC() bool {
	maxExt := cpuidMaxExt()
	if maxExt < 0x80000007 {
		return false
	}
	_, _, _, edx := cpuid(0x80000007, 0)
	return edx&(1<<8) != 0
}

//go:nosplit
func (s *stack) slice() []byte {
	stackTop := uintptr(unsafe.Pointer(&s[0])) + unsafe.Sizeof(*s)
	// Align to 16 bytes.
	alignment := int(stackTop & 0xf)
	return s[:len(s)-alignment]
}

//go:nosplit
func (s *stack) top() virtualAddress {
	stackTop := uintptr(unsafe.Pointer(&s[0])) + unsafe.Sizeof(*s)
	// Align to 16 bytes.
	stackTop = stackTop &^ 0xf
	return virtualAddress(stackTop)
}

func wrmsr0(register, lo, hi uint32)
func rdmsr0(register uint32) (lo, hi uint32)
func halt()
func cpuid(function, sub uint32) (eax, ebx, ecx, edx uint32)
func jumpToGo()
func fninit()
func setCR4Reg(flags uint64)
func outb(port uint16, b uint8)
func inb(port uint16) uint8
func outl(port uint16, b uint32)
func inl(port uint16) uint32

//go:nosplit
func (k kernError) Error() string {
	return string(k)
}
