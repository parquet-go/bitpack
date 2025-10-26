//go:build !purego

#include "funcdata.h"
#include "textflag.h"

// func unpackInt64Default(dst []int64, src []byte, bitWidth uint)
TEXT ·unpackInt64Default(SB), NOSPLIT, $0-56
	MOVD dst_base+0(FP), R0   // R0 = dst pointer
	MOVD dst_len+8(FP), R1    // R1 = dst length
	MOVD src_base+24(FP), R2  // R2 = src pointer
	MOVD bitWidth+48(FP), R3  // R3 = bitWidth

	MOVD $1, R4               // R4 = bitMask = (1 << bitWidth) - 1
	LSL R3, R4, R4
	SUB $1, R4, R4

	MOVD $0, R5               // R5 = bitOffset
	MOVD $0, R6               // R6 = index
	B test

loop:
	MOVD R5, R7               // R7 = i = bitOffset / 32
	LSR $5, R7, R7

	MOVD R5, R8               // R8 = j = bitOffset % 32
	AND $31, R8, R8

	LSL $2, R7, R16           // R16 = i * 4
	MOVWU (R2)(R16), R9       // R9 = src[i] (load as 32-bit, zero-extend to 64)
	MOVD R4, R10              // R10 = bitMask
	LSL R8, R10, R10          // R10 = bitMask << j
	AND R10, R9, R9           // R9 = src[i] & (bitMask << j)
	LSR R8, R9, R9            // R9 = d = (src[i] & (bitMask << j)) >> j

	ADD R3, R8, R11           // R11 = j + bitWidth
	CMP $32, R11
	BLE check64               // if j+bitWidth <= 32, check if > 64

	ADD $1, R7, R12           // R12 = i + 1
	LSL $2, R12, R16          // R16 = (i + 1) * 4
	MOVWU (R2)(R16), R13      // R13 = src[i+1]

	MOVD $32, R14             // R14 = k = 32 - j
	SUB R8, R14, R14

	MOVD R4, R15              // R15 = bitMask
	LSR R14, R15, R15         // R15 = bitMask >> k
	AND R15, R13, R13         // R13 = src[i+1] & (bitMask >> k)
	LSL R14, R13, R13         // R13 = (src[i+1] & (bitMask >> k)) << k
	ORR R13, R9, R9           // R9 = d | c

check64:
	CMP $64, R11
	BLE next                  // if j+bitWidth <= 64, skip to next

	ADD $2, R7, R12           // R12 = i + 2 (reuse R12)
	LSL $2, R12, R16          // R16 = (i + 2) * 4
	MOVWU (R2)(R16), R13      // R13 = src[i+2] (reuse R13)

	MOVD $64, R14             // R14 = k = 64 - j (reuse R14)
	SUB R8, R14, R14

	MOVD R4, R15              // R15 = bitMask (reuse R15)
	LSR R14, R15, R15         // R15 = bitMask >> k
	AND R15, R13, R13         // R13 = src[i+2] & (bitMask >> k)
	LSL R14, R13, R13         // R13 = (src[i+2] & (bitMask >> k)) << k
	ORR R13, R9, R9           // R9 = d | c

next:
	LSL $3, R6, R16           // R16 = index * 8
	MOVD R9, (R0)(R16)        // dst[index] = d (64-bit store)
	ADD R3, R5, R5            // bitOffset += bitWidth
	ADD $1, R6, R6            // index++

test:
	CMP R1, R6
	BNE loop
	RET

