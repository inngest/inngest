//go:build amd64 && !appengine

#include "textflag.h"

// Byte values 0x00..0x3f: the bit positions inside one 64-bit word.
DATA bitPositions<>+0x00(SB)/8, $0x0706050403020100
DATA bitPositions<>+0x08(SB)/8, $0x0f0e0d0c0b0a0908
DATA bitPositions<>+0x10(SB)/8, $0x1716151413121110
DATA bitPositions<>+0x18(SB)/8, $0x1f1e1d1c1b1a1918
DATA bitPositions<>+0x20(SB)/8, $0x2726252423222120
DATA bitPositions<>+0x28(SB)/8, $0x2f2e2d2c2b2a2928
DATA bitPositions<>+0x30(SB)/8, $0x3736353433323130
DATA bitPositions<>+0x38(SB)/8, $0x3f3e3d3c3b3a3938
GLOBL bitPositions<>(SB), RODATA|NOPTR, $64

// func fillLeastSignificant16bitsVector(bitmap []uint64, x []uint32, pos int, mask uint32) int
//
// One VPCOMPRESSB per 64-bit word turns the whole word into 64 byte-sized bit
// positions in a single shot; the positions are then widened 16 at a time with
// VPMOVZXBD and added to the running base.
//
// The compress-then-widen idea is taken from simdjson's bit_indexer::write
// (icelake kernel, Apache-2.0): https://github.com/simdjson/simdjson
//
// It differs here in the stores. simdjson writes whole 64-byte blocks and
// relies on its output buffer having up to 64 uint32 of slack. ToArray
// allocates exactly GetCardinality() values, with no padding, so each block is
// written with a mask derived from BZHI(-1, popcount) instead. That keeps the
// writes exactly popcount(word) wide, and it is also faster on dense
// containers because no store bandwidth is spent on values nobody asked for.
TEXT ·fillLeastSignificant16bitsVector(SB), NOSPLIT, $0-72
	MOVQ bitmap_base+0(FP), SI
	MOVQ bitmap_len+8(FP), CX
	MOVQ x_base+24(FP), DI
	MOVQ pos+48(FP), R8
	LEAQ (DI)(R8*4), DI
	MOVL mask+56(FP), R9

	VMOVDQU64 bitPositions<>(SB), Z0
	VPBROADCASTD R9, Z1
	MOVL $64, AX
	VPBROADCASTD AX, Z3

	TESTQ CX, CX
	JZ    done

loop:
	MOVQ (SI), AX
	TESTQ AX, AX
	JZ    next

	KMOVQ       AX, K1
	VPCOMPRESSB.Z Z0, K1, Z2
	POPCNTQ     AX, R10
	MOVQ        $-1, R11
	BZHIQ       R10, R11, R11

	KMOVW     R11, K2
	VPMOVZXBD X2, Z4
	VPADDD    Z1, Z4, Z4
	VMOVDQU32 Z4, K2, (DI)

	CMPQ R10, $16
	JLE  advance

	SHRQ          $16, R11
	KMOVW         R11, K2
	VEXTRACTI32X4 $1, Z2, X5
	VPMOVZXBD     X5, Z4
	VPADDD        Z1, Z4, Z4
	VMOVDQU32     Z4, K2, 64(DI)

	CMPQ R10, $32
	JLE  advance

	SHRQ          $16, R11
	KMOVW         R11, K2
	VEXTRACTI32X4 $2, Z2, X5
	VPMOVZXBD     X5, Z4
	VPADDD        Z1, Z4, Z4
	VMOVDQU32     Z4, K2, 128(DI)

	CMPQ R10, $48
	JLE  advance

	SHRQ          $16, R11
	KMOVW         R11, K2
	VEXTRACTI32X4 $3, Z2, X5
	VPMOVZXBD     X5, Z4
	VPADDD        Z1, Z4, Z4
	VMOVDQU32     Z4, K2, 192(DI)

advance:
	LEAQ (DI)(R10*4), DI
	ADDQ R10, R8

next:
	VPADDD Z3, Z1, Z1
	ADDQ   $8, SI
	DECQ   CX
	JNZ    loop

done:
	VZEROUPPER
	MOVQ R8, ret+64(FP)
	RET
