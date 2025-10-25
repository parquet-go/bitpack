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