// unpackInt64x1to32bitsARM64 implements optimized unpacking for bit widths 1-32
// Uses optimized scalar ARM64 operations with batched processing
//
// func unpackInt64x1to32bitsARM64(dst []int64, src []byte, bitWidth uint)
TEXT ·unpackInt64x1to32bitsARM64(SB), NOSPLIT, $0-56
	MOVD dst_base+0(FP), R0   // R0 = dst pointer
	MOVD dst_len+8(FP), R1    // R1 = dst length
	MOVD src_base+24(FP), R2  // R2 = src pointer
	MOVD bitWidth+48(FP), R3  // R3 = bitWidth

	// Check if we have at least 4 values to process
	CMP $4, R1
	BLT scalar_fallback_int64

	// Determine which path to use based on bitWidth
	CMP $1, R3
	BEQ int64_1bit
	CMP $2, R3
	BEQ int64_2bit
	CMP $3, R3
	BEQ int64_3bit
	CMP $4, R3
	BEQ int64_4bit
	CMP $5, R3
	BEQ int64_5bit
	CMP $6, R3
	BEQ int64_6bit
	CMP $7, R3
	BEQ int64_7bit
	CMP $8, R3
	BEQ int64_8bit
	CMP $16, R3
	BEQ int64_16bit
	CMP $32, R3
	BEQ int64_32bit

	// For other bit widths, fall back to scalar
	B scalar_fallback_int64

int64_1bit:
	// BitWidth 1: 8 int64 values packed in 1 byte
	// Process 8 values at a time

	// Round down to multiple of 8 for processing
	MOVD R1, R4
	LSR $3, R4, R4      // R4 = len / 8
	LSL $3, R4, R4      // R4 = aligned length (multiple of 8)

	MOVD $0, R5
	CMP $0, R4
	BEQ scalar_fallback_int64

int64_1bit_loop:
	// Load 1 byte (contains 8 values, 1 bit each)
	MOVBU (R2), R6

	// Extract 8 bits
	AND $1, R6, R7
	MOVD R7, (R0)
	LSR $1, R6, R7
	AND $1, R7, R7
	MOVD R7, 8(R0)
	LSR $2, R6, R7
	AND $1, R7, R7
	MOVD R7, 16(R0)
	LSR $3, R6, R7
	AND $1, R7, R7
	MOVD R7, 24(R0)
	LSR $4, R6, R7
	AND $1, R7, R7
	MOVD R7, 32(R0)
	LSR $5, R6, R7
	AND $1, R7, R7
	MOVD R7, 40(R0)
	LSR $6, R6, R7
	AND $1, R7, R7
	MOVD R7, 48(R0)
	LSR $7, R6, R7
	AND $1, R7, R7
	MOVD R7, 56(R0)

	ADD $1, R2, R2
	ADD $64, R0, R0
	ADD $8, R5, R5

	CMP R4, R5
	BLT int64_1bit_loop

	CMP R1, R5
	BEQ int64_done
	SUB R5, R1, R1
	B scalar_fallback_entry_int64

int64_2bit:
	// BitWidth 2: 8 int64 values packed in 2 bytes
	MOVD R1, R4
	LSR $3, R4, R4
	LSL $3, R4, R4

	MOVD $0, R5
	CMP $0, R4
	BEQ scalar_fallback_int64

int64_2bit_loop:
	MOVHU (R2), R6

	AND $3, R6, R7
	MOVD R7, (R0)
	LSR $2, R6, R7
	AND $3, R7, R7
	MOVD R7, 8(R0)
	LSR $4, R6, R7
	AND $3, R7, R7
	MOVD R7, 16(R0)
	LSR $6, R6, R7
	AND $3, R7, R7
	MOVD R7, 24(R0)
	LSR $8, R6, R7
	AND $3, R7, R7
	MOVD R7, 32(R0)
	LSR $10, R6, R7
	AND $3, R7, R7
	MOVD R7, 40(R0)
	LSR $12, R6, R7
	AND $3, R7, R7
	MOVD R7, 48(R0)
	LSR $14, R6, R7
	AND $3, R7, R7
	MOVD R7, 56(R0)

	ADD $2, R2, R2
	ADD $64, R0, R0
	ADD $8, R5, R5

	CMP R4, R5
	BLT int64_2bit_loop

	CMP R1, R5
	BEQ int64_done
	SUB R5, R1, R1
	B scalar_fallback_entry_int64

int64_3bit:
	// BitWidth 3: 8 int64 values packed in 3 bytes
	MOVD R1, R4
	LSR $3, R4, R4
	LSL $3, R4, R4

	MOVD $0, R5
	CMP $0, R4
	BEQ scalar_fallback_int64

