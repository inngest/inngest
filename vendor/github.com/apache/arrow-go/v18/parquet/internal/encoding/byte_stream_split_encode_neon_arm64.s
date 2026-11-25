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

// NEON implementation of BYTE_STREAM_SPLIT encoding.

#include "textflag.h"

// func _encodeByteStreamSplitWidth4NEON(in, out unsafe.Pointer, nValues int)
//
// VLD4 deinterleaves sixteen 4-byte values into the four output streams.
TEXT ·_encodeByteStreamSplitWidth4NEON(SB), NOSPLIT, $0-24
	MOVD in+0(FP), R0
	MOVD out+8(FP), R1
	MOVD nValues+16(FP), R2
	CBZ R2, encode_w4_neon_done

	MOVD R2, R3
	LSR $4, R3, R3
	LSL $4, R3, R12
	SUB R12, R2, R13

	MOVD R1, R4
	ADD R2, R1, R5
	ADD R2, R5, R6
	ADD R2, R6, R7

	CBZ R3, encode_w4_neon_tail

encode_w4_neon_vector:
	VLD4 (R0), [V0.B16, V1.B16, V2.B16, V3.B16]
	VST1 [V0.B16], (R4)
	VST1 [V1.B16], (R5)
	VST1 [V2.B16], (R6)
	VST1 [V3.B16], (R7)

	ADD $64, R0, R0
	ADD $16, R4, R4
	ADD $16, R5, R5
	ADD $16, R6, R6
	ADD $16, R7, R7
	SUB $1, R3, R3
	CBNZ R3, encode_w4_neon_vector

encode_w4_neon_tail:
	CBZ R13, encode_w4_neon_done

encode_w4_neon_tail_loop:
	MOVBU 0(R0), R14
	MOVB R14, (R4)
	MOVBU 1(R0), R14
	MOVB R14, (R5)
	MOVBU 2(R0), R14
	MOVB R14, (R6)
	MOVBU 3(R0), R14
	MOVB R14, (R7)

	ADD $4, R0, R0
	ADD $1, R4, R4
	ADD $1, R5, R5
	ADD $1, R6, R6
	ADD $1, R7, R7
	SUB $1, R13, R13
	CBNZ R13, encode_w4_neon_tail_loop

encode_w4_neon_done:
	RET

// func _encodeByteStreamSplitWidth8NEON(in, out unsafe.Pointer, nValues int)
//
// VLD4 first separates each 8-byte value into its low and high four-byte
// groups. VUZP then separates those groups into the eight byte streams.
TEXT ·_encodeByteStreamSplitWidth8NEON(SB), NOSPLIT, $0-24
	MOVD in+0(FP), R0
	MOVD out+8(FP), R1
	MOVD nValues+16(FP), R2
	CBZ R2, encode_w8_neon_done

	MOVD R2, R3
	LSR $3, R3, R3
	LSL $3, R3, R12
	SUB R12, R2, R13

	MOVD R1, R4
	ADD R2, R1, R5
	ADD R2, R5, R6
	ADD R2, R6, R7
	LSL $2, R2, R12
	ADD R12, R1, R8
	ADD R12, R5, R9
	ADD R12, R6, R10
	ADD R12, R7, R11

	CBZ R3, encode_w8_neon_tail

encode_w8_neon_vector:
	VLD4 (R0), [V0.B16, V1.B16, V2.B16, V3.B16]

	VUZP1 V0.B16, V0.B16, V4.B16
	VUZP2 V0.B16, V0.B16, V8.B16
	VUZP1 V1.B16, V1.B16, V5.B16
	VUZP2 V1.B16, V1.B16, V9.B16
	VUZP1 V2.B16, V2.B16, V6.B16
	VUZP2 V2.B16, V2.B16, V10.B16
	VUZP1 V3.B16, V3.B16, V7.B16
	VUZP2 V3.B16, V3.B16, V11.B16

	VST1 [V4.B8], (R4)
	VST1 [V5.B8], (R5)
	VST1 [V6.B8], (R6)
	VST1 [V7.B8], (R7)
	VST1 [V8.B8], (R8)
	VST1 [V9.B8], (R9)
	VST1 [V10.B8], (R10)
	VST1 [V11.B8], (R11)

	ADD $64, R0, R0
	ADD $8, R4, R4
	ADD $8, R5, R5
	ADD $8, R6, R6
	ADD $8, R7, R7
	ADD $8, R8, R8
	ADD $8, R9, R9
	ADD $8, R10, R10
	ADD $8, R11, R11
	SUB $1, R3, R3
	CBNZ R3, encode_w8_neon_vector

encode_w8_neon_tail:
	CBZ R13, encode_w8_neon_done

encode_w8_neon_tail_loop:
	MOVBU 0(R0), R14
	MOVB R14, (R4)
	MOVBU 1(R0), R14
	MOVB R14, (R5)
	MOVBU 2(R0), R14
	MOVB R14, (R6)
	MOVBU 3(R0), R14
	MOVB R14, (R7)
	MOVBU 4(R0), R14
	MOVB R14, (R8)
	MOVBU 5(R0), R14
	MOVB R14, (R9)
	MOVBU 6(R0), R14
	MOVB R14, (R10)
	MOVBU 7(R0), R14
	MOVB R14, (R11)

	ADD $8, R0, R0
	ADD $1, R4, R4
	ADD $1, R5, R5
	ADD $1, R6, R6
	ADD $1, R7, R7
	ADD $1, R8, R8
	ADD $1, R9, R9
	ADD $1, R10, R10
	ADD $1, R11, R11
	SUB $1, R13, R13
	CBNZ R13, encode_w8_neon_tail_loop

encode_w8_neon_done:
	RET
