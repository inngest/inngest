// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in licenses/BSD-golang.txt.

// This code is based on compare_amd64.s from Go 1.12.5.

TEXT ·CommonPrefix(SB),$0-56
    // SI = uintptr(unsafe.Pointer(&a[0]))
    MOVQ    a_base+0(FP), SI
    // BX = len(a)
    MOVQ    a_len+8(FP), BX
    // DI = uintptr(unsafe.Pointer(&b[0]))
    MOVQ    b_base+24(FP), DI
    // DX = len(b)
    MOVQ    b_len+32(FP), DX

    CMPQ    BX, DX
    MOVQ    DX, R8
    CMOVQLT BX, R8 // R8 = min(alen, blen) = # of bytes to compare
    // Throughout this function, DX remembers the original min(alen, blen) and
    // R8 is the number of bytes we still need to compare (with bytes 0 to
    // DX-R8 known to match).
    MOVQ    R8, DX
    CMPQ    R8, $8
    JB  small

    CMPQ    R8, $63
    JBE     loop
    JMP     big_loop
    RET

// loop is used when we have between 8 and 63 bytes left to compare (8 <= R8 < 64).
// Invariant: 8 <= R8 < 64
loop:
    CMPQ    R8, $16
    JB      _0through15
    // X0 = a[:16]
    MOVOU   (SI), X0
    // X0 = b[:16]
    MOVOU   (DI), X1
    // Compare Packed Data for Equal:
    //   for i := 0; i < 16; i++ {
    //     if X0[i] != X1[i] {
    //      X1[i] = 0
    //     } else {
    //      X1[i] = 0xFF
    //     }
    //   }
    PCMPEQB X0, X1
    // Move Byte Mask.
    //   AX = 0
    //   for i := 0; i < 16; i++ {
    //     if X1[i] & 0x80 != 0 {
    //       AX |= (1 << i)
    //   }
    PMOVMSKB X1, AX
    // AX ^= 0xFFFF
    XORQ    $0xffff, AX    // convert EQ to NE
    // if AX != 0 {
    //   goto diff16
    // }
    JNE     diff16    // branch if at least one byte is not equal
    // a = a[16:]
    ADDQ    $16, SI
    // b = b[16:]
    ADDQ    $16, DI
    // R8 -= 16
    SUBQ    $16, R8
    JMP     loop

// Invariant: a[0:48] matches b[0:48] and AX contains a bit mask of differences
// between a[48:64] and b[48:64].
diff64:
    // R8 -= 48
    SUBQ    $48, R8
    JMP     diff16

// Invariant: a[0:32] matches b[0:32] and AX contains a bit mask of differences
// between a[32:48] and b[32:48].
diff48:
    // R8 -= 32
    SUBQ    $32, R8
    JMP     diff16

// Invariant: a[0:16] matches b[0:16] and AX contains a bit mask of differences
// between a[16:32] and b[16:32].
diff32:
    // R8 -= 16
    SUBQ    $16, R8

// Invariant: AX contains a bit mask of differences between a[:16] and b[:16].
//   AX & (1 << i) == 1 iff a[i] != b[i]
diff16:
    // Bit Scan Forward (return the index of the least significant set bit)
    //   BX = bits.TrailingZeros64(AX)
    BSFQ    AX, BX
    // BX is now the prefix of bytes that matched, advance by this much.
    // R8 -= BX
    SUBQ    BX, R8

    // Return DX (original min(alen, blen)) - R8 (bytes left to compare)
    SUBQ    R8, DX
    MOVQ    DX, ret+48(FP)
    RET

// Invariants:
//  - original slices contained at least 8 bytes (DX >= 8)
//  - we have at most 15 bytes left to compare (R8 < 16)
_0through15:
    // if R8 <= 8 {
    //   goto _0through8
    // }
    CMPQ    R8, $8
    JBE     _0through8
    // AX = a[:8]
    MOVQ    (SI), AX
    // CX = b[:8]
    MOVQ    (DI), CX
    // if AX != CX {
    //   goto diff8
    // }
    CMPQ    AX, CX
    JNE     diff8

// Invariants:
//  - original slices contained at least 8 bytes (DX >= 8)
//  - we have at most 8 bytes left to compare (R8 <= 8)
//
// Because the backing slices have at least 8 bytes and all the bytes so far
// matched, we can (potentially) back up to where we have exactly 8 bytes to
// compare.
_0through8:
    // AX = b[len(b)-8:]
    MOVQ    -8(SI)(R8*1), AX
    // CX = b[len(b)-8:]
    MOVQ    -8(DI)(R8*1), CX
    // if AX == CX {
    //   goto allsame
    // }
    CMPQ    AX, CX
    JEQ     allsame
    // R8 = 8
    MOVQ    $8, R8

