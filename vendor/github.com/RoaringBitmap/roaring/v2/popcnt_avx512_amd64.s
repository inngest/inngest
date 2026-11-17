//go:build amd64 && !appengine
// +build amd64,!appengine

#include "textflag.h"

// AVX-512 population-count routines for amd64, counterparts to the AVX2 ones
// in popcnt_avx2_amd64.s. They run when the CPU has AVX512_VPOPCNTDQ (see
// useAVX512Popcnt); otherwise the AVX2 or scalar paths are used instead.
//
// Algorithm
// ---------
// AVX2 has no vector population count, so the AVX2 routines build one out of a
// VPSHUFB nibble lookup plus VPSADBW and handle 256 bits per iteration.
// AVX512_VPOPCNTDQ provides VPOPCNTQ, which counts all eight 64-bit lanes of a
// ZMM register in one instruction, so each iteration here handles 2048 bits:
// four ZMM loads, four VPOPCNTQ, four adds.
//
// Four accumulators (Z4..Z7) keep the adds off a single dependency chain; they
// are folded together after the loop and reduced to a scalar by HSUM512. A
// scalar POPCNTQ tail handles the trailing len%32 words, so any slice length is
// counted correctly.
//
// Go assembler conventions are as in popcnt_avx2_amd64.s: operands are written
// source(s) first and destination last, Yn/Xn alias the low halves of Zn, and a
// []uint64 argument is a {ptr,len,cap} header, so a second slice argument
// starts at +24(FP) and the uint64 result follows the arguments.

// HSUM512 horizontally sums the eight 64-bit lanes of Z4 into out.
#define HSUM512(out)             \
	VEXTRACTI64X4 $1, Z4, Y1    \
	VPADDQ        Y1, Y4, Y1    \
	VEXTRACTI128  $1, Y1, X2    \
	VPADDQ        X2, X1, X1    \
	VPSHUFD       $0x4e, X1, X2 \
	VPADDQ        X2, X1, X1    \
	VMOVQ         X1, out

// ZEROACC clears the four accumulators.
#define ZEROACC       \
	VPXORQ Z4, Z4, Z4 \
	VPXORQ Z5, Z5, Z5 \
	VPXORQ Z6, Z6, Z6 \
	VPXORQ Z7, Z7, Z7

// FOLDACC sums the four accumulators into Z4.
#define FOLDACC       \
	VPADDQ Z5, Z4, Z4 \
	VPADDQ Z7, Z6, Z6 \
	VPADDQ Z6, Z4, Z4

// func popcntSliceAVX512(s []uint64) uint64
TEXT ·popcntSliceAVX512(SB), NOSPLIT, $0-32
	MOVQ s_base+0(FP), SI
	MOVQ s_len+8(FP), DX
	ZEROACC
	MOVQ  DX, CX
	SHRQ  $5, CX
	TESTQ CX, CX
	JZ    ps_tail

ps_loop:
	VPOPCNTQ (SI), Z0
	VPOPCNTQ 64(SI), Z1
	VPOPCNTQ 128(SI), Z2
	VPOPCNTQ 192(SI), Z3
	VPADDQ   Z0, Z4, Z4
	VPADDQ   Z1, Z5, Z5
	VPADDQ   Z2, Z6, Z6
	VPADDQ   Z3, Z7, Z7
	ADDQ     $256, SI
	DECQ     CX
	JNZ      ps_loop

ps_tail:
	FOLDACC
	HSUM512(AX)
	ANDQ $31, DX
	JZ   ps_done

ps_scalar:
	POPCNTQ (SI), BX
	ADDQ    BX, AX
	ADDQ    $8, SI
	DECQ    DX
	JNZ     ps_scalar

ps_done:
	VZEROUPPER
	MOVQ AX, ret+24(FP)
	RET

// func popcntAndSliceAVX512(s, m []uint64) uint64
TEXT ·popcntAndSliceAVX512(SB), NOSPLIT, $0-56
	MOVQ s_base+0(FP), SI
	MOVQ s_len+8(FP), DX
	MOVQ m_base+24(FP), DI
	ZEROACC
	MOVQ  DX, CX
	SHRQ  $5, CX
	TESTQ CX, CX
	JZ    pa_tail

pa_loop:
	VMOVDQU64 (SI), Z0
	VMOVDQU64 64(SI), Z1
	VMOVDQU64 128(SI), Z2
	VMOVDQU64 192(SI), Z3
	VPANDQ    (DI), Z0, Z0
	VPANDQ    64(DI), Z1, Z1
	VPANDQ    128(DI), Z2, Z2
	VPANDQ    192(DI), Z3, Z3
	VPOPCNTQ  Z0, Z0
	VPOPCNTQ  Z1, Z1
	VPOPCNTQ  Z2, Z2
	VPOPCNTQ  Z3, Z3
	VPADDQ    Z0, Z4, Z4
	VPADDQ    Z1, Z5, Z5
	VPADDQ    Z2, Z6, Z6
	VPADDQ    Z3, Z7, Z7
	ADDQ      $256, SI
	ADDQ      $256, DI
	DECQ      CX
	JNZ       pa_loop

pa_tail:
	FOLDACC
	HSUM512(AX)
	ANDQ $31, DX
	JZ   pa_done

pa_scalar:
	MOVQ    (SI), BX
	ANDQ    (DI), BX
	POPCNTQ BX, BX
	ADDQ    BX, AX
	ADDQ    $8, SI
	ADDQ    $8, DI
	DECQ    DX
	JNZ     pa_scalar

pa_done:
	VZEROUPPER
	MOVQ AX, ret+48(FP)
	RET