int64_3bit_loop:
	MOVWU (R2), R6

	AND $7, R6, R7
	MOVD R7, (R0)
	LSR $3, R6, R7
	AND $7, R7, R7
	MOVD R7, 8(R0)
	LSR $6, R6, R7
	AND $7, R7, R7
	MOVD R7, 16(R0)
	LSR $9, R6, R7
	AND $7, R7, R7
	MOVD R7, 24(R0)
	LSR $12, R6, R7
	AND $7, R7, R7
	MOVD R7, 32(R0)
	LSR $15, R6, R7
	AND $7, R7, R7
	MOVD R7, 40(R0)
	LSR $18, R6, R7
	AND $7, R7, R7
	MOVD R7, 48(R0)
	LSR $21, R6, R7
	AND $7, R7, R7
	MOVD R7, 56(R0)

	ADD $3, R2, R2
	ADD $64, R0, R0
	ADD $8, R5, R5

	CMP R4, R5
	BLT int64_3bit_loop

	CMP R1, R5
	BEQ int64_done
	SUB R5, R1, R1
	B scalar_fallback_entry_int64

int64_4bit:
	// BitWidth 4: 8 int64 values packed in 4 bytes
	MOVD R1, R4
	LSR $3, R4, R4
	LSL $3, R4, R4

	MOVD $0, R5
	CMP $0, R4
	BEQ scalar_fallback_int64

int64_4bit_loop:
	MOVWU (R2), R6

	AND $15, R6, R7
	MOVD R7, (R0)
	LSR $4, R6, R7
	AND $15, R7, R7
	MOVD R7, 8(R0)
	LSR $8, R6, R7
	AND $15, R7, R7
	MOVD R7, 16(R0)
	LSR $12, R6, R7
	AND $15, R7, R7
	MOVD R7, 24(R0)
	LSR $16, R6, R7
	AND $15, R7, R7
	MOVD R7, 32(R0)
	LSR $20, R6, R7
	AND $15, R7, R7
	MOVD R7, 40(R0)
	LSR $24, R6, R7
	AND $15, R7, R7
	MOVD R7, 48(R0)
	LSR $28, R6, R7
	AND $15, R7, R7
	MOVD R7, 56(R0)

	ADD $4, R2, R2
	ADD $64, R0, R0
	ADD $8, R5, R5

	CMP R4, R5
	BLT int64_4bit_loop

	CMP R1, R5
	BEQ int64_done
	SUB R5, R1, R1
	B scalar_fallback_entry_int64

int64_5bit:
	// BitWidth 5: 8 int64 values packed in 5 bytes
	MOVD R1, R4
	LSR $3, R4, R4
	LSL $3, R4, R4

	MOVD $0, R5
	CMP $0, R4
	BEQ scalar_fallback_int64

int64_5bit_loop:
	MOVD (R2), R6

	AND $31, R6, R7
	MOVD R7, (R0)
	LSR $5, R6, R7
	AND $31, R7, R7
	MOVD R7, 8(R0)
	LSR $10, R6, R7
	AND $31, R7, R7
	MOVD R7, 16(R0)
	LSR $15, R6, R7
	AND $31, R7, R7
	MOVD R7, 24(R0)
	LSR $20, R6, R7
	AND $31, R7, R7
	MOVD R7, 32(R0)
	LSR $25, R6, R7
	AND $31, R7, R7
	MOVD R7, 40(R0)
	LSR $30, R6, R7
	AND $31, R7, R7
	MOVD R7, 48(R0)
	LSR $35, R6, R7
	AND $31, R7, R7
	MOVD R7, 56(R0)

	ADD $5, R2, R2
	ADD $64, R0, R0
	ADD $8, R5, R5

	CMP R4, R5
	BLT int64_5bit_loop

	CMP R1, R5
	BEQ int64_done
	SUB R5, R1, R1
	B scalar_fallback_entry_int64

int64_6bit:
	// BitWidth 6: 8 int64 values packed in 6 bytes
	MOVD R1, R4
	LSR $3, R4, R4
	LSL $3, R4, R4

	MOVD $0, R5
	CMP $0, R4
	BEQ scalar_fallback_int64

