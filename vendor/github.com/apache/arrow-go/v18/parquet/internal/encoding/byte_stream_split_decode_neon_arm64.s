//+build !noasm !appengine
// NEON implementation following AVX2 algorithm structure with 128-bit vectors

#include "textflag.h"

// func _decodeByteStreamSplitWidth4NEON(data, out unsafe.Pointer, nValues, stride int)
//
// NEON implementation following the AVX2 algorithm structure:
// - Processes suffix FIRST, then vectorized blocks
// - Uses 128-bit NEON vectors (16 bytes per register)
// - Processes 16 float32 values per block (64 bytes total)
// - Uses 2-stage byte unpacking hierarchy
TEXT ·_decodeByteStreamSplitWidth4NEON(SB), NOSPLIT, $0-32
	MOVD data+0(FP), R0      // R0 = data pointer
	MOVD out+8(FP), R1       // R1 = out pointer
	MOVD nValues+16(FP), R2  // R2 = nValues
	MOVD stride+24(FP), R3   // R3 = stride

	// Setup stream pointers
	MOVD R0, R4              // R4 = stream 0
	ADD R3, R0, R5           // R5 = stream 1 = data + stride
	ADD R3, R5, R6           // R6 = stream 2 = data + 2*stride
	ADD R3, R6, R7           // R7 = stream 3 = data + 3*stride

	// Calculate num_blocks = nValues / 16
	LSR $4, R2, R8           // R8 = num_blocks (divide by 16)

	// Calculate num_processed_elements = num_blocks * 16
	LSL $4, R8, R9           // R9 = num_processed_elements

	// First handle suffix (elements beyond complete blocks)
	MOVD R9, R10             // R10 = i = num_processed_elements
	B suffix_check_neon

suffix_loop_neon:
	// Gather bytes: gathered_byte_data[b] = data[b * stride + i]
	MOVBU (R4)(R10), R11      // byte from stream 0
	MOVBU (R5)(R10), R12      // byte from stream 1

	// Calculate output offset: i * 4
	LSL $2, R10, R13         // R13 = i * 4
	ADD R1, R13, R14         // R14 = out + (i * 4)

	// Store gathered bytes
	MOVB R11, (R14)
	MOVB R12, 1(R14)

	MOVBU (R6)(R10), R11      // byte from stream 2
	MOVBU (R7)(R10), R12      // byte from stream 3

	MOVB R11, 2(R14)
	MOVB R12, 3(R14)

	ADD $1, R10, R10

suffix_check_neon:
	CMP R2, R10
	BLT suffix_loop_neon

	// Check if we have blocks to process
	CBZ R9, done_neon        // if num_processed_elements == 0, done

	// Process blocks with NEON
	MOVD $0, R10             // R10 = block index i = 0
	LSR $4, R9, R9           // R9 = num_blocks

block_loop_neon:
	// Calculate offset for this block: i * 16
	LSL $4, R10, R11         // R11 = i * 16

	// Load 16 bytes from each stream - REVERSED for little-endian!
	ADD R7, R11, R12
	VLD1 (R12), [V0.B16]     // V0 = stream 3 (MSB in little-endian)

	ADD R6, R11, R12
	VLD1 (R12), [V1.B16]     // V1 = stream 2

	ADD R5, R11, R12
	VLD1 (R12), [V2.B16]     // V2 = stream 1

	ADD R4, R11, R12
	VLD1 (R12), [V3.B16]     // V3 = stream 0 (LSB in little-endian)

	// Stage 1: Interleave like AVX2 VPUNPCKLBW/VPUNPCKHBW
	VZIP1 V0.B16, V2.B16, V4.B16    // Interleave streams 0,2 low bytes
	VZIP2 V0.B16, V2.B16, V5.B16    // Interleave streams 0,2 high bytes
	VZIP1 V1.B16, V3.B16, V6.B16    // Interleave streams 1,3 low bytes
	VZIP2 V1.B16, V3.B16, V7.B16    // Interleave streams 1,3 high bytes

	// Stage 2: Second level
	VZIP1 V4.B16, V6.B16, V0.B16
	VZIP2 V4.B16, V6.B16, V1.B16
	VZIP1 V5.B16, V7.B16, V2.B16
	VZIP2 V5.B16, V7.B16, V3.B16

	// Store results in sequential order
	// Calculate output base: i * 64
	LSL $6, R10, R11         // R11 = i * 64

	ADD R1, R11, R12
	VST1 [V0.B16], (R12)          // Store at offset 0

	ADD $16, R12
	VST1 [V1.B16], (R12)          // Store at offset 16

	ADD $16, R12
	VST1 [V2.B16], (R12)          // Store at offset 32

	ADD $16, R12
	VST1 [V3.B16], (R12)          // Store at offset 48

	ADD $1, R10, R10
	CMP R9, R10
	BLT block_loop_neon

