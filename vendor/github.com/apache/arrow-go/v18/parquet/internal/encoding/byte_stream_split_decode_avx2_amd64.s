//+build !noasm !appengine
// AVX2 implementation with 256-bit vectors

#include "textflag.h"

// func _decodeByteStreamSplitWidth4AVX2(data, out unsafe.Pointer, nValues, stride int)
//
// AVX2 implementation with 256-bit vectors:
// - Processes suffix FIRST, then vectorized blocks
// - Uses 256-bit AVX2 vectors (32 bytes per register)
// - Processes 32 float32 values per block (128 bytes total)
// - Uses 2-stage byte unpacking hierarchy
TEXT ·_decodeByteStreamSplitWidth4AVX2(SB), NOSPLIT, $0-32
	MOVQ data+0(FP), SI      // SI = data pointer
	MOVQ out+8(FP), DI       // DI = out pointer
	MOVQ nValues+16(FP), CX  // CX = nValues
	MOVQ stride+24(FP), DX   // DX = stride

	// Setup stream pointers
	MOVQ SI, R9              // stream 0
	LEAQ (SI)(DX*1), R10     // stream 1 = data + stride
	LEAQ (SI)(DX*2), R11     // stream 2 = data + 2*stride
	LEAQ (R10)(DX*2), R12    // stream 3 = data + 3*stride

	// Calculate num_blocks = nValues / 32
	MOVQ CX, AX
	SHRQ $5, AX              // AX = num_blocks (divide by 32)

	// Calculate num_processed_elements = num_blocks * 32
	MOVQ AX, R13
	SHLQ $5, R13             // R13 = num_processed_elements

	// First handle suffix (elements beyond complete blocks)
	MOVQ R13, R14            // R14 = i = num_processed_elements
	JMP suffix_check_avx2

suffix_loop_avx2:
	MOVBQZX (R9)(R14*1), BX      // s0
	MOVBQZX (R10)(R14*1), R15    // s1
	SHLQ $8, R15
	ORQ R15, BX
	MOVBQZX (R11)(R14*1), R15    // s2
	SHLQ $16, R15
	ORQ R15, BX
	MOVBQZX (R12)(R14*1), R15    // s3
	SHLQ $24, R15
	ORQ R15, BX
	MOVQ R14, AX

	SHLQ $2, AX              // AX = i*4
	MOVL BX, (DI)(AX*1)      // ← single 32-bit store

	INCQ R14

suffix_check_avx2:
	CMPQ R14, CX
	JL suffix_loop_avx2

	// Check if we have blocks to process
	TESTQ R13, R13           // Check if num_processed_elements > 0
	JZ done_avx2

	// Process blocks with AVX2
	XORQ R14, R14            // R14 = block index i = 0
	SHRQ $5, R13             // R13 = num_blocks

block_loop_avx2:
	// Calculate offset for this block: i * 32
	MOVQ R14, AX
	SHLQ $5, AX              // AX = i * 32

	// Load 32 bytes from each stream
	// stage[0][j] = _mm256_loadu_si256(&data[i * 32 + j * stride])
	VMOVDQU (R9)(AX*1), Y0    // stage[0][0] from stream 0
	VMOVDQU (R10)(AX*1), Y1   // stage[0][1] from stream 1
	VMOVDQU (R11)(AX*1), Y2   // stage[0][2] from stream 2
	VMOVDQU (R12)(AX*1), Y3   // stage[0][3] from stream 3

	// Stage 1: First level of byte interleaving
	// stage[1][0] = _mm256_unpacklo_epi8(stage[0][0], stage[0][2])
	// stage[1][1] = _mm256_unpackhi_epi8(stage[0][0], stage[0][2])
	// stage[1][2] = _mm256_unpacklo_epi8(stage[0][1], stage[0][3])
	// stage[1][3] = _mm256_unpackhi_epi8(stage[0][1], stage[0][3])

	VPUNPCKLBW Y2, Y0, Y4    // Y4 = unpacklo_epi8(Y0, Y2)
	VPUNPCKHBW Y2, Y0, Y5    // Y5 = unpackhi_epi8(Y0, Y2)
	VPUNPCKLBW Y3, Y1, Y6    // Y6 = unpacklo_epi8(Y1, Y3)
	VPUNPCKHBW Y3, Y1, Y7    // Y7 = unpackhi_epi8(Y1, Y3)

	// Stage 2: Second level of byte interleaving
	// stage[2][0] = _mm256_unpacklo_epi8(stage[1][0], stage[1][2])
	// stage[2][1] = _mm256_unpackhi_epi8(stage[1][0], stage[1][2])
	// stage[2][2] = _mm256_unpacklo_epi8(stage[1][1], stage[1][3])
	// stage[2][3] = _mm256_unpackhi_epi8(stage[1][1], stage[1][3])

	VPUNPCKLBW Y6, Y4, Y0    // Y0 = unpacklo_epi8(Y4, Y6)
	VPUNPCKHBW Y6, Y4, Y1    // Y1 = unpackhi_epi8(Y4, Y6)
	VPUNPCKLBW Y7, Y5, Y2    // Y2 = unpacklo_epi8(Y5, Y7)
	VPUNPCKHBW Y7, Y5, Y3    // Y3 = unpackhi_epi8(Y5, Y7)

	// Fix lane order: AVX2 unpacking operates within each 128-bit lane
	// After two levels of unpacking, we have:
	// Y0 = [bytes 0-7 of values 0-7 | bytes 0-7 of values 16-23]
	// Y1 = [bytes 0-7 of values 8-15 | bytes 0-7 of values 24-31]
	// Y2 = [same pattern for different byte positions]
	// Y3 = [same pattern for different byte positions]
	// We need: [values 0-7 | values 8-15 | values 16-23 | values 24-31]

	VPERM2I128 $0x20, Y1, Y0, Y4  // Y4 = [Y0_low(0-7) | Y1_low(8-15)]
	VPERM2I128 $0x31, Y1, Y0, Y5  // Y5 = [Y0_high(16-23) | Y1_high(24-31)]
	VPERM2I128 $0x20, Y3, Y2, Y6  // Y6 = [Y2_low | Y3_low]
	VPERM2I128 $0x31, Y3, Y2, Y7  // Y7 = [Y2_high | Y3_high]

	// Store results: out[(i * 4 + j) * 32] = stage[result][j]
	// Calculate output base: i * 128
	MOVQ R14, AX
	SHLQ $7, AX              // AX = i * 128

	VMOVDQU Y4, (DI)(AX*1)       // Store at offset 0
	VMOVDQU Y6, 32(DI)(AX*1)     // Store at offset 32
	VMOVDQU Y5, 64(DI)(AX*1)     // Store at offset 64
	VMOVDQU Y7, 96(DI)(AX*1)     // Store at offset 96

	INCQ R14
	CMPQ R14, R13
	JL block_loop_avx2

