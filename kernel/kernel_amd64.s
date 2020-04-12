// SPDX-License-Identifier: Unlicense OR MIT

#include "textflag.h"

#define CONTEXT_SELF 0*8
#define CONTEXT_IP 1*8
#define CONTEXT_SP 2*8
#define CONTEXT_FLAGS 3*8
#define CONTEXT_BP 4*8
#define CONTEXT_AX 5*8
#define CONTEXT_BX 6*8
#define CONTEXT_CX 7*8
#define CONTEXT_DX 8*8
#define CONTEXT_SI 9*8
#define CONTEXT_DI 10*8
#define CONTEXT_R8 11*8
#define CONTEXT_R9 12*8
#define CONTEXT_R10 13*8
#define CONTEXT_R11 14*8
#define CONTEXT_R12 15*8
#define CONTEXT_R13 16*8
#define CONTEXT_R14 17*8
#define CONTEXT_R15 18*8
#define CONTEXT_FSBASE 19*8
#define CONTEXT_FPSTATE 20*8

// Field offsets of type clock.
#define CLOCK_SEQ 0
#define CLOCK_SECONDS 8
#define CLOCK_NANOSECONDS 16

// Send end-of-interrupt to the APIC.
#define APICEOI	MOVQ	·apicEOI(SB), AX \
			MOVL	$0, (AX)

// INTERRUPT_SAVE/RESTORE assumes stack is aligned so that
// SP % 16 == 8.
#define INTERRUPT_SAVE SUBQ    $16*8+512, SP \
			MOVQ    BP, 0*8(SP) \
			MOVQ    AX, 1*8(SP) \
			MOVQ    BX, 2*8(SP) \
			MOVQ    CX, 3*8(SP) \
			MOVQ    DX, 4*8(SP) \
			MOVQ    SI, 5*8(SP) \
			MOVQ    DI, 6*8(SP) \
			MOVQ    R8, 7*8(SP) \
			MOVQ    R9, 8*8(SP) \
			MOVQ    R10, 9*8(SP) \
			MOVQ    R11, 10*8(SP) \
			MOVQ    R12, 11*8(SP) \
			MOVQ    R13, 12*8(SP) \
			MOVQ    R14, 13*8(SP) \
			MOVQ    R15, 14*8(SP) \
			FXSAVE	15*8(SP)

#define INTERRUPT_RESTORE FXRSTOR	15*8(SP) \
			MOVQ	0*8(SP), BP \
			MOVQ	1*8(SP), AX \
			MOVQ	2*8(SP), BX \
			MOVQ	3*8(SP), CX \
			MOVQ	4*8(SP), DX \
			MOVQ	5*8(SP), SI \
			MOVQ	6*8(SP), DI \
			MOVQ	7*8(SP), R8 \
			MOVQ	8*8(SP), R9 \
			MOVQ	9*8(SP), R10 \
			MOVQ	10*8(SP), R11 \
			MOVQ	11*8(SP), R12 \
			MOVQ	12*8(SP), R13 \
			MOVQ	13*8(SP), R14 \
			MOVQ	14*8(SP), R15 \
			ADDQ	$16*8+512, SP

#define USER_TRAMPOLINE(VECTOR) INTERRUPT_SAVE \
	PUSHQ $VECTOR \
	CALL ·userInterrupt(SB) \
	ADDQ	$8, SP \
	APICEOI \
	INTERRUPT_RESTORE \
	IRETQ

TEXT ·syscallTrampoline(SB),NOSPLIT|NOFRAME,$0
	SWAPGS
	// SYSCALL passes return address in CX, flags in R11.
	MOVQ	CX, CONTEXT_IP(GS)
	MOVQ	R11, CONTEXT_FLAGS(GS)
	// Save stack pointer.
	MOVQ	SP, CONTEXT_SP(GS)

	// Switch stack.
	MOVQ	·kstackTop(SB), SP

	CALL	·saveThread(SB)

	SUBQ	$8*8, SP

	MOVQ	CONTEXT_SELF(GS), BX
	MOVQ	BX, 0*8(SP) // Thread.
	MOVQ	AX, 1*8(SP) // Syscall number.
	// Up to 6 arguments.
	MOVQ	DI, 2*8(SP)
	MOVQ	SI, 3*8(SP)
	MOVQ	DX, 4*8(SP)
	MOVQ	R10, 5*8(SP)
	MOVQ	R8, 6*8(SP)
	MOVQ	R9, 7*8(SP)
	CALL	·sysenter(SB)

	ADDQ	$8*8, SP

	UNDEF // sysenter never returns.

