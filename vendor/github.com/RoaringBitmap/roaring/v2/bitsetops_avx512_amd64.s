//go:build amd64 && !appengine
// +build amd64,!appengine

#include "textflag.h"

// AVX-512 word operations on bitmap containers.
//
// Each routine computes dst[i] = a[i] op b[i] over a whole container, eight
// words at a time. The Card variants additionally return the population count
// of the result using VPOPCNTQ, so a caller that needs both the result and its
// cardinality makes a single pass over the container instead of two.
//
// dst may alias a or b: within an iteration the sources are loaded before the
// destination is stored, and iterations never touch each other's words.
//
// Each iteration handles four ZMM registers, i.e. 32 words; a scalar tail
// handles the trailing len%32 words, so any slice length works. The Card
// variants use four accumulators to keep the adds off one dependency chain.
//
// Go assembler conventions are as in popcnt_avx2_amd64.s: operands are written
// source(s) first and destination last, and a []uint64 argument is a
// {ptr,len,cap} header, so the second and third slices start at +24(FP) and
// +48(FP), and any result follows the arguments.

// HSUM512 horizontally sums the eight 64-bit lanes of Z4 into out.
#define HSUM512(out)             \
	VEXTRACTI64X4 $1, Z4, Y1    \
	VPADDQ        Y1, Y4, Y1    \
	VEXTRACTI128  $1, Y1, X2    \
	VPADDQ        X2, X1, X1    \
	VPSHUFD       $0x4e, X1, X2 \
	VPADDQ        X2, X1, X1    \
	VMOVQ         X1, out

// func orSliceAVX512(dst, a, b []uint64)
TEXT ·orSliceAVX512(SB), NOSPLIT, $0-72
	MOVQ dst_base+0(FP), DX
	MOVQ a_base+24(FP), SI
	MOVQ a_len+32(FP), BX
	MOVQ b_base+48(FP), DI
	MOVQ  BX, CX
	SHRQ  $5, CX
	TESTQ CX, CX
	JZ    or_tail

or_loop:
	VMOVDQU64 (SI), Z0
	VMOVDQU64 64(SI), Z1
	VMOVDQU64 128(SI), Z2
	VMOVDQU64 192(SI), Z3
	VPORQ (DI), Z0, Z0
	VPORQ 64(DI), Z1, Z1
	VPORQ 128(DI), Z2, Z2
	VPORQ 192(DI), Z3, Z3
	VMOVDQU64 Z0, (DX)
	VMOVDQU64 Z1, 64(DX)
	VMOVDQU64 Z2, 128(DX)
	VMOVDQU64 Z3, 192(DX)
	ADDQ $256, SI
	ADDQ $256, DI
	ADDQ $256, DX
	DECQ CX
	JNZ  or_loop

or_tail:
	ANDQ $31, BX
	JZ   or_done

or_scalar:
	MOVQ (SI), R9
	ORQ (DI), R9
	MOVQ R9, (DX)
	ADDQ $8, SI
	ADDQ $8, DI
	ADDQ $8, DX
	DECQ BX
	JNZ  or_scalar

or_done:
	VZEROUPPER
	RET

// func andSliceAVX512(dst, a, b []uint64)
TEXT ·andSliceAVX512(SB), NOSPLIT, $0-72
	MOVQ dst_base+0(FP), DX
	MOVQ a_base+24(FP), SI
	MOVQ a_len+32(FP), BX
	MOVQ b_base+48(FP), DI
	MOVQ  BX, CX
	SHRQ  $5, CX
	TESTQ CX, CX
	JZ    and_tail

and_loop:
	VMOVDQU64 (SI), Z0
	VMOVDQU64 64(SI), Z1
	VMOVDQU64 128(SI), Z2
	VMOVDQU64 192(SI), Z3
	VPANDQ (DI), Z0, Z0
	VPANDQ 64(DI), Z1, Z1
	VPANDQ 128(DI), Z2, Z2
	VPANDQ 192(DI), Z3, Z3
	VMOVDQU64 Z0, (DX)
	VMOVDQU64 Z1, 64(DX)
	VMOVDQU64 Z2, 128(DX)
	VMOVDQU64 Z3, 192(DX)
	ADDQ $256, SI
	ADDQ $256, DI
	ADDQ $256, DX
	DECQ CX
	JNZ  and_loop