done_avx2:
	VZEROUPPER
	RET

// func _decodeByteStreamSplitWidth8AVX2(data, out unsafe.Pointer, nValues, stride int)
//
// AVX2 implementation for width=8 (float64/int64) with 256-bit vectors:
// - Processes suffix FIRST, then vectorized blocks
// - Uses 256-bit AVX2 vectors (32 bytes per register)
// - Processes 16 float64 values per block (128 bytes total)
// - Uses 3-stage byte unpacking hierarchy (for 8 streams)
TEXT ·_decodeByteStreamSplitWidth8AVX2(SB), NOSPLIT, $0-32
	MOVQ data+0(FP), SI      // SI = data pointer
	MOVQ out+8(FP), DI       // DI = out pointer
	MOVQ nValues+16(FP), CX  // CX = nValues
	MOVQ stride+24(FP), DX   // DX = stride

	// Setup 8 stream pointers
	MOVQ SI, R9              // stream 0
	LEAQ (SI)(DX*1), R10     // stream 1 = data + stride
	LEAQ (SI)(DX*2), R11     // stream 2 = data + 2*stride
	LEAQ (R10)(DX*2), R12    // stream 3 = data + 3*stride
	LEAQ (SI)(DX*4), R13     // stream 4 = data + 4*stride
	LEAQ (R10)(DX*4), R14    // stream 5 = data + 5*stride
	LEAQ (R11)(DX*4), R15    // stream 6 = data + 6*stride
	LEAQ (R12)(DX*4), BX     // stream 7 = data + 7*stride

	// Calculate num_blocks = nValues / 16
	MOVQ CX, AX
	SHRQ $4, AX              // AX = num_blocks

	// Calculate num_processed_elements = num_blocks * 16
	MOVQ AX, R8
	SHLQ $4, R8              // R8 = num_processed_elements

	// First handle suffix (elements beyond complete blocks)
	MOVQ R8, SI              // SI = i = num_processed_elements
	JMP suffix_check_w8_avx2

suffix_loop_w8_avx2:
	// Load first byte (stream 0) and start accumulator
	MOVBQZX (R9)(SI*1), DX   // DX = s0[i] (lowest byte)

	// stream 1 << 8
	MOVBQZX (R10)(SI*1), AX
	SHLQ $8, AX
	ORQ AX, DX

	// stream 2 << 16
	MOVBQZX (R11)(SI*1), AX
	SHLQ $16, AX
	ORQ AX, DX

	// stream 3 << 24
	MOVBQZX (R12)(SI*1), AX
	SHLQ $24, AX
	ORQ AX, DX

	// stream 4 << 32
	MOVBQZX (R13)(SI*1), AX
	SHLQ $32, AX
	ORQ AX, DX

	// stream 5 << 40
	MOVBQZX (R14)(SI*1), AX
	SHLQ $40, AX
	ORQ AX, DX

	// stream 6 << 48
	MOVBQZX (R15)(SI*1), AX
	SHLQ $48, AX
	ORQ AX, DX

	// stream 7 << 56
	MOVBQZX (BX)(SI*1), AX
	SHLQ $56, AX
	ORQ AX, DX

	MOVQ SI, AX
	SHLQ $3, AX              // AX = i * 8
	MOVQ DX, (DI)(AX*1)

	INCQ SI

