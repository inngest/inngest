//+build !noasm !appengine

// SSE4 vectorized bytes-to-bools using PSHUFB broadcast + PCMPEQB bit-test.
// Processes 2 input bytes → 16 output bools per vector iteration.
// Replaces the original c2goasm-generated scalar code which used zero SIMD.

#include "textflag.h"

// broadcast byte 0 → lanes 0-7, byte 1 → lanes 8-15
DATA shuffle_sse4<>+0x00(SB)/8, $0x0000000000000000
DATA shuffle_sse4<>+0x08(SB)/8, $0x0101010101010101
GLOBL shuffle_sse4<>(SB), (NOPTR+RODATA), $16

// [1, 2, 4, 8, 16, 32, 64, 128] × 2
DATA bitmask_sse4<>+0x00(SB)/8, $0x8040201008040201
DATA bitmask_sse4<>+0x08(SB)/8, $0x8040201008040201
GLOBL bitmask_sse4<>(SB), (NOPTR+RODATA), $16

// [1, 1, 1, ...] × 16
DATA ones_sse4<>+0x00(SB)/8, $0x0101010101010101
DATA ones_sse4<>+0x08(SB)/8, $0x0101010101010101
GLOBL ones_sse4<>(SB), (NOPTR+RODATA), $16

TEXT ·_bytes_to_bools_sse4(SB), NOSPLIT, $0-32

	MOVQ in+0(FP), DI
	MOVQ len+8(FP), SI
	MOVQ out+16(FP), DX
	MOVQ outlen+24(FP), R13

	TESTL SI, SI
	JLE  done

	MOVOU shuffle_sse4<>(SB), X3
	MOVOU bitmask_sse4<>(SB), X4
	MOVOU ones_sse4<>(SB), X5

	XORQ R8, R8
	XORQ R9, R9

loop16:
	MOVQ SI, AX
	SUBQ R8, AX
	CMPQ AX, $2
	JL   loop8

	MOVQ R13, AX
	SUBQ R9, AX
	CMPQ AX, $16
	JL   loop8

	MOVWLZX (DI)(R8*1), AX
	MOVD AX, X0
	PSHUFB X3, X0
	MOVOU X0, X1
	PAND  X4, X1
	PCMPEQB X4, X1
	PAND X5, X1
	MOVOU X1, (DX)(R9*1)

	ADDQ $2, R8
	ADDQ $16, R9
	JMP  loop16

loop8:
	CMPQ R8, SI
	JGE  done

	MOVQ R13, AX
	SUBQ R9, AX
	CMPQ AX, $8
	JL   scalar

	MOVBLZX (DI)(R8*1), AX
	MOVD AX, X0
	PXOR X6, X6
	PSHUFB X6, X0
	PAND  X4, X0
	PCMPEQB X4, X0
	PAND  X5, X0
	MOVQ X0, (DX)(R9*1)

	ADDQ $1, R8
	ADDQ $8, R9
	JMP  loop8

scalar:
	CMPQ R8, SI
	JGE  done
	CMPQ R9, R13
	JGE  done

	MOVBLZX (DI)(R8*1), AX
	XORQ CX, CX

scalar_bit:
	CMPQ CX, $8
	JGE  scalar_next
	CMPQ R9, R13
	JGE  done

	MOVL AX, R11
	SHRL CL, R11
	ANDL $1, R11
	MOVB R11, (DX)(R9*1)

	INCQ CX
	INCQ R9
	JMP  scalar_bit

scalar_next:
	INCQ R8
	JMP  scalar

done:
	RET
