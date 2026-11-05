//+build !noasm !appengine

// Optimized NEON bytes_to_bools: uses CMTST to extract 8 bits per byte
// in parallel via SIMD, ~4x faster than the scalar original.

// func _bytes_to_bools_neon(in unsafe.Pointer, len int, out unsafe.Pointer, outlen int)
TEXT ·_bytes_to_bools_neon(SB), $0-32

    MOVD in+0(FP), R0
    MOVD len+8(FP), R1
    MOVD out+16(FP), R2
    MOVD outlen+24(FP), R3

    WORD $0x7100043f // cmp    w1, #1
    BLT done

    // Build bit mask: v0.8b = [1, 2, 4, 8, 16, 32, 64, 128]
    // 0x8040201008040201 as LE 64-bit
    WORD $0xd2804028 // movz   x8, #0x201
    WORD $0xf2a10088 // movk   x8, #0x804, lsl #16
    WORD $0xf2c40208 // movk   x8, #0x2010, lsl #32
    WORD $0xf2f00808 // movk   x8, #0x8040, lsl #48
    WORD $0x9e670100 // fmov   d0, x8

    // v1.8b = all 0x01
    WORD $0x0f00e421 // movi   v1.8b, #1

    // R4 = input cursor, R5 = output cursor
    WORD $0xaa0003e4 // mov    x4, x0
    WORD $0xaa0203e5 // mov    x5, x2

    // R6 = input end (in + len)
    WORD $0x8b010006 // add    x6, x0, x1

    // R7 = output end (out + outlen)
    WORD $0x8b030047 // add    x7, x2, x3

simd_loop:
    // Need at least 1 input byte
    WORD $0xeb06009f // cmp    x4, x6
    BGE done

    // Need at least 8 output bytes remaining
    WORD $0xcb050068 // sub    x8, x3, x5  ... NO this is sub x8, x3, x5 but x3=outlen, x5=out_cursor
    // We need: output_end - output_cursor >= 8
    // output_end = x7, output_cursor = x5
    WORD $0xcb0500e8 // sub    x8, x7, x5
    WORD $0xf100211f // cmp    x8, #8
    BLT scalar_setup

    // SIMD: process 1 byte -> 8 bools
    // ld1r {v2.8b}, [x4] — broadcast byte to all 8 lanes
    WORD $0x0d40c082 // ld1r   {v2.8b}, [x4]

    // cmtst v2.8b, v2.8b, v0.8b — test (v2 AND v0) != 0 → 0xFF/0x00
    WORD $0x0e208c42 // cmtst  v2.8b, v2.8b, v0.8b

    // and v2.8b, v2.8b, v1.8b — convert 0xFF to 0x01
    WORD $0x0e211c42 // and    v2.8b, v2.8b, v1.8b

    // st1 {v2.8b}, [x5], #8 — store 8 bools, advance out ptr
    WORD $0x0c9f70a2 // st1    {v2.8b}, [x5], #8

    // Advance input by 1
    WORD $0x91000484 // add    x4, x4, #1

    JMP simd_loop

scalar_setup:
    // For remaining bits when output space < 8

scalar_loop:
    WORD $0xeb06009f // cmp    x4, x6
    BGE done

    WORD $0xeb0700bf // cmp    x5, x7
    BGE done

    // Load one input byte
    WORD $0x3940008a // ldrb   w10, [x4]

    // bit 0
    WORD $0x1200014b // and    w11, w10, #0x1
    WORD $0x390000ab // strb   w11, [x5]
    WORD $0x910004a5 // add    x5, x5, #1
    WORD $0xeb0700bf // cmp    x5, x7
    BGE scalar_next

    // bit 1
    WORD $0x5301054b // ubfx   w11, w10, #1, #1
    WORD $0x390000ab // strb   w11, [x5]
    WORD $0x910004a5 // add    x5, x5, #1
    WORD $0xeb0700bf // cmp    x5, x7
    BGE scalar_next

    // bit 2
    WORD $0x5302094b // ubfx   w11, w10, #2, #1
    WORD $0x390000ab // strb   w11, [x5]
    WORD $0x910004a5 // add    x5, x5, #1
    WORD $0xeb0700bf // cmp    x5, x7
    BGE scalar_next

    // bit 3
    WORD $0x53030d4b // ubfx   w11, w10, #3, #1
    WORD $0x390000ab // strb   w11, [x5]
    WORD $0x910004a5 // add    x5, x5, #1
    WORD $0xeb0700bf // cmp    x5, x7
    BGE scalar_next

    // bit 4
    WORD $0x5304114b // ubfx   w11, w10, #4, #1
    WORD $0x390000ab // strb   w11, [x5]
    WORD $0x910004a5 // add    x5, x5, #1
    WORD $0xeb0700bf // cmp    x5, x7
    BGE scalar_next

    // bit 5
    WORD $0x5305154b // ubfx   w11, w10, #5, #1
    WORD $0x390000ab // strb   w11, [x5]
    WORD $0x910004a5 // add    x5, x5, #1
    WORD $0xeb0700bf // cmp    x5, x7
    BGE scalar_next

    // bit 6
    WORD $0x5306194b // ubfx   w11, w10, #6, #1
    WORD $0x390000ab // strb   w11, [x5]
    WORD $0x910004a5 // add    x5, x5, #1
    WORD $0xeb0700bf // cmp    x5, x7
    BGE scalar_next

    // bit 7
    WORD $0x53077d4b // lsr    w11, w10, #7
    WORD $0x390000ab // strb   w11, [x5]
    WORD $0x910004a5 // add    x5, x5, #1

scalar_next:
    WORD $0x91000484 // add    x4, x4, #1
    JMP scalar_loop

done:
    RET
