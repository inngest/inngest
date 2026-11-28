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

//go:build amd64 && !noasm && !appengine
// +build amd64,!noasm,!appengine

#include "textflag.h"

TEXT ·_dictionary_gather_32_avx2(SB), NOSPLIT, $0-32
	MOVQ dictionary+0(FP), DI
	MOVQ output+8(FP), SI
	MOVQ indices+16(FP), DX
	MOVQ length+24(FP), CX

	XORQ R8, R8
	MOVQ CX, R9
	ANDQ $-8, R9

gather32_vector:
	CMPQ R8, R9
	JAE gather32_scalar
	VMOVDQU (DX)(R8*4), Y0
	VPCMPEQD Y1, Y1, Y1
	VPGATHERDD Y1, (DI)(Y0*4), Y2
	VMOVDQU Y2, (SI)(R8*4)
	ADDQ $8, R8
	JMP gather32_vector

gather32_scalar:
	CMPQ R8, CX
	JAE gather32_done
	MOVL (DX)(R8*4), R10
	MOVL (DI)(R10*4), R11
	MOVL R11, (SI)(R8*4)
	INCQ R8
	JMP gather32_scalar

gather32_done:
	VZEROUPPER
	RET

TEXT ·_dictionary_gather_64_avx2(SB), NOSPLIT, $0-32
	MOVQ dictionary+0(FP), DI
	MOVQ output+8(FP), SI
	MOVQ indices+16(FP), DX
	MOVQ length+24(FP), CX

	XORQ R8, R8
	MOVQ CX, R9
	ANDQ $-4, R9

gather64_vector:
	CMPQ R8, R9
	JAE gather64_scalar
	VMOVDQU (DX)(R8*4), X0
	VPCMPEQD Y1, Y1, Y1
	VPGATHERDQ Y1, (DI)(X0*8), Y2
	VMOVDQU Y2, (SI)(R8*8)
	ADDQ $4, R8
	JMP gather64_vector

gather64_scalar:
	CMPQ R8, CX
	JAE gather64_done
	MOVL (DX)(R8*4), R10
	MOVQ (DI)(R10*8), R11
	MOVQ R11, (SI)(R8*8)
	INCQ R8
	JMP gather64_scalar

gather64_done:
	VZEROUPPER
	RET