// vdsoGettimeofday uses the C ABI.
TEXT ·vdsoGettimeofday(SB),NOSPLIT|NOFRAME,$0
	MOVQ	$·unixClock(SB), R10

retry:
	MOVQ	CLOCK_SEQ(R10), R8
	// Retry if seq is odd, indicating a write in progress.
	TESTB	$1, R8
	JNZ		retry
	MOVQ	CLOCK_SECONDS(R10), CX
	MOVL	CLOCK_NANOSECONDS(R10), AX
	// Retry if seq changed during the read.
	MOVQ	CLOCK_SEQ(R10), R9
	CMPQ	R8, R9
	JNE		retry
	// Convert to milliseconds.
	MOVL	$0, DX
	MOVL	$1000, R9
	DIVL	R9
	// Address of the result is in DI.
	MOVQ	CX, 0(DI) // Seconds.
	MOVL	AX, 8(DI) // Microseconds.
	MOVQ	$0, AX // Success.
	RET

TEXT ·generalProtectionFaultTrampoline(SB),NOSPLIT|NOFRAME,$0
	// The error code offsets the interrupt alignment by 8.
	// Re-align.
	SUBQ	$1*8, SP
	INTERRUPT_SAVE

	MOVQ	18*8+512(SP), AX // Instruction pointer from interrupt frame.

	SUBQ	$1*8, SP
	MOVQ	AX, 0*8(SP)
	CALL	·gpFault(SB)
	ADDQ	$1*8, SP

	INTERRUPT_RESTORE

	// Pop error code and alignment.
	ADDQ	$2*8, SP

	IRETQ

TEXT ·pageFaultTrampoline(SB),NOSPLIT|NOFRAME,$0
	// The error code offsets the interrupt alignment by 8.
	// Re-align.
	SUBQ	$1*8, SP
	INTERRUPT_SAVE

	MOVQ	17*8+512(SP), AX // Error code from interrupt frame.
	MOVQ	CR2, BX	// Fault address.

	SUBQ	$2*8, SP
	MOVQ	AX, 0*8(SP)
	MOVQ	BX, 1*8(SP)
	CALL	·handlePageFault(SB)
	ADDQ	$2*8, SP

	INTERRUPT_RESTORE

	// Pop error code and alignment.
	ADDQ	$2*8, SP

	IRETQ

TEXT ·timerTrampoline(SB),NOSPLIT|NOFRAME,$0
	SWAPGS
	// Save CX and R11 not saved by saveThread.
	MOVQ	CX, CONTEXT_CX(GS)
	MOVQ	R11, CONTEXT_R11(GS)

	CALL	·saveThread(SB)

	// Save return address, stack pointer, flags from the
	// interrupt stack frame.
	MOVQ	3*8(SP), AX // SP.
	MOVQ	AX, CONTEXT_SP(GS)
	MOVQ	2*8(SP), AX	// rflags.
	MOVQ	AX, CONTEXT_FLAGS(GS)
	MOVQ	0*8(SP), AX // Return address.
	MOVQ	AX, CONTEXT_IP(GS)

	// Send end-of-interrupt.
	APICEOI

	MOVQ	CONTEXT_SELF(GS), BX
	MOVQ	$·kernelThread(SB), CX
	CMPQ	BX, CX
	JEQ	kernelThread

	// Pop interrupt frame (5 words)
	ADDQ	$5*8, SP

	SUBQ	$8, SP
	MOVQ	BX, 0*8(SP) // Thread.
	CALL	·interruptSchedule(SB)
	ADDQ	$8, SP

	UNDEF // interruptSchedule never returns.