int64_6bit_loop:
	MOVD (R2), R6

	AND $63, R6, R7
	MOVD R7, (R0)
	LSR $6, R6, R7
	AND $63, R7, R7
	MOVD R7, 8(R0)
	LSR $12, R6, R7
	AND $63, R7, R7
	MOVD R7, 16(R0)
	LSR $18, R6, R7
	AND $63, R7, R7
	MOVD R7, 24(R0)
	LSR $24, R6, R7
	AND $63, R7, R7
	MOVD R7, 32(R0)
	LSR $30, R6, R7
	AND $63, R7, R7
	MOVD R7, 40(R0)
	LSR $36, R6, R7
	AND $63, R7, R7
	MOVD R7, 48(R0)
	LSR $42, R6, R7
	AND $63, R7, R7
	MOVD R7, 56(R0)

	ADD $6, R2, R2
	ADD $64, R0, R0
	ADD $8, R5, R5

	CMP R4, R5
	BLT int64_6bit_loop

	CMP R1, R5
	BEQ int64_done
	SUB R5, R1, R1
	B scalar_fallback_entry_int64

int64_7bit:
	// BitWidth 7: 8 int64 values packed in 7 bytes
	MOVD R1, R4
	LSR $3, R4, R4
	LSL $3, R4, R4

	MOVD $0, R5
	CMP $0, R4
	BEQ scalar_fallback_int64

int64_7bit_loop:
	MOVD (R2), R6

	AND $127, R6, R7
	MOVD R7, (R0)
	LSR $7, R6, R7
	AND $127, R7, R7
	MOVD R7, 8(R0)
	LSR $14, R6, R7
	AND $127, R7, R7
	MOVD R7, 16(R0)
	LSR $21, R6, R7
	AND $127, R7, R7
	MOVD R7, 24(R0)
	LSR $28, R6, R7
	AND $127, R7, R7
	MOVD R7, 32(R0)
	LSR $35, R6, R7
	AND $127, R7, R7
	MOVD R7, 40(R0)
	LSR $42, R6, R7
	AND $127, R7, R7
	MOVD R7, 48(R0)
	LSR $49, R6, R7
	AND $127, R7, R7
	MOVD R7, 56(R0)

	ADD $7, R2, R2
	ADD $64, R0, R0
	ADD $8, R5, R5

	CMP R4, R5
	BLT int64_7bit_loop

	CMP R1, R5
	BEQ int64_done
	SUB R5, R1, R1
	B scalar_fallback_entry_int64

int64_8bit:
	// BitWidth 8: 8 int64 values packed in 8 bytes
	// Process 8 values at a time

	// Round down to multiple of 8 for processing
	MOVD R1, R4
	LSR $3, R4, R4      // R4 = len / 8
	LSL $3, R4, R4      // R4 = aligned length (multiple of 8)

	MOVD $0, R5         // R5 = index
	CMP $0, R4
	BEQ scalar_fallback_int64

int64_8bit_loop:
	// Load 8 bytes (contains 8 values, 1 byte each)
	MOVD (R2), R6

	// Extract 8 bytes and store as int64
	// Value 0: byte 0
	AND $0xFF, R6, R7
	MOVD R7, (R0)

	// Value 1: byte 1
	LSR $8, R6, R7
	AND $0xFF, R7, R7
	MOVD R7, 8(R0)

	// Value 2: byte 2
	LSR $16, R6, R7
	AND $0xFF, R7, R7
	MOVD R7, 16(R0)

	// Value 3: byte 3
	LSR $24, R6, R7
	AND $0xFF, R7, R7
	MOVD R7, 24(R0)

	// Value 4: byte 4
	LSR $32, R6, R7
	AND $0xFF, R7, R7
	MOVD R7, 32(R0)

	// Value 5: byte 5
	LSR $40, R6, R7
	AND $0xFF, R7, R7
	MOVD R7, 40(R0)

	// Value 6: byte 6
	LSR $48, R6, R7
	AND $0xFF, R7, R7
	MOVD R7, 48(R0)

	// Value 7: byte 7
	LSR $56, R6, R7
	MOVD R7, 56(R0)

	// Advance pointers
	ADD $8, R2, R2      // src += 8 bytes (8 values)
	ADD $64, R0, R0     // dst += 8 int64 (64 bytes)
	ADD $8, R5, R5      // index += 8

	CMP R4, R5
	BLT int64_8bit_loop

	// Handle tail with scalar
	CMP R1, R5
	BEQ int64_done

	SUB R5, R1, R1
	B scalar_fallback_entry_int64

