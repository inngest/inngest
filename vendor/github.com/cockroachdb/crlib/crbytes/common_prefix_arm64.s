// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in licenses/BSD-golang.txt.

// This code is based on compare_arm64.s from Go 1.12.5.

TEXT ·CommonPrefix(SB),$0-56
    // R0 = uintptr(unsafe.Pointer(&a[0]))
    MOVD    a_base+0(FP), R0
    // R1 = len(a)
    MOVD    a_len+8(FP), R1
    // R2 = uintptr(unsafe.Pointer(&b[0]))
    MOVD    b_base+24(FP), R2
    // R3 = len(b)
    MOVD    b_len+32(FP), R3

    CMP     R1, R3
    // R6 = min(alen, blen)
    CSEL    LT, R3, R1, R6
    // Throughout this function, R7 remembers the original min(alen, blen) and
    // R6 is the number of bytes we still need to compare (with bytes 0 to R7-R6
    // known to match).
    MOVD    R6, R7

    // if R6 == 0 {
    //   goto samebytes
    // }
    CBZ     R6, samebytes
    // IF R6 < 16 {
    //   goto small
    // }
    CMP     $16, R6
    BLT     small

// chunk16_loop compares 16 bytes at a time.
// Invariant: R6 >= 16
chunk16_loop:
    // R4, R8, a = a[:8], a[8:16], a[16:]
    LDP.P   16(R0), (R4, R8)
    // R5, R9, b = b[:8], b[8:16]; b[16:]
    LDP.P   16(R2), (R5, R9)
    // if R4 != R5 {
    //   goto cmp
    // }
    CMP     R4, R5
    BNE     cmp
    // if R8 != R9 {
    //   goto cmpnext
    // }
    CMP     R8, R9
    BNE     cmpnext
    // R6 -= 16
    SUB     $16, R6
    // if R6 >= 16 {
    //   goto chunk16_loop
    // }
    CMP     $16, R6
    BGE     chunk16_loop
    // if R6 == 0 {
    //   goto samebytes
    // }
    CBZ     R6, samebytes
    // if R6 <= 8 {
    //   goto tail
    // }
    CMP     $8, R6
    BLE     tail
    // We have more than 8 bytes remaining; compare the first 8 bytes.
    // R4, a = a[:8], a[8:]
    // R5, b = b[:8], b[8:]
    MOVD.P  8(R0), R4
    MOVD.P  8(R2), R5
    // if R4 != R5 {
    //   goto cmp
    // }
    CMP     R4, R5
    BNE     cmp
    // R6 -= 8
    SUB     $8, R6

// Invariants:
//  - the original slices have at least 8 bytes (R7 >= 8)
//  - there are at most 8 bytes left to compare (R6 <= 8)
tail:
    // R6 -= 8
    SUB     $8, R6
    // R4 = a[R6:R6+8]
    MOVD    (R0)(R6), R4
    // R5 = b[R6:R6+8]
    MOVD    (R2)(R6), R5
    // if R4 == R6 {
    //   goto samebytes
    // }
    CMP     R4, R5
    BEQ     samebytes
    // R6 = 8
    MOVD    $8, R6

// Invariants: R4 and R5 contain the next 8 bytes and R4 != R5.
cmp:
    // R4 = bits.ReverseBytes64(R4)
    REV     R4, R4
    // R5 = bits.ReverseBytes64(R5)
    REV     R5, R5
// Invariant: R4 and R5 contain the next 8 bytes in reverse order and R4 != R5.
cmprev:
    // R5 ^= R4
    EOR     R4, R5, R5
    // R5 = bits.LeadingZeros64(R5)
    // This is the number of bits that match.
    CLZ     R5, R5
    // R5 /= 8
    // This is the number of bytes that match.
    LSR     $3, R5, R5
    // R6 -= R5
    SUBS    R5, R6, R6
    // if R6 == 0 {
    //   goto samebytes
    // }
    BLT samebytes

ret:
    // return R7 - R6
    SUB R6, R7
    MOVD    R7, ret+48(FP)
    RET

// Invariant: we have less than 16 bytes to compare (R6 = R7, R6 < 16).
small:
    // Test Bit and Branch if Zero:
    //   if R6 & 8 != 0 {
    //     goto lt_8
    //   }
    TBZ     $3, R6, lt_8
    // R4 = a[:8]
    MOVD    (R0), R4
    // R5 = b[:8]
    MOVD    (R2), R5
    // if R4 != R5 {
    //   goto cmp
    // }
    CMP     R4, R5
    BNE     cmp
    // R6 -= 8
    SUBS    $8, R6, R6
    // if R6 == 0 {
    //   goto samebytes
    // }
    BEQ     samebytes
    // a = a[8:]
    ADD     $8, R0
    // b = b[8:]
    ADD     $8, R2
    // goto tail
    B       tail

// Invariant: we have less than 8 bytes to compare (R6 = R7, R6 < 8).
lt_8:
    // Test Bit and Branch if Zero:
    //   if R6 & 4 != 0 {
    //     goto lt_4
    //   }
    TBZ     $2, R6, lt_4
    // R4 = a[:4]
    MOVWU   (R0), R4
    // R5 = b[:4]
    MOVWU   (R2), R5
    // if R4 != R5 {
    //   goto cmp
    // }
    CMPW    R4, R5
    BNE     cmp
    // R6 -= 4
    SUBS    $4, R6
    // if R6 == 0 {
    //   goto samebytes
    // }
    BEQ     samebytes
    // a = a[4:]
    ADD     $4, R0
    // b = b[4:]
    ADD     $4, R2

// Invariant: we have less than 4 bytes to compare (R6 = R7, R6 < 4).
lt_4:
    // Test Bit and Branch if Zero:
    //   if R6 & 2 != 0 {
    //     goto lt_2
    //   }
    TBZ     $1, R6, lt_2
    // R4 = a[:2]
    MOVHU   (R0), R4
    // R5 = b[:2]
    MOVHU   (R2), R5
    CMPW    R4, R5
    // if R4 != R5 {
    //   goto cmp
    // }
    BNE     cmp
    // a = a[2:]
    ADD     $2, R0
    // b = b[2:]
    ADD     $2, R2
    // R6 -= 2
    SUB     $2, R6

// Invariant: we have less than 2 bytes to compare (R6 = R7, R6 < 2).
lt_2:
    // if R6 == 0 {
    //   goto samebytes
    // }
    TBZ    $0, R6, samebytes

// Invariant: we have 1 byte to compare (R6 = R7 = 1).
one:
    // R4 = a[:1]
    MOVBU   (R0), R4
    // R6 = b[:1]
    MOVBU   (R2), R5
    // if R4 != R5 {
    //   goto ret
    // }
    CMPW    R4, R5
    BNE     ret

// Invariant: all R7 bytes matched.
samebytes:
    // Return R7
    MOVD    R7, ret+48(FP)
    RET

// Invariants:
//   - the next 8 bytes match (a[:8] == b[:8])
//   - the following bytes R8 and R9 contain the following 8 bytes (R8 = a[8:16], R9 = b[8:16])
//   - R8 != R9
cmpnext:
    // R6 -= 8
    SUB     $8, R6
    // R4 = bits.ReverseBytes64(R8)
    REV     R8, R4
    // R5 = bits.ReverseBytes64(R9)
    REV     R9, R5
    // goto cmprev
    B       cmprev
