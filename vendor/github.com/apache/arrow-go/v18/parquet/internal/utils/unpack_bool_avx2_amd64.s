//+build !noasm !appengine

// AVX2 vectorized bytes-to-bools using VPBROADCASTD + VPSHUFB + VPCMPEQB.
// Processes 4 input bytes → 32 output bools per vector iteration.
// Replaces the original c2goasm-generated scalar code which used zero SIMD.

#include "textflag.h"

// VPSHUFB operates on two independent 128-bit lanes.
// Lower lane: byte 0 → lanes 0-7, byte 1 → lanes 8-15
// Upper lane: byte 2 → lanes 16-23, byte 3 → lanes 24-31
DATA shuffle_avx2<>+0x00(SB)/8, $0x0000000000000000
DATA shuffle_avx2<>+0x08(SB)/8, $0x0101010101010101
DATA shuffle_avx2<>+0x10(SB)/8, $0x0202020202020202
DATA shuffle_avx2<>+0x18(SB)/8, $0x0303030303030303
GLOBL shuffle_avx2<>(SB), (NOPTR+RODATA), $32

// [1, 2, 4, 8, 16, 32, 64, 128] × 4
DATA bitmask_avx2<>+0x00(SB)/8, $0x8040201008040201
DATA bitmask_avx2<>+0x08(SB)/8, $0x8040201008040201
DATA bitmask_avx2<>+0x10(SB)/8, $0x8040201008040201
DATA bitmask_avx2<>+0x18(SB)/8, $0x8040201008040201
GLOBL bitmask_avx2<>(SB), (NOPTR+RODATA), $32

// [1, 1, 1, ...] × 32
DATA ones_avx2<>+0x00(SB)/8, $0x0101010101010101
DATA ones_avx2<>+0x08(SB)/8, $0x0101010101010101
DATA ones_avx2<>+0x10(SB)/8, $0x0101010101010101
DATA ones_avx2<>+0x18(SB)/8, $0x0101010101010101
GLOBL ones_avx2<>(SB), (NOPTR+RODATA), $32

TEXT ·_bytes_to_bools_avx2(SB), NOSPLIT, $0-32

	MOVQ in+0(FP), DI
	MOVQ len+8(FP), SI
	MOVQ out+16(FP), DX
	MOVQ outlen+24(FP), R13

	TESTL SI, SI
	JLE  done

	VMOVDQU shuffle_avx2<>(SB), Y3
	VMOVDQU bitmask_avx2<>(SB), Y4
	VMOVDQU ones_avx2<>(SB), Y5

	XORQ R8, R8
	XORQ R9, R9

loop32:
	MOVQ SI, AX
	SUBQ R8, AX
	CMPQ AX, $4
	JL   loop8

	MOVQ R13, AX
	SUBQ R9, AX
	CMPQ AX, $32
	JL   loop8

	MOVL (DI)(R8*1), AX
	MOVD AX, X0
	VPBROADCASTD X0, Y0
	VPSHUFB Y3, Y0, Y0
	VPAND Y4, Y0, Y1
	VPCMPEQB Y4, Y1, Y1
	VPAND Y5, Y1, Y1
	VMOVDQU Y1, (DX)(R9*1)

	ADDQ $4, R8
	ADDQ $32, R9
	JMP  loop32

loop8:
	CMPQ R8, SI
	JGE  avx_done

	MOVQ R13, AX
	SUBQ R9, AX
	CMPQ AX, $8
	JL   scalar

	MOVBLZX (DI)(R8*1), AX
	MOVD AX, X0
	VPBROADCASTD X0, Y0
	VPSHUFB Y3, Y0, Y0
	VPAND Y4, Y0, Y1
	VPCMPEQB Y4, Y1, Y1
	VPAND Y5, Y1, Y1
	MOVQ X1, (DX)(R9*1)

	ADDQ $1, R8
	ADDQ $8, R9
	JMP  loop8

scalar:
	CMPQ R8, SI
	JGE  avx_done
	CMPQ R9, R13
	JGE  avx_done

	MOVBLZX (DI)(R8*1), AX
	XORQ CX, CX

scalar_bit:
	CMPQ CX, $8
	JGE  scalar_next
	CMPQ R9, R13
	JGE  avx_done

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

avx_done:
	VZEROUPPER

done:
	RET