// func popcntOrSliceAVX512(s, m []uint64) uint64
TEXT ·popcntOrSliceAVX512(SB), NOSPLIT, $0-56
	MOVQ s_base+0(FP), SI
	MOVQ s_len+8(FP), DX
	MOVQ m_base+24(FP), DI
	ZEROACC
	MOVQ  DX, CX
	SHRQ  $5, CX
	TESTQ CX, CX
	JZ    po_tail

po_loop:
	VMOVDQU64 (SI), Z0
	VMOVDQU64 64(SI), Z1
	VMOVDQU64 128(SI), Z2
	VMOVDQU64 192(SI), Z3
	VPORQ     (DI), Z0, Z0
	VPORQ     64(DI), Z1, Z1
	VPORQ     128(DI), Z2, Z2
	VPORQ     192(DI), Z3, Z3
	VPOPCNTQ  Z0, Z0
	VPOPCNTQ  Z1, Z1
	VPOPCNTQ  Z2, Z2
	VPOPCNTQ  Z3, Z3
	VPADDQ    Z0, Z4, Z4
	VPADDQ    Z1, Z5, Z5
	VPADDQ    Z2, Z6, Z6
	VPADDQ    Z3, Z7, Z7
	ADDQ      $256, SI
	ADDQ      $256, DI
	DECQ      CX
	JNZ       po_loop

po_tail:
	FOLDACC
	HSUM512(AX)
	ANDQ $31, DX
	JZ   po_done

po_scalar:
	MOVQ    (SI), BX
	ORQ     (DI), BX
	POPCNTQ BX, BX
	ADDQ    BX, AX
	ADDQ    $8, SI
	ADDQ    $8, DI
	DECQ    DX
	JNZ     po_scalar

po_done:
	VZEROUPPER
	MOVQ AX, ret+48(FP)
	RET

// func popcntXorSliceAVX512(s, m []uint64) uint64
TEXT ·popcntXorSliceAVX512(SB), NOSPLIT, $0-56
	MOVQ s_base+0(FP), SI
	MOVQ s_len+8(FP), DX
	MOVQ m_base+24(FP), DI
	ZEROACC
	MOVQ  DX, CX
	SHRQ  $5, CX
	TESTQ CX, CX
	JZ    px_tail

px_loop:
	VMOVDQU64 (SI), Z0
	VMOVDQU64 64(SI), Z1
	VMOVDQU64 128(SI), Z2
	VMOVDQU64 192(SI), Z3
	VPXORQ    (DI), Z0, Z0
	VPXORQ    64(DI), Z1, Z1
	VPXORQ    128(DI), Z2, Z2
	VPXORQ    192(DI), Z3, Z3
	VPOPCNTQ  Z0, Z0
	VPOPCNTQ  Z1, Z1
	VPOPCNTQ  Z2, Z2
	VPOPCNTQ  Z3, Z3
	VPADDQ    Z0, Z4, Z4
	VPADDQ    Z1, Z5, Z5
	VPADDQ    Z2, Z6, Z6
	VPADDQ    Z3, Z7, Z7
	ADDQ      $256, SI
	ADDQ      $256, DI
	DECQ      CX
	JNZ       px_loop

px_tail:
	FOLDACC
	HSUM512(AX)
	ANDQ $31, DX
	JZ   px_done

px_scalar:
	MOVQ    (SI), BX
	XORQ    (DI), BX
	POPCNTQ BX, BX
	ADDQ    BX, AX
	ADDQ    $8, SI
	ADDQ    $8, DI
	DECQ    DX
	JNZ     px_scalar

px_done:
	VZEROUPPER
	MOVQ AX, ret+48(FP)
	RET

// func popcntMaskSliceAVX512(s, m []uint64) uint64
// Returns the sum of popcount(s[i] &^ m[i]). As in the AVX2 routine, VPANDN
// negates its first source and only the second may come from memory, so m goes
// into the register and s is read from memory: "VPANDNQ (SI), Zm, Zm" gives
// (NOT m) AND s.
TEXT ·popcntMaskSliceAVX512(SB), NOSPLIT, $0-56
	MOVQ s_base+0(FP), SI
	MOVQ s_len+8(FP), DX
	MOVQ m_base+24(FP), DI
	ZEROACC
	MOVQ  DX, CX
	SHRQ  $5, CX
	TESTQ CX, CX
	JZ    pm_tail

pm_loop:
	VMOVDQU64 (DI), Z0
	VMOVDQU64 64(DI), Z1
	VMOVDQU64 128(DI), Z2
	VMOVDQU64 192(DI), Z3
	VPANDNQ   (SI), Z0, Z0
	VPANDNQ   64(SI), Z1, Z1
	VPANDNQ   128(SI), Z2, Z2
	VPANDNQ   192(SI), Z3, Z3
	VPOPCNTQ  Z0, Z0
	VPOPCNTQ  Z1, Z1
	VPOPCNTQ  Z2, Z2
	VPOPCNTQ  Z3, Z3
	VPADDQ    Z0, Z4, Z4
	VPADDQ    Z1, Z5, Z5
	VPADDQ    Z2, Z6, Z6
	VPADDQ    Z3, Z7, Z7
	ADDQ      $256, SI
	ADDQ      $256, DI
	DECQ      CX
	JNZ       pm_loop

pm_tail:
	FOLDACC
	HSUM512(AX)
	ANDQ $31, DX
	JZ   pm_done

pm_scalar:
	MOVQ    (DI), BX
	NOTQ    BX
	ANDQ    (SI), BX
	POPCNTQ BX, BX
	ADDQ    BX, AX
	ADDQ    $8, SI
	ADDQ    $8, DI
	DECQ    DX
	JNZ     pm_scalar

pm_done:
	VZEROUPPER
	MOVQ AX, ret+48(FP)
	RET