done_neon:
	RET

// func _decodeByteStreamSplitWidth8NEON(data, out unsafe.Pointer, nValues, stride int)
//
// NEON implementation for width=8 following AVX2 algorithm structure:
// - Processes suffix FIRST, then vectorized blocks
// - Uses 128-bit NEON vectors (16 bytes per register)
// - Processes 16 float64 values per block (128 bytes total)
// - Uses 3-stage byte unpacking hierarchy (for 8 streams)
TEXT ·_decodeByteStreamSplitWidth8NEON(SB), NOSPLIT, $0-32
	MOVD data+0(FP), R0      // R0 = data pointer
	MOVD out+8(FP), R1       // R1 = out pointer
	MOVD nValues+16(FP), R2  // R2 = nValues
	MOVD stride+24(FP), R3   // R3 = stride

	// Setup 8 stream pointers
	MOVD R0, R4              // R4 = stream 0
	ADD R3, R0, R5           // R5 = stream 1 = data + stride
	ADD R3, R5, R6           // R6 = stream 2 = data + 2*stride
	ADD R3, R6, R7           // R7 = stream 3 = data + 3*stride
	LSL $2, R3, R8           // R8 = 4*stride
	ADD R0, R8, R9           // R9 = stream 4 = data + 4*stride
	ADD R5, R8, R10          // R10 = stream 5 = data + 5*stride
	ADD R6, R8, R11          // R11 = stream 6 = data + 6*stride
	ADD R7, R8, R12          // R12 = stream 7 = data + 7*stride

	// Calculate num_blocks = nValues / 16
	LSR $4, R2, R13          // R13 = num_blocks (divide by 16)

	// Calculate num_processed_elements = num_blocks * 16
	LSL $4, R13, R14         // R14 = num_processed_elements

	// First handle suffix (elements beyond complete blocks)
	MOVD R14, R15            // R15 = i = num_processed_elements
	B suffix_check_w8_neon

suffix_loop_w8_neon:
	// Calculate output offset: i * 8
	LSL $3, R15, R16         // R16 = i * 8
	ADD R1, R16, R17         // R17 = out + (i * 8)

	// Load and store bytes from all 8 streams
	MOVBU (R4)(R15), R19
	MOVB R19, (R17)

	MOVBU (R5)(R15), R19
	MOVB R19, 1(R17)

	MOVBU (R6)(R15), R19
	MOVB R19, 2(R17)

	MOVBU (R7)(R15), R19
	MOVB R19, 3(R17)

	MOVBU (R9)(R15), R19
	MOVB R19, 4(R17)

	MOVBU (R10)(R15), R19
	MOVB R19, 5(R17)

	MOVBU (R11)(R15), R19
	MOVB R19, 6(R17)

	MOVBU (R12)(R15), R19
	MOVB R19, 7(R17)

	ADD $1, R15, R15

suffix_check_w8_neon:
	CMP R2, R15
	BLT suffix_loop_w8_neon

	// Check if we have blocks to process
	CBZ R14, done_w8_neon    // if num_processed_elements == 0, done

	// Process blocks with NEON
	MOVD $0, R15             // R15 = block index i = 0
	LSR $4, R14, R14         // R14 = num_blocks

