// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build !noasm && !appengine
// +build !noasm,!appengine

// AVX2 implementation of BYTE_STREAM_SPLIT encoding.

#include "textflag.h"

// func _encodeByteStreamSplitWidth4AVX2(in, out unsafe.Pointer, nValues int)
//
// Transposes eight 4-byte values at a time. The unpack stages produce two
// 8-byte streams per output register.
TEXT ·_encodeByteStreamSplitWidth4AVX2(SB), NOSPLIT, $0-24
	MOVQ in+0(FP), SI
	MOVQ out+8(FP), DI
	MOVQ nValues+16(FP), CX
	TESTQ CX, CX
	JZ encode_w4_done

	MOVQ CX, AX
	SHRQ $3, AX
	LEAQ (DI)(CX*1), R8
	LEAQ (DI)(CX*2), R9
	LEAQ (R8)(CX*2), R10
	MOVQ AX, R11
	SHLQ $3, R11
	SUBQ R11, CX

	TESTQ AX, AX
	JZ encode_w4_tail

encode_w4_vector:
	VMOVD 0(SI), X0
	VMOVD 4(SI), X1
	VMOVD 8(SI), X2
	VMOVD 12(SI), X3
	VMOVD 16(SI), X4
	VMOVD 20(SI), X5
	VMOVD 24(SI), X6
	VMOVD 28(SI), X7

	VPUNPCKLBW X1, X0, X8
	VPUNPCKLBW X3, X2, X9
	VPUNPCKLBW X5, X4, X10
	VPUNPCKLBW X7, X6, X11

	VPUNPCKLWD X9, X8, X12
	VPUNPCKLWD X11, X10, X13

	VPUNPCKLDQ X13, X12, X0
	VPUNPCKHDQ X13, X12, X1

	VMOVQ X0, (DI)
	VPSRLDQ $8, X0, X0
	VMOVQ X0, (R8)
	VMOVQ X1, (R9)
	VPSRLDQ $8, X1, X1
	VMOVQ X1, (R10)

	ADDQ $32, SI
	ADDQ $8, DI
	ADDQ $8, R8
	ADDQ $8, R9
	ADDQ $8, R10
	DECQ AX
	JNZ encode_w4_vector

encode_w4_tail:
	TESTQ CX, CX
	JZ encode_w4_done

encode_w4_tail_loop:
	MOVBQZX 0(SI), BX
	MOVB BX, (DI)
	MOVBQZX 1(SI), BX
	MOVB BX, (R8)
	MOVBQZX 2(SI), BX
	MOVB BX, (R9)
	MOVBQZX 3(SI), BX
	MOVB BX, (R10)

	ADDQ $4, SI
	INCQ DI
	INCQ R8
	INCQ R9
	INCQ R10
	DECQ CX
	JNZ encode_w4_tail_loop

encode_w4_done:
	VZEROUPPER
	RET

// func _encodeByteStreamSplitWidth8AVX2(in, out unsafe.Pointer, nValues int)
//
// Transposes eight 8-byte values at a time. The unpack stages produce two
// 8-byte streams per output register.
TEXT ·_encodeByteStreamSplitWidth8AVX2(SB), NOSPLIT, $0-24
	MOVQ in+0(FP), SI
	MOVQ out+8(FP), DI
	MOVQ nValues+16(FP), CX
	TESTQ CX, CX
	JZ encode_w8_done

	MOVQ CX, AX
	SHRQ $3, AX
	LEAQ (DI)(CX*1), R8
	LEAQ (DI)(CX*2), R9
	LEAQ (R8)(CX*2), R10
	LEAQ (DI)(CX*4), R11
	LEAQ (R8)(CX*4), R12
	LEAQ (R9)(CX*4), R13
	LEAQ (R10)(CX*4), R14
	MOVQ AX, R15
	SHLQ $3, R15
	SUBQ R15, CX

	TESTQ AX, AX
	JZ encode_w8_tail

encode_w8_vector:
	VMOVQ 0(SI), X0
	VMOVQ 8(SI), X1
	VMOVQ 16(SI), X2
	VMOVQ 24(SI), X3
	VMOVQ 32(SI), X4
	VMOVQ 40(SI), X5
	VMOVQ 48(SI), X6
	VMOVQ 56(SI), X7

	VPUNPCKLBW X1, X0, X8
	VPUNPCKLBW X3, X2, X9
	VPUNPCKLBW X5, X4, X10
	VPUNPCKLBW X7, X6, X11

	VPUNPCKLWD X9, X8, X12
	VPUNPCKHWD X9, X8, X13
	VPUNPCKLWD X11, X10, X14
	VPUNPCKHWD X11, X10, X15

	VPUNPCKLDQ X14, X12, X0
	VPUNPCKHDQ X14, X12, X1
	VPUNPCKLDQ X15, X13, X2
	VPUNPCKHDQ X15, X13, X3

	VMOVQ X0, (DI)
	VPSRLDQ $8, X0, X0
	VMOVQ X0, (R8)
	VMOVQ X1, (R9)
	VPSRLDQ $8, X1, X1
	VMOVQ X1, (R10)
	VMOVQ X2, (R11)
	VPSRLDQ $8, X2, X2
	VMOVQ X2, (R12)
	VMOVQ X3, (R13)
	VPSRLDQ $8, X3, X3
	VMOVQ X3, (R14)

	ADDQ $64, SI
	ADDQ $8, DI
	ADDQ $8, R8
	ADDQ $8, R9
	ADDQ $8, R10
	ADDQ $8, R11
	ADDQ $8, R12
	ADDQ $8, R13
	ADDQ $8, R14
	DECQ AX
	JNZ encode_w8_vector

encode_w8_tail:
	TESTQ CX, CX
	JZ encode_w8_done

encode_w8_tail_loop:
	MOVBQZX 0(SI), BX
	MOVB BX, (DI)
	MOVBQZX 1(SI), BX
	MOVB BX, (R8)
	MOVBQZX 2(SI), BX
	MOVB BX, (R9)
	MOVBQZX 3(SI), BX
	MOVB BX, (R10)
	MOVBQZX 4(SI), BX
	MOVB BX, (R11)
	MOVBQZX 5(SI), BX
	MOVB BX, (R12)
	MOVBQZX 6(SI), BX
	MOVB BX, (R13)
	MOVBQZX 7(SI), BX
	MOVB BX, (R14)

	ADDQ $8, SI
	INCQ DI
	INCQ R8
	INCQ R9
	INCQ R10
	INCQ R11
	INCQ R12
	INCQ R13
	INCQ R14
	DECQ CX
	JNZ encode_w8_tail_loop

encode_w8_done:
	VZEROUPPER
	RET