kernelThread:
	// We're interrupting the kernel thread, just
	// return.
	CALL	·restoreThread(SB)

	// Restore remaining registers.
	MOVQ	CONTEXT_CX(GS), CX
	MOVQ	CONTEXT_R11(GS), R11

	SWAPGS
	IRETQ

TEXT ·rt0(SB),NOSPLIT|NOFRAME,$0
	// Switch stack.
	CALL	·kernelStackTop(SB)
	MOVQ	0(SP), BP
	MOVQ	BP, SP

	SUBQ	$5*8, SP
	MOVQ	DI, 0(SP)	// Memory map size
	MOVQ	SI, 8(SP)	// Memory map descriptor size
	MOVQ	DX, 16(SP)	// Kernel image size
	MOVQ	CX, 24(SP)	// Memory map
	MOVQ	R8, 32(SP)	// Kernel image
	CALL	·runKernel(SB)
	ADDQ	$5*8, SP

	// runKernel should never return.
	UNDEF
	RET

TEXT ·jumpToGo(SB),NOSPLIT|NOFRAME,$0
	JMP _rt0_amd64_linux(SB)

TEXT ·rdmsr0(SB),NOSPLIT,$0-16
	MOVL register+0(FP), CX
	RDMSR
	MOVL	AX, lo+8(FP)
	MOVL	DX, hi+12(FP)
	RET

TEXT ·wrmsr0(SB),NOSPLIT,$0-12
	MOVL	register+0(FP), CX
	MOVL	lo+4(FP), AX
	MOVL	hi+8(FP), DX
	WRMSR
	RET

TEXT ·lgdt(SB),NOSPLIT,$0-8
	MOVQ	addr+0(FP), AX
	LGDT	(AX)
	RET

TEXT ·lidt(SB),NOSPLIT,$0-8
	MOVQ	addr+0(FP), AX
	LIDT	(AX)
	RET

TEXT ·setCSReg(SB),NOSPLIT,$0-2
	MOVW    seg+0(FP), AX
	// Allocate space for the long pointer (addr, segment).
	SUBQ	$16, SP
	MOVW	AX, 8(SP) // Segment.
	// The long jump should only change the CS segment
	// register. Use the address of the instruction
	// just after the jump.
	// The Go assembler can't load the instruction pointer.
	// MOVQ	IP+8, BX
	BYTE $0x48; BYTE $0x8d; BYTE $0x1d; BYTE $0x08; BYTE $0x00; BYTE $0x00; BYTE $0x0
	// Use raw opcodes to ensure their size.
	// MOVQ BX, 0(SP)
	BYTE $0x48; BYTE $0x89; BYTE $0x1c; BYTE $0x24
	// LJMPQ	(SP)
	BYTE $0x48; BYTE $0xFF; BYTE $0x2C; BYTE $0x24
	ADDQ	$16, SP
	RET

TEXT ·setCR3Reg(SB),NOSPLIT,$0-8
	MOVQ	addr+0(FP), AX
	MOVQ	AX, CR3
	RET

TEXT ·setCR4Reg(SB),NOSPLIT,$0-8
	MOVQ	flags+0(FP), AX
	MOVQ	AX, CR4
	RET

TEXT ·setDSReg(SB),NOSPLIT,$0-2
	MOVW	seg+0(FP), AX
	MOVW	AX, DS
	RET

TEXT ·setSSReg(SB),NOSPLIT,$0-2
	MOVW	seg+0(FP), AX
	MOVW	AX, SS
	RET

TEXT ·setESReg(SB),NOSPLIT,$0-2
	MOVW	seg+0(FP), AX
	MOVW	AX, ES
	RET

TEXT ·setFSReg(SB),NOSPLIT,$0-2
	MOVW	seg+0(FP), AX
	MOVW	AX, FS
	RET

TEXT ·setGSReg(SB),NOSPLIT,$0-2
	MOVW	seg+0(FP), AX
	MOVW	AX, GS
	RET

