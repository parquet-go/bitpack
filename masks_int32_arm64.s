//go:build !purego

#include "textflag.h"

// -----------------------------------------------------------------------------
// NEON Shuffle masks and shift tables for unpacking int32 values
//
// NEON uses 128-bit registers (vs AVX2's 256-bit), so we process 4 int32
// values per iteration instead of 8.
//
// TBL instruction: byte shuffle within 16 bytes using indices 0-15
// USHL instruction: variable shift per lane (negative = right shift)
// -----------------------------------------------------------------------------

// Shuffle masks for unpacking values from bit widths 1 to 16.
//
// For NEON, we process 4 int32 values at a time. Each mask is 16 bytes.
// The masks are indexed by: offset = 16 * (bitWidth - 1)
//
// Special value 0xFF means "load zero byte"
//
GLOBL ·shuffleInt32x1to16bitsNEON(SB), RODATA|NOPTR, $256

// 1 bit => 32 bits (4 values from 4 bits = 0.5 bytes)
// Values 0,1,2,3 packed in first byte
DATA ·shuffleInt32x1to16bitsNEON+0+0(SB)/4,  $0x808080FF // value 0: byte 0, bits 0
DATA ·shuffleInt32x1to16bitsNEON+0+4(SB)/4,  $0x808080FF // value 1: byte 0, bits 1
DATA ·shuffleInt32x1to16bitsNEON+0+8(SB)/4,  $0x808080FF // value 2: byte 0, bits 2
DATA ·shuffleInt32x1to16bitsNEON+0+12(SB)/4, $0x808080FF // value 3: byte 0, bits 3

// 2 bits => 32 bits (4 values from 8 bits = 1 byte)
DATA ·shuffleInt32x1to16bitsNEON+16+0(SB)/4,  $0x808080FF // value 0: byte 0, bits 0-1
DATA ·shuffleInt32x1to16bitsNEON+16+4(SB)/4,  $0x808080FF // value 1: byte 0, bits 2-3
DATA ·shuffleInt32x1to16bitsNEON+16+8(SB)/4,  $0x808080FF // value 2: byte 0, bits 4-5
DATA ·shuffleInt32x1to16bitsNEON+16+12(SB)/4, $0x808080FF // value 3: byte 0, bits 6-7

// 3 bits => 32 bits (4 values from 12 bits = 1.5 bytes)
DATA ·shuffleInt32x1to16bitsNEON+32+0(SB)/4,  $0x808080FF // value 0: byte 0, bits 0-2
DATA ·shuffleInt32x1to16bitsNEON+32+4(SB)/4,  $0x808080FF // value 1: byte 0-1, bits 3-5
DATA ·shuffleInt32x1to16bitsNEON+32+8(SB)/4,  $0x808080FF // value 2: byte 1, bits 6-7,0
DATA ·shuffleInt32x1to16bitsNEON+32+12(SB)/4, $0x808080FF // value 3: byte 1, bits 1-3

// 4 bits => 32 bits (4 values from 16 bits = 2 bytes)
DATA ·shuffleInt32x1to16bitsNEON+48+0(SB)/4,  $0x808080FF // value 0: byte 0, bits 0-3
DATA ·shuffleInt32x1to16bitsNEON+48+4(SB)/4,  $0x808080FF // value 1: byte 0, bits 4-7
DATA ·shuffleInt32x1to16bitsNEON+48+8(SB)/4,  $0x808080FF // value 2: byte 1, bits 0-3
DATA ·shuffleInt32x1to16bitsNEON+48+12(SB)/4, $0x808080FF // value 3: byte 1, bits 4-7

// 5-16 bits: Similar pattern, will implement incrementally
// For now, using placeholders - these will be filled in based on the algorithm

// 8 bits => 32 bits (4 values from 32 bits = 4 bytes)
DATA ·shuffleInt32x1to16bitsNEON+112+0(SB)/4,  $0x80808000 // value 0: byte 0
DATA ·shuffleInt32x1to16bitsNEON+112+4(SB)/4,  $0x80808001 // value 1: byte 1
DATA ·shuffleInt32x1to16bitsNEON+112+8(SB)/4,  $0x80808002 // value 2: byte 2
DATA ·shuffleInt32x1to16bitsNEON+112+12(SB)/4, $0x80808003 // value 3: byte 3