suffix_check_w8_avx2:
	CMPQ SI, CX
	JL suffix_loop_w8_avx2

	// Check if we have blocks to process
	TESTQ R8, R8             // Check if num_processed_elements > 0
	JZ done_w8_avx2

	// Process blocks with AVX2
	XORQ SI, SI              // SI = block index i = 0
	SHRQ $4, R8              // R8 = num_blocks

block_loop_w8_avx2:
	// Calculate offset for this block: i * 16
	MOVQ SI, AX
	SHLQ $4, AX              // AX = i * 16

	// Load 16 bytes from each of 8 streams (using 128-bit loads for narrower data)
	// We load 16 bytes but will use AVX2 operations
	VMOVDQU (R9)(AX*1), X0    // stream 0
	VMOVDQU (R10)(AX*1), X1   // stream 1
	VMOVDQU (R11)(AX*1), X2   // stream 2
	VMOVDQU (R12)(AX*1), X3   // stream 3
	VMOVDQU (R13)(AX*1), X4   // stream 4
	VMOVDQU (R14)(AX*1), X5   // stream 5
	VMOVDQU (R15)(AX*1), X6   // stream 6
	VMOVDQU (BX)(AX*1), X7    // stream 7

	// Stage 1: First level of byte interleaving (pairs 0-4, 1-5, 2-6, 3-7)
	VPUNPCKLBW X4, X0, X8     // X8 = unpacklo_epi8(X0, X4)
	VPUNPCKHBW X4, X0, X9     // X9 = unpackhi_epi8(X0, X4)
	VPUNPCKLBW X5, X1, X10    // X10 = unpacklo_epi8(X1, X5)
	VPUNPCKHBW X5, X1, X11    // X11 = unpackhi_epi8(X1, X5)
	VPUNPCKLBW X6, X2, X12    // X12 = unpacklo_epi8(X2, X6)
	VPUNPCKHBW X6, X2, X13    // X13 = unpackhi_epi8(X2, X6)
	VPUNPCKLBW X7, X3, X14    // X14 = unpacklo_epi8(X3, X7)
	VPUNPCKHBW X7, X3, X15    // X15 = unpackhi_epi8(X3, X7)

	// Stage 2: Second level of byte interleaving
	VPUNPCKLBW X12, X8, X0    // X0 = unpacklo_epi8(X8, X12)
	VPUNPCKHBW X12, X8, X1    // X1 = unpackhi_epi8(X8, X12)
	VPUNPCKLBW X13, X9, X2    // X2 = unpacklo_epi8(X9, X13)
	VPUNPCKHBW X13, X9, X3    // X3 = unpackhi_epi8(X9, X13)
	VPUNPCKLBW X14, X10, X4   // X4 = unpacklo_epi8(X10, X14)
	VPUNPCKHBW X14, X10, X5   // X5 = unpackhi_epi8(X10, X14)
	VPUNPCKLBW X15, X11, X6   // X6 = unpacklo_epi8(X11, X15)
	VPUNPCKHBW X15, X11, X7   // X7 = unpackhi_epi8(X11, X15)

	// Stage 3: Third level of byte interleaving
	VPUNPCKLBW X4, X0, X8     // X8 = unpacklo_epi8(X0, X4)
	VPUNPCKHBW X4, X0, X9     // X9 = unpackhi_epi8(X0, X4)
	VPUNPCKLBW X5, X1, X10    // X10 = unpacklo_epi8(X1, X5)
	VPUNPCKHBW X5, X1, X11    // X11 = unpackhi_epi8(X1, X5)
	VPUNPCKLBW X6, X2, X12    // X12 = unpacklo_epi8(X2, X6)
	VPUNPCKHBW X6, X2, X13    // X13 = unpackhi_epi8(X2, X6)
	VPUNPCKLBW X7, X3, X14    // X14 = unpacklo_epi8(X3, X7)
	VPUNPCKHBW X7, X3, X15    // X15 = unpackhi_epi8(X3, X7)

	// Store results: out[(i * 8 + j) * 16] = result[j]
	// Calculate output base: i * 128
	MOVQ SI, AX
	SHLQ $7, AX              // AX = i * 128

	VMOVDQU X8, (DI)(AX*1)         // Store at offset 0
	VMOVDQU X9, 16(DI)(AX*1)       // Store at offset 16
	VMOVDQU X10, 32(DI)(AX*1)      // Store at offset 32
	VMOVDQU X11, 48(DI)(AX*1)      // Store at offset 48
	VMOVDQU X12, 64(DI)(AX*1)      // Store at offset 64
	VMOVDQU X13, 80(DI)(AX*1)      // Store at offset 80
	VMOVDQU X14, 96(DI)(AX*1)      // Store at offset 96
	VMOVDQU X15, 112(DI)(AX*1)     // Store at offset 112

	INCQ SI
	CMPQ SI, R8
	JL block_loop_w8_avx2

done_w8_avx2:
	VZEROUPPER
	RET