TEXT ·ltr(SB),NOSPLIT,$0-2
	MOVW	seg+0(FP), AX
	LTR	AX
	RET

TEXT ·yield0(SB),NOSPLIT,$0-0
	STI
	HLT
	CLI
	RET

TEXT ·halt(SB),NOSPLIT,$0-0
hlt:
	HLT
	JMP hlt
	RET

TEXT ·cpuid(SB),NOSPLIT,$0-24
	MOVL	function+0(FP), AX
	MOVL	sub+4(FP), CX
	CPUID
	MOVL	AX, eax+8(FP)
	MOVL	BX, ebx+12(FP)
	MOVL	CX, ecx+16(FP)
	MOVL	DX, edx+20(FP)
	RET

TEXT ·inl(SB),NOSPLIT,$0-12
	MOVW	port+0(FP), DX
	INL
	MOVL	AX, ret+8(FP)
	RET

TEXT ·outl(SB),NOSPLIT,$0-8
	MOVW	port+0(FP), DX
	MOVL	b+4(FP), AX
	OUTL
	RET

TEXT ·inb(SB),NOSPLIT,$0-9
	MOVW	port+0(FP), DX
	INB
	MOVB	AX, ret+8(FP)
	RET

TEXT ·outb(SB),NOSPLIT,$0-3
	MOVW	port+0(FP), DX
	MOVB	b+2(FP), AX
	OUTB
	RET

TEXT ·swapgs(SB),NOSPLIT|NOFRAME,$0-0
	SWAPGS
	RET

TEXT ·fninit(SB),NOSPLIT|NOFRAME,$0-0
	BYTE $0xdb; BYTE $0xe3; // FNINIT instruction.
	RET

TEXT ·currentThread(SB),NOSPLIT,$0-8
	MOVQ	CONTEXT_SELF(GS), AX
	MOVQ	AX, ret+0(FP)
	RET

TEXT ·saveThread(SB),NOSPLIT|NOFRAME,$0
	MOVQ	BP, CONTEXT_BP(GS)
	MOVQ	AX, CONTEXT_AX(GS)
	MOVQ	BX, CONTEXT_BX(GS)
	MOVQ	DX, CONTEXT_DX(GS)
	MOVQ	SI, CONTEXT_SI(GS)
	MOVQ	DI, CONTEXT_DI(GS)
	MOVQ	R8, CONTEXT_R8(GS)
	MOVQ	R9, CONTEXT_R9(GS)
	MOVQ	R10, CONTEXT_R10(GS)
	MOVQ	R12, CONTEXT_R12(GS)
	MOVQ	R13, CONTEXT_R13(GS)
	MOVQ	R14, CONTEXT_R14(GS)
	MOVQ	R15, CONTEXT_R15(GS)

	// Save floating point state.
	MOVQ	CONTEXT_SELF(GS), AX
	ADDQ	$CONTEXT_FPSTATE, AX
	FXSAVE (AX)

	MOVQ	CONTEXT_AX(GS), AX

	RET

TEXT ·restoreThread(SB),NOSPLIT|NOFRAME,$0
	// Restore fsbase.
	MOVQ	CONTEXT_FSBASE(GS), AX
	MOVL	$0xc0000100, CX // IA32_FS_BASE
	MOVQ	AX, DX
	SHRQ	$32, DX
	WRMSR

	// Restore floating point state.
	MOVQ	CONTEXT_SELF(GS), AX
	ADDQ	$CONTEXT_FPSTATE, AX
	FXRSTOR	(AX)

	// Restore registers.
	MOVQ	CONTEXT_BP(GS), BP
	MOVQ	CONTEXT_AX(GS), AX
	MOVQ	CONTEXT_BX(GS), BX
	MOVQ	CONTEXT_DX(GS), DX
	MOVQ	CONTEXT_SI(GS), SI
	MOVQ	CONTEXT_DI(GS), DI
	MOVQ	CONTEXT_R8(GS), R8
	MOVQ	CONTEXT_R9(GS), R9
	MOVQ	CONTEXT_R10(GS), R10
	MOVQ	CONTEXT_R12(GS), R12
	MOVQ	CONTEXT_R13(GS), R13
	MOVQ	CONTEXT_R14(GS), R14
	MOVQ	CONTEXT_R15(GS), R15

	RET