// 16 bits => 32 bits (4 values from 64 bits = 8 bytes)
DATA ·shuffleInt32x1to16bitsNEON+240+0(SB)/4,  $0x80800100 // value 0: bytes 0-1
DATA ·shuffleInt32x1to16bitsNEON+240+4(SB)/4,  $0x80800302 // value 1: bytes 2-3
DATA ·shuffleInt32x1to16bitsNEON+240+8(SB)/4,  $0x80800504 // value 2: bytes 4-5
DATA ·shuffleInt32x1to16bitsNEON+240+12(SB)/4, $0x80800706 // value 3: bytes 6-7

// Shift amounts for NEON USHL instruction
// USHL uses signed shift amounts: negative = right shift, positive = left shift
// Each entry contains 4 int32 shift amounts for 4 values
//
// Formula: shift[i] = -(i * bitWidth) % 8  (negative for right shift)
//
GLOBL ·shiftRightInt32NEON(SB), RODATA|NOPTR, $256

// 1 bit: shifts are 0, -1, -2, -3
DATA ·shiftRightInt32NEON+0+0(SB)/4,  $0          // value 0: shift right by 0
DATA ·shiftRightInt32NEON+0+4(SB)/4,  $0xFFFFFFFF // value 1: shift right by 1
DATA ·shiftRightInt32NEON+0+8(SB)/4,  $0xFFFFFFFE // value 2: shift right by 2
DATA ·shiftRightInt32NEON+0+12(SB)/4, $0xFFFFFFFD // value 3: shift right by 3

// 2 bits: shifts are 0, -2, -4, -6
DATA ·shiftRightInt32NEON+16+0(SB)/4,  $0          // value 0: shift right by 0
DATA ·shiftRightInt32NEON+16+4(SB)/4,  $0xFFFFFFFE // value 1: shift right by 2
DATA ·shiftRightInt32NEON+16+8(SB)/4,  $0xFFFFFFFC // value 2: shift right by 4
DATA ·shiftRightInt32NEON+16+12(SB)/4, $0xFFFFFFFA // value 3: shift right by 6

// 3 bits: shifts are 0, -3, -6, -1 (wraps at 8)
DATA ·shiftRightInt32NEON+32+0(SB)/4,  $0          // value 0: shift right by 0
DATA ·shiftRightInt32NEON+32+4(SB)/4,  $0xFFFFFFFD // value 1: shift right by 3
DATA ·shiftRightInt32NEON+32+8(SB)/4,  $0xFFFFFFFA // value 2: shift right by 6
DATA ·shiftRightInt32NEON+32+12(SB)/4, $0xFFFFFFFF // value 3: shift right by 1

// 4 bits: shifts are 0, -4, 0, -4 (wraps at 8)
DATA ·shiftRightInt32NEON+48+0(SB)/4,  $0          // value 0: shift right by 0
DATA ·shiftRightInt32NEON+48+4(SB)/4,  $0xFFFFFFFC // value 1: shift right by 4
DATA ·shiftRightInt32NEON+48+8(SB)/4,  $0          // value 2: shift right by 0
DATA ·shiftRightInt32NEON+48+12(SB)/4, $0xFFFFFFFC // value 3: shift right by 4

// 8 bits: no shift needed
DATA ·shiftRightInt32NEON+112+0(SB)/4,  $0 // value 0: shift right by 0
DATA ·shiftRightInt32NEON+112+4(SB)/4,  $0 // value 1: shift right by 0
DATA ·shiftRightInt32NEON+112+8(SB)/4,  $0 // value 2: shift right by 0
DATA ·shiftRightInt32NEON+112+12(SB)/4, $0 // value 3: shift right by 0

// 16 bits: no shift needed
DATA ·shiftRightInt32NEON+240+0(SB)/4,  $0 // value 0: shift right by 0
DATA ·shiftRightInt32NEON+240+4(SB)/4,  $0 // value 1: shift right by 0
DATA ·shiftRightInt32NEON+240+8(SB)/4,  $0 // value 2: shift right by 0
DATA ·shiftRightInt32NEON+240+12(SB)/4, $0 // value 3: shift right by 0