and_tail:
	ANDQ $31, BX
	JZ   and_done

and_scalar:
	MOVQ (SI), R9
	ANDQ (DI), R9
	MOVQ R9, (DX)
	ADDQ $8, SI
	ADDQ $8, DI
	ADDQ $8, DX
	DECQ BX
	JNZ  and_scalar

and_done:
	VZEROUPPER
	RET

// func xorSliceAVX512(dst, a, b []uint64)
TEXT ·xorSliceAVX512(SB), NOSPLIT, $0-72
	MOVQ dst_base+0(FP), DX
	MOVQ a_base+24(FP), SI
	MOVQ a_len+32(FP), BX
	MOVQ b_base+48(FP), DI
	MOVQ  BX, CX
	SHRQ  $5, CX
	TESTQ CX, CX
	JZ    xor_tail

xor_loop:
	VMOVDQU64 (SI), Z0
	VMOVDQU64 64(SI), Z1
	VMOVDQU64 128(SI), Z2
	VMOVDQU64 192(SI), Z3
	VPXORQ (DI), Z0, Z0
	VPXORQ 64(DI), Z1, Z1
	VPXORQ 128(DI), Z2, Z2
	VPXORQ 192(DI), Z3, Z3
	VMOVDQU64 Z0, (DX)
	VMOVDQU64 Z1, 64(DX)
	VMOVDQU64 Z2, 128(DX)
	VMOVDQU64 Z3, 192(DX)
	ADDQ $256, SI
	ADDQ $256, DI
	ADDQ $256, DX
	DECQ CX
	JNZ  xor_loop

xor_tail:
	ANDQ $31, BX
	JZ   xor_done

xor_scalar:
	MOVQ (SI), R9
	XORQ (DI), R9
	MOVQ R9, (DX)
	ADDQ $8, SI
	ADDQ $8, DI
	ADDQ $8, DX
	DECQ BX
	JNZ  xor_scalar

xor_done:
	VZEROUPPER
	RET

// func andNotSliceAVX512(dst, a, b []uint64)
// VPANDN negates its first source and only the second may come from
// memory, so b goes into the register and a is read from memory:
// "VPANDNQ (SI), Zb, Zb" gives (NOT b) AND a = a &^ b.
TEXT ·andNotSliceAVX512(SB), NOSPLIT, $0-72
	MOVQ dst_base+0(FP), DX
	MOVQ a_base+24(FP), SI
	MOVQ a_len+32(FP), BX
	MOVQ b_base+48(FP), DI
	MOVQ  BX, CX
	SHRQ  $5, CX
	TESTQ CX, CX
	JZ    andnot_tail

andnot_loop:
	VMOVDQU64 (DI), Z0
	VMOVDQU64 64(DI), Z1
	VMOVDQU64 128(DI), Z2
	VMOVDQU64 192(DI), Z3
	VPANDNQ (SI), Z0, Z0
	VPANDNQ 64(SI), Z1, Z1
	VPANDNQ 128(SI), Z2, Z2
	VPANDNQ 192(SI), Z3, Z3
	VMOVDQU64 Z0, (DX)
	VMOVDQU64 Z1, 64(DX)
	VMOVDQU64 Z2, 128(DX)
	VMOVDQU64 Z3, 192(DX)
	ADDQ $256, SI
	ADDQ $256, DI
	ADDQ $256, DX
	DECQ CX
	JNZ  andnot_loop

andnot_tail:
	ANDQ $31, BX
	JZ   andnot_done

andnot_scalar:
	MOVQ (DI), R9
	NOTQ R9
	ANDQ (SI), R9
	MOVQ R9, (DX)
	ADDQ $8, SI
	ADDQ $8, DI
	ADDQ $8, DX
	DECQ BX
	JNZ  andnot_scalar

andnot_done:
	VZEROUPPER
	RET

// func orCardSliceAVX512(dst, a, b []uint64) uint64
TEXT ·orCardSliceAVX512(SB), NOSPLIT, $0-80
	MOVQ dst_base+0(FP), DX
	MOVQ a_base+24(FP), SI
	MOVQ a_len+32(FP), BX
	MOVQ b_base+48(FP), DI
	VPXORQ Z4, Z4, Z4
	VPXORQ Z5, Z5, Z5
	VPXORQ Z6, Z6, Z6
	VPXORQ Z7, Z7, Z7
	MOVQ  BX, CX
	SHRQ  $5, CX
	TESTQ CX, CX
	JZ    orc_tail