TEXT ·resumeThreadFast(SB),NOSPLIT|NOFRAME,$0
	CALL	·restoreThread(SB)

	MOVQ	CONTEXT_IP(GS), CX
	MOVQ	CONTEXT_FLAGS(GS), R11
	MOVQ	CONTEXT_SP(GS), SP
	SWAPGS
	BYTE	$0x48; BYTE $0x0f; BYTE $0x07; // SYSRETQ

TEXT ·resumeThread(SB),NOSPLIT|NOFRAME,$0
	// Create return frame for IRETQ that will restore the stack
	// and instruction pointers and the flags. The frame is 5
	// values, but we need to pop the return address as well.
	SUBQ	$(5-1)*8, SP
	MOVQ	$(4 << 3) | 3, 4*8(SP) // SS = segmentData3 << 3 | ring3
	MOVQ	CONTEXT_SP(GS), AX
	MOVQ	AX, 3*8(SP) // SP = context.sp
	MOVQ	CONTEXT_FLAGS(GS), BX
	MOVQ	BX, 2*8(SP)	// RFLAGS = context.rflags
	MOVQ	$(5 << 3) | 3, 1*8(SP) // CS = segment64Code3 << 3 | ring3
	MOVQ	CONTEXT_IP(GS), CX
	MOVQ	CX, 0*8(SP) // IP = context.ip

	CALL ·restoreThread(SB)

	// Restore remaining registers.
	MOVQ	CONTEXT_CX(GS), CX
	MOVQ	CONTEXT_R11(GS), R11

	SWAPGS
	IRETQ // Resume thread

TEXT ·StoreUint16(SB), NOSPLIT, $0-10
	MOVQ    addr+0(FP), BX
	MOVW    val+8(FP), AX
	XCHGW   AX, 0(BX)
	RET

TEXT ·StoreUint8(SB), NOSPLIT, $0-9
	MOVQ    addr+0(FP), BX
	MOVB    val+8(FP), AX
	XCHGB   AX, 0(BX)
	RET

TEXT ·OrUint8(SB), NOSPLIT, $0-9
	MOVQ    addr+0(FP), AX
	MOVB    val+8(FP), BX
	LOCK
	ORB BX, (AX)
	RET

TEXT ·unknownInterruptTrampoline(SB),NOSPLIT|NOFRAME,$0
	INTERRUPT_SAVE
	CALL ·unknownInterrupt(SB)
	APICEOI
	INTERRUPT_RESTORE
	IRETQ

TEXT ·userInterruptTrampoline0(SB),NOSPLIT|NOFRAME,$0
	USER_TRAMPOLINE(0)
TEXT ·userInterruptTrampoline1(SB),NOSPLIT|NOFRAME,$0
	USER_TRAMPOLINE(1)
TEXT ·userInterruptTrampoline2(SB),NOSPLIT|NOFRAME,$0
	USER_TRAMPOLINE(2)
TEXT ·userInterruptTrampoline3(SB),NOSPLIT|NOFRAME,$0
	USER_TRAMPOLINE(3)
TEXT ·userInterruptTrampoline4(SB),NOSPLIT|NOFRAME,$0
	USER_TRAMPOLINE(4)
TEXT ·userInterruptTrampoline5(SB),NOSPLIT|NOFRAME,$0
	USER_TRAMPOLINE(5)
TEXT ·userInterruptTrampoline6(SB),NOSPLIT|NOFRAME,$0
	USER_TRAMPOLINE(6)
TEXT ·userInterruptTrampoline7(SB),NOSPLIT|NOFRAME,$0
	USER_TRAMPOLINE(7)
TEXT ·userInterruptTrampoline8(SB),NOSPLIT|NOFRAME,$0
	USER_TRAMPOLINE(8)
TEXT ·userInterruptTrampoline9(SB),NOSPLIT|NOFRAME,$0
	USER_TRAMPOLINE(9)