int64_16bit:
	// BitWidth 16: 4 int64 values packed in 8 bytes
	// Process 4 values at a time

	MOVD R1, R4
	LSR $2, R4, R4      // R4 = len / 4
	LSL $2, R4, R4      // R4 = aligned length (multiple of 4)

	MOVD $0, R5         // R5 = index
	CMP $0, R4
	BEQ scalar_fallback_int64

int64_16bit_loop:
	// Load 8 bytes as 4 uint16 values
	MOVD (R2), R6

	// Extract 16-bit values and write as int64
	// Value 0 (bits 0-15)
	AND $0xFFFF, R6, R7
	MOVD R7, (R0)

	// Value 1 (bits 16-31)
	LSR $16, R6, R7
	AND $0xFFFF, R7, R7
	MOVD R7, 8(R0)

	// Value 2 (bits 32-47)
	LSR $32, R6, R7
	AND $0xFFFF, R7, R7
	MOVD R7, 16(R0)

	// Value 3 (bits 48-63)
	LSR $48, R6, R7
	MOVD R7, 24(R0)

	// Advance pointers
	ADD $8, R2, R2      // src += 8 bytes (4 values)
	ADD $32, R0, R0     // dst += 4 int64 (32 bytes)
	ADD $4, R5, R5      // index += 4

	CMP R4, R5
	BLT int64_16bit_loop

	// Handle tail with scalar
	CMP R1, R5
	BEQ int64_done

	SUB R5, R1, R1
	B scalar_fallback_entry_int64

int64_32bit:
	// BitWidth 32: 2 int64 values packed in 8 bytes
	// Process 2 values at a time

	MOVD R1, R4
	LSR $1, R4, R4      // R4 = len / 2
	LSL $1, R4, R4      // R4 = aligned length (multiple of 2)

	MOVD $0, R5         // R5 = index
	CMP $0, R4
	BEQ scalar_fallback_int64

int64_32bit_loop:
	// Load 8 bytes as 2 uint32 values
	MOVD (R2), R6

	// Extract 32-bit values and write as int64
	// Value 0 (bits 0-31)
	AND $0xFFFFFFFF, R6, R7
	MOVD R7, (R0)

	// Value 1 (bits 32-63)
	LSR $32, R6, R7
	MOVD R7, 8(R0)

	// Advance pointers
	ADD $8, R2, R2      // src += 8 bytes (2 values)
	ADD $16, R0, R0     // dst += 2 int64 (16 bytes)
	ADD $2, R5, R5      // index += 2

	CMP R4, R5
	BLT int64_32bit_loop

	// Handle tail with scalar
	CMP R1, R5
	BEQ int64_done

	SUB R5, R1, R1
	B scalar_fallback_entry_int64

int64_done:
	RET

scalar_fallback_int64:
	MOVD $0, R5         // Start from beginning

scalar_fallback_entry_int64:
	// R0 = current dst position (already advanced)
	// R1 = remaining elements
	// R2 = current src position (already advanced)
	// R3 = bitWidth
	// R5 = elements already processed

	// Fall back to scalar implementation for remaining elements
	CMP $0, R1
	BEQ scalar_done_int64     // No remaining elements

	MOVD $1, R4         // R4 = bitMask = (1 << bitWidth) - 1
	LSL R3, R4, R4
	SUB $1, R4, R4

	// bitOffset starts from 0 relative to current R2 position
	MOVD $0, R6         // R6 = bitOffset (relative to current R2)
	MOVD $0, R7         // R7 = index (within remaining elements)
	B scalar_test_int64