orc_loop:
	VMOVDQU64 (SI), Z0
	VMOVDQU64 64(SI), Z1
	VMOVDQU64 128(SI), Z2
	VMOVDQU64 192(SI), Z3
	VPORQ (DI), Z0, Z0
	VMOVDQU64 Z0, (DX)
	VPOPCNTQ Z0, Z0
	VPADDQ Z0, Z4, Z4
	VPORQ 64(DI), Z1, Z1
	VMOVDQU64 Z1, 64(DX)
	VPOPCNTQ Z1, Z1
	VPADDQ Z1, Z5, Z5
	VPORQ 128(DI), Z2, Z2
	VMOVDQU64 Z2, 128(DX)
	VPOPCNTQ Z2, Z2
	VPADDQ Z2, Z6, Z6
	VPORQ 192(DI), Z3, Z3
	VMOVDQU64 Z3, 192(DX)
	VPOPCNTQ Z3, Z3
	VPADDQ Z3, Z7, Z7
	ADDQ $256, SI
	ADDQ $256, DI
	ADDQ $256, DX
	DECQ CX
	JNZ  orc_loop

orc_tail:
	VPADDQ Z5, Z4, Z4
	VPADDQ Z7, Z6, Z6
	VPADDQ Z6, Z4, Z4
	HSUM512(AX)
	ANDQ $31, BX
	JZ   orc_done

orc_scalar:
	MOVQ (SI), R9
	ORQ (DI), R9
	MOVQ R9, (DX)
	POPCNTQ R9, R9
	ADDQ R9, AX
	ADDQ $8, SI
	ADDQ $8, DI
	ADDQ $8, DX
	DECQ BX
	JNZ  orc_scalar

orc_done:
	VZEROUPPER
	MOVQ AX, ret+72(FP)
	RET

// func andCardSliceAVX512(dst, a, b []uint64) uint64
TEXT ·andCardSliceAVX512(SB), NOSPLIT, $0-80
	MOVQ dst_base+0(FP), DX
	MOVQ a_base+24(FP), SI
	MOVQ a_len+32(FP), BX
	MOVQ b_base+48(FP), DI
	VPXORQ Z4, Z4, Z4
	VPXORQ Z5, Z5, Z5
	VPXORQ Z6, Z6, Z6
	VPXORQ Z7, Z7, Z7
	MOVQ  BX, CX
	SHRQ  $5, CX
	TESTQ CX, CX
	JZ    andc_tail

andc_loop:
	VMOVDQU64 (SI), Z0
	VMOVDQU64 64(SI), Z1
	VMOVDQU64 128(SI), Z2
	VMOVDQU64 192(SI), Z3
	VPANDQ (DI), Z0, Z0
	VMOVDQU64 Z0, (DX)
	VPOPCNTQ Z0, Z0
	VPADDQ Z0, Z4, Z4
	VPANDQ 64(DI), Z1, Z1
	VMOVDQU64 Z1, 64(DX)
	VPOPCNTQ Z1, Z1
	VPADDQ Z1, Z5, Z5
	VPANDQ 128(DI), Z2, Z2
	VMOVDQU64 Z2, 128(DX)
	VPOPCNTQ Z2, Z2
	VPADDQ Z2, Z6, Z6
	VPANDQ 192(DI), Z3, Z3
	VMOVDQU64 Z3, 192(DX)
	VPOPCNTQ Z3, Z3
	VPADDQ Z3, Z7, Z7
	ADDQ $256, SI
	ADDQ $256, DI
	ADDQ $256, DX
	DECQ CX
	JNZ  andc_loop

andc_tail:
	VPADDQ Z5, Z4, Z4
	VPADDQ Z7, Z6, Z6
	VPADDQ Z6, Z4, Z4
	HSUM512(AX)
	ANDQ $31, BX
	JZ   andc_done

andc_scalar:
	MOVQ (SI), R9
	ANDQ (DI), R9
	MOVQ R9, (DX)
	POPCNTQ R9, R9
	ADDQ R9, AX
	ADDQ $8, SI
	ADDQ $8, DI
	ADDQ $8, DX
	DECQ BX
	JNZ  andc_scalar

andc_done:
	VZEROUPPER
	MOVQ AX, ret+72(FP)
	RET