// Invariant: AX contains a bit mask of differences between a[:8] and b[:8].
//   AX & (1 << i) == 1 iff a[i] != b[i]
diff8:
    // CX ^= AX
    XORQ    AX, CX
    // Bit Scan Forward (return the index of the least significant set bit)
    //   CX = bits.TrailingZeros64(CX)
    BSFQ    CX, CX
    // CX /= 8
    SHRQ    $3, CX
    // CX is now the 0-based index of the first byte that differs.
    // R8 -= CX
    SUBQ    CX, R8

    // Return DX (original min(alen, blen)) - R8 (bytes left to compare)
    SUBQ    R8, DX
    MOVQ    DX, ret+48(FP)
    RET

// Invariant: original min(alen, blen) < 8. DX < 8, R8 = DX.
small:
    // CX = R8 * 8
    LEAQ    (R8*8), CX
    // CX = -CX
    // We only care about the lower 6 bits of CX, so this is equivalent to:
    // CX = (8-min(alen, blen)) * 8
    NEGQ    CX
    JEQ     allsame

    // We will load 8 bytes, even though some of them are outside the slice
    // bounds. We go out of bounds either before or after the slice depending on
    // the value of the pointer.

    // if uintptr(unsafe.Pointer(&a[0]) > 0xF8 {
    //   goto si_high
    // }
    CMPB    SI, $0xf8
    JA      si_high
    // SI = a[:8]
    MOVQ    (SI), SI
    // Discard the upper bytes which were out of bounds and add 0s (to be
    // removed below).
    SHLQ    CX, SI
    JMP     si_finish
si_high:
    // SI = a[len(a)-8:]
    MOVQ    -8(SI)(R8*1), SI
si_finish:
    // SI = SI >> CX
    // Discard the lower bytes which were added by SHLQ in one case, or that
    // were out of bounds in the si_high case.
    // In both cases, SI = a[:].
    SHRQ    CX, SI

    // if uintptr(unsafe.Pointer(&b[0]) > 0xF8 {
    //   goto di_high
    // }
    CMPB    DI, $0xf8
    JA      di_high
    // DI = b[:8]
    MOVQ    (DI), DI
    // Discard the upper bytes which were out of bounds and add 0s (to be
    // removed below).
    SHLQ    CX, DI
    JMP     di_finish
di_high:
    // DI = b[len(b)-8:]
    MOVQ    -8(DI)(R8*1), DI
di_finish:
    // DI = DI >> CX
    // Discard the lower bytes which were added by SHLQ in one case, or that
    // were out of bounds in the di_high case.
    // In both cases, DI = b[:].
    SHRQ    CX, DI

    // DI ^= SI
    XORQ    SI, DI
    // if DI == 0 {
    //   goto allsame
    // }
    JEQ     allsame

    // Bit Scan Forward (return the index of the least significant set bit)
    //   DI = bits.TrailingZeros64(DI)
    BSFQ    DI, DI
    // DI /= 8
    SHRQ    $3, DI
    // DI is now the 0-based index of the first byte that differs.
    // R8 -= DI
    SUBQ    DI, R8

    // Return DX (original min(alen, blen)) - R8 (bytes left to compare)
    SUBQ    R8, DX
allsame:
    MOVQ    DX, ret+48(FP)
    RET

// big_loop is used when we have at least 64 bytes to compare. It is similar to
// <loop>, except that we do 4 iterations at a time.
big_loop:
    MOVOU    (SI), X0
    MOVOU    (DI), X1
    PCMPEQB  X0, X1
    PMOVMSKB X1, AX
    XORQ     $0xffff, AX
    JNE      diff16

    MOVOU    16(SI), X0
    MOVOU    16(DI), X1
    PCMPEQB  X0, X1
    PMOVMSKB X1, AX
    XORQ     $0xffff, AX
    JNE      diff32

    MOVOU    32(SI), X0
    MOVOU    32(DI), X1
    PCMPEQB  X0, X1
    PMOVMSKB X1, AX
    XORQ     $0xffff, AX
    JNE      diff48

    MOVOU    48(SI), X0
    MOVOU    48(DI), X1
    PCMPEQB  X0, X1
    PMOVMSKB X1, AX
    XORQ     $0xffff, AX
    JNE      diff64

    // a = a[64:]
    ADDQ    $64, SI
    // b = b[64:]
    ADDQ    $64, DI
    // R8 -= 64
    SUBQ    $64, R8
    CMPQ    R8, $64
    // if R8 < 64 {
    //   goto loop
    // }
    JBE     loop
    JMP     big_loop