scalar_loop_int64:
	MOVD R6, R8         // R8 = i = bitOffset / 32
	LSR $5, R8, R8

	MOVD R6, R9         // R9 = j = bitOffset % 32
	AND $31, R9, R9

	LSL $2, R8, R10     // R10 = i * 4
	MOVWU (R2)(R10), R11  // R11 = src[i]
	MOVD R4, R12        // R12 = bitMask
	LSL R9, R12, R12    // R12 = bitMask << j
	AND R12, R11, R11   // R11 = src[i] & (bitMask << j)
	LSR R9, R11, R11    // R11 = d = (src[i] & (bitMask << j)) >> j

	ADD R3, R9, R12     // R12 = j + bitWidth
	CMP $32, R12
	BLE scalar_check64_int64

	ADD $1, R8, R13     // R13 = i + 1
	LSL $2, R13, R10    // R10 = (i + 1) * 4
	MOVWU (R2)(R10), R14  // R14 = src[i+1]

	MOVD $32, R15       // R15 = k = 32 - j
	SUB R9, R15, R15

	MOVD R4, R16        // R16 = bitMask
	LSR R15, R16, R16   // R16 = bitMask >> k
	AND R16, R14, R14   // R14 = src[i+1] & (bitMask >> k)
	LSL R15, R14, R14   // R14 = (src[i+1] & (bitMask >> k)) << k
	ORR R14, R11, R11   // R11 = d | c

scalar_check64_int64:
	CMP $64, R12
	BLE scalar_next_int64

	ADD $2, R8, R13     // R13 = i + 2
	LSL $2, R13, R10    // R10 = (i + 2) * 4
	MOVWU (R2)(R10), R14  // R14 = src[i+2]

	MOVD $64, R15       // R15 = k = 64 - j
	SUB R9, R15, R15

	MOVD R4, R16        // R16 = bitMask
	LSR R15, R16, R16   // R16 = bitMask >> k
	AND R16, R14, R14   // R14 = src[i+2] & (bitMask >> k)
	LSL R15, R14, R14   // R14 = (src[i+2] & (bitMask >> k)) << k
	ORR R14, R11, R11   // R11 = d | c

scalar_next_int64:
	LSL $3, R7, R10     // R10 = index * 8
	MOVD R11, (R0)(R10) // dst[index] = d (relative to current R0)
	ADD R3, R6, R6      // bitOffset += bitWidth
	ADD $1, R7, R7      // index++

scalar_test_int64:
	CMP R1, R7
	BLT scalar_loop_int64

scalar_done_int64:
	RET

// Macro definitions for unsupported NEON instructions using WORD encodings
// USHLL Vd.8H, Vn.8B, #0 - widen 8x8-bit to 8x16-bit
#define USHLL_8H_8B(vd, vn) WORD $(0x2f08a400 | (vd) | ((vn)<<5))

// USHLL2 Vd.8H, Vn.16B, #0 - widen upper 8x8-bit to 8x16-bit
#define USHLL2_8H_16B(vd, vn) WORD $(0x6f08a400 | (vd) | ((vn)<<5))

// USHLL Vd.4S, Vn.4H, #0 - widen 4x16-bit to 4x32-bit
#define USHLL_4S_4H(vd, vn) WORD $(0x2f10a400 | (vd) | ((vn)<<5))

// USHLL2 Vd.4S, Vn.8H, #0 - widen upper 4x16-bit to 4x32-bit
#define USHLL2_4S_8H(vd, vn) WORD $(0x6f10a400 | (vd) | ((vn)<<5))

// USHLL Vd.2D, Vn.2S, #0 - widen 2x32-bit to 2x64-bit
#define USHLL_2D_2S(vd, vn) WORD $(0x2f20a400 | (vd) | ((vn)<<5))

// USHLL2 Vd.2D, Vn.4S, #0 - widen upper 2x32-bit to 2x64-bit
#define USHLL2_2D_4S(vd, vn) WORD $(0x6f20a400 | (vd) | ((vn)<<5))

// unpackInt64x1bitNEON implements table-based NEON unpacking for int64 bitWidth=1
// Similar to int32 version but with additional widening to 64-bit
//
// func unpackInt64x1bitNEON(dst []int64, src []byte, bitWidth uint)
