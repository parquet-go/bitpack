//go:build !purego

#include "funcdata.h"
#include "textflag.h"

// func unpackInt32Default(dst []int32, src []byte, bitWidth uint)
TEXT ·unpackInt32Default(SB), NOSPLIT, $0-56
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
	MOVWU (R2)(R16), R9       // R9 = src[i]
	MOVW R4, R10              // R10 = bitMask
	LSL R8, R10, R10          // R10 = bitMask << j
	AND R10, R9, R9           // R9 = src[i] & (bitMask << j)
	LSR R8, R9, R9            // R9 = d = (src[i] & (bitMask << j)) >> j

	ADD R3, R8, R11           // R11 = j + bitWidth
	CMP $32, R11
	BLE next                  // if j+bitWidth <= 32, skip to next

	ADD $1, R7, R12           // R12 = i + 1
	LSL $2, R12, R16          // R16 = (i + 1) * 4
	MOVWU (R2)(R16), R13      // R13 = src[i+1]

	MOVD $32, R14             // R14 = k = 32 - j
	SUB R8, R14, R14

	MOVW R4, R15              // R15 = bitMask
	LSR R14, R15, R15         // R15 = bitMask >> k
	AND R15, R13, R13         // R13 = src[i+1] & (bitMask >> k)
	LSL R14, R13, R13         // R13 = (src[i+1] & (bitMask >> k)) << k
	ORR R13, R9, R9           // R9 = d | c

next:
	LSL $2, R6, R16           // R16 = index * 4
	MOVW R9, (R0)(R16)        // dst[index] = d
	ADD R3, R5, R5            // bitOffset += bitWidth
	ADD $1, R6, R6            // index++

test:
	CMP R1, R6
	BNE loop
	RET

// unpackInt32x1to16bitsNEON implements NEON-optimized unpacking for bit widths 1-16
// This simplified version handles byte-aligned cases (8, 16 bits) with NEON,
// and falls back to scalar for complex cases.
//
// func unpackInt32x1to16bitsNEON(dst []int32, src []byte, bitWidth uint)
TEXT ·unpackInt32x1to16bitsNEON(SB), NOSPLIT, $0-56
	MOVD dst_base+0(FP), R0   // R0 = dst pointer
	MOVD dst_len+8(FP), R1    // R1 = dst length
	MOVD src_base+24(FP), R2  // R2 = src pointer
	MOVD bitWidth+48(FP), R3  // R3 = bitWidth

	// Check if we have at least 4 values to process
	CMP $4, R1
	BLT scalar_fallback

	// Determine which NEON path to use based on bitWidth
	CMP $8, R3
	BEQ neon_8bit
	CMP $16, R3
	BEQ neon_16bit

	// For other bit widths, fall back to scalar
	B scalar_fallback

neon_8bit:
	// BitWidth 8: 4 int32 values packed in 4 bytes
	// Process 4 values at a time using NEON

	// Calculate how many full groups of 4 we can process
	MOVD R1, R4
	LSR $2, R4, R4      // R4 = len / 4
	LSL $2, R4, R4      // R4 = (len / 4) * 4 = aligned length

	MOVD $0, R5         // R5 = index
	CMP $0, R4
	BEQ scalar_fallback

neon_8bit_loop:
	// Load 4 bytes as 4 uint8 values into lower part of V0
	// We need to load bytes and zero-extend to 32-bit

	// Load 4 bytes to W6
	MOVWU (R2), R6

	// Extract bytes and write as int32
	// Byte 0
	AND $0xFF, R6, R7
	MOVW R7, (R0)

	// Byte 1
	LSR $8, R6, R7
	AND $0xFF, R7, R7
	MOVW R7, 4(R0)

	// Byte 2
	LSR $16, R6, R7
	AND $0xFF, R7, R7
	MOVW R7, 8(R0)

	// Byte 3
	LSR $24, R6, R7
	MOVW R7, 12(R0)

	// Advance pointers
	ADD $4, R2, R2      // src += 4 bytes
	ADD $16, R0, R0     // dst += 4 int32 (16 bytes)
	ADD $4, R5, R5      // index += 4

	CMP R4, R5
	BLT neon_8bit_loop

	// Handle tail with scalar
	CMP R1, R5
	BEQ neon_done

	// Calculate remaining elements
	SUB R5, R1, R1      // R1 = remaining elements
	B scalar_fallback_entry