block_loop_w8_neon:
	// Calculate offset for this block: i * 16
	LSL $4, R15, R16         // R16 = i * 16

	// Load 16 bytes from each stream - REVERSED for little-endian!
	// V0 = stream 7 (MSB), V7 = stream 0 (LSB)
	ADD R12, R16, R17
	VLD1 (R17), [V0.B16]     // V0 = stream 7 (MSB in little-endian)

	ADD R11, R16, R17
	VLD1 (R17), [V1.B16]     // V1 = stream 6

	ADD R10, R16, R17
	VLD1 (R17), [V2.B16]     // V2 = stream 5

	ADD R9, R16, R17
	VLD1 (R17), [V3.B16]     // V3 = stream 4

	ADD R7, R16, R17
	VLD1 (R17), [V4.B16]     // V4 = stream 3

	ADD R6, R16, R17
	VLD1 (R17), [V5.B16]     // V5 = stream 2

	ADD R5, R16, R17
	VLD1 (R17), [V6.B16]     // V6 = stream 1

	ADD R4, R16, R17
	VLD1 (R17), [V7.B16]     // V7 = stream 0 (LSB in little-endian)

	// Stage 1: First level of byte interleaving (pairs 0-4, 1-5, 2-6, 3-7)
	VZIP1 V0.B16, V4.B16, V8.B16     // Interleave streams 3,7 low
	VZIP2 V0.B16, V4.B16, V9.B16     // Interleave streams 3,7 high
	VZIP1 V1.B16, V5.B16, V10.B16    // Interleave streams 2,6 low
	VZIP2 V1.B16, V5.B16, V11.B16    // Interleave streams 2,6 high
	VZIP1 V2.B16, V6.B16, V12.B16    // Interleave streams 1,5 low
	VZIP2 V2.B16, V6.B16, V13.B16    // Interleave streams 1,5 high
	VZIP1 V3.B16, V7.B16, V14.B16    // Interleave streams 0,4 low
	VZIP2 V3.B16, V7.B16, V15.B16    // Interleave streams 0,4 high

	// Stage 2: Second level of byte interleaving
	VZIP1 V8.B16, V12.B16, V0.B16
	VZIP2 V8.B16, V12.B16, V1.B16
	VZIP1 V9.B16, V13.B16, V2.B16
	VZIP2 V9.B16, V13.B16, V3.B16
	VZIP1 V10.B16, V14.B16, V4.B16
	VZIP2 V10.B16, V14.B16, V5.B16
	VZIP1 V11.B16, V15.B16, V6.B16
	VZIP2 V11.B16, V15.B16, V7.B16

	// Stage 3: Third level of byte interleaving
	VZIP1 V0.B16, V4.B16, V8.B16
	VZIP2 V0.B16, V4.B16, V9.B16
	VZIP1 V1.B16, V5.B16, V10.B16
	VZIP2 V1.B16, V5.B16, V11.B16
	VZIP1 V2.B16, V6.B16, V12.B16
	VZIP2 V2.B16, V6.B16, V13.B16
	VZIP1 V3.B16, V7.B16, V14.B16
	VZIP2 V3.B16, V7.B16, V15.B16

	// Store results: out[(i * 8 + j) * 16] = result[j]
	// Calculate output base: i * 128
	LSL $7, R15, R16         // R16 = i * 128

	ADD R1, R16, R17
	VST1 [V8.B16], (R17)          // Store at offset 0

	ADD $16, R17
	VST1 [V9.B16], (R17)          // Store at offset 16

	ADD $16, R17
	VST1 [V10.B16], (R17)         // Store at offset 32

	ADD $16, R17
	VST1 [V11.B16], (R17)         // Store at offset 48

	ADD $16, R17
	VST1 [V12.B16], (R17)         // Store at offset 64

	ADD $16, R17
	VST1 [V13.B16], (R17)         // Store at offset 80

	ADD $16, R17
	VST1 [V14.B16], (R17)         // Store at offset 96

	ADD $16, R17
	VST1 [V15.B16], (R17)         // Store at offset 112

	ADD $1, R15, R15
	CMP R14, R15
	BLT block_loop_w8_neon

done_w8_neon:
	RET