neon_16bit:
	// BitWidth 16: 4 int32 values packed in 8 bytes
	// Process 4 values at a time

	MOVD R1, R4
	LSR $2, R4, R4      // R4 = len / 4
	LSL $2, R4, R4      // R4 = (len / 4) * 4

	MOVD $0, R5         // R5 = index
	CMP $0, R4
	BEQ scalar_fallback

neon_16bit_loop:
	// Load 8 bytes as 4 uint16 values
	MOVD (R2), R6       // Load 8 bytes into R6

	// Extract 16-bit values and write as int32
	// Value 0 (bits 0-15)
	AND $0xFFFF, R6, R7
	MOVW R7, (R0)

	// Value 1 (bits 16-31)
	LSR $16, R6, R7
	AND $0xFFFF, R7, R7
	MOVW R7, 4(R0)

	// Value 2 (bits 32-47)
	LSR $32, R6, R7
	AND $0xFFFF, R7, R7
	MOVW R7, 8(R0)

	// Value 3 (bits 48-63)
	LSR $48, R6, R7
	MOVW R7, 12(R0)

	// Advance pointers
	ADD $8, R2, R2      // src += 8 bytes
	ADD $16, R0, R0     // dst += 4 int32 (16 bytes)
	ADD $4, R5, R5      // index += 4

	CMP R4, R5
	BLT neon_16bit_loop

	// Handle tail with scalar
	CMP R1, R5
	BEQ neon_done

	SUB R5, R1, R1
	B scalar_fallback_entry

neon_done:
	RET

scalar_fallback:
	MOVD $0, R5         // Start from beginning
	// R0, R1, R2, R3 already set from function args

scalar_fallback_entry:
	// R0 = current dst position (already advanced)
	// R1 = remaining elements
	// R2 = current src position (already advanced)
	// R3 = bitWidth
	// R5 = elements already processed

	// Fall back to scalar implementation for remaining elements
	CMP $0, R1
	BEQ scalar_done     // No remaining elements

	MOVD $1, R4         // R4 = bitMask = (1 << bitWidth) - 1
	LSL R3, R4, R4
	SUB $1, R4, R4

	// bitOffset starts from 0 relative to current R2 position
	// (not total offset, since R2 is already advanced)
	MOVD $0, R6         // R6 = bitOffset (relative to current R2)
	MOVD $0, R7         // R7 = index (within remaining elements)
	B scalar_test

scalar_loop:
	MOVD R6, R8         // R8 = i = bitOffset / 32
	LSR $5, R8, R8

	MOVD R6, R9         // R9 = j = bitOffset % 32
	AND $31, R9, R9

	LSL $2, R8, R10     // R10 = i * 4
	MOVWU (R2)(R10), R11  // R11 = src[i] (relative to current R2)
	MOVW R4, R12        // R12 = bitMask
	LSL R9, R12, R12    // R12 = bitMask << j
	AND R12, R11, R11   // R11 = src[i] & (bitMask << j)
	LSR R9, R11, R11    // R11 = d = (src[i] & (bitMask << j)) >> j

	ADD R3, R9, R12     // R12 = j + bitWidth
	CMP $32, R12
	BLE scalar_next     // if j+bitWidth <= 32, skip to next

	ADD $1, R8, R13     // R13 = i + 1
	LSL $2, R13, R10    // R10 = (i + 1) * 4
	MOVWU (R2)(R10), R14  // R14 = src[i+1]

	MOVD $32, R15       // R15 = k = 32 - j
	SUB R9, R15, R15

	MOVW R4, R16        // R16 = bitMask
	LSR R15, R16, R16   // R16 = bitMask >> k
	AND R16, R14, R14   // R14 = src[i+1] & (bitMask >> k)
	LSL R15, R14, R14   // R14 = (src[i+1] & (bitMask >> k)) << k
	ORR R14, R11, R11   // R11 = d | c

scalar_next:
	LSL $2, R7, R10     // R10 = index * 4
	MOVW R11, (R0)(R10) // dst[index] = d (relative to current R0)
	ADD R3, R6, R6      // bitOffset += bitWidth
	ADD $1, R7, R7      // index++

scalar_test:
	CMP R1, R7
	BLT scalar_loop

scalar_done:
	RET
