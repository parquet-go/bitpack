//go:build !purego

package bitpack

import (
	"github.com/parquet-go/bitpack/unsafecast"
)

//go:noescape
func unpackInt32Default(dst []int32, src []byte, bitWidth uint)

//go:noescape
func unpackInt32x1to16bitsNEON(dst []int32, src []byte, bitWidth uint)

func unpackInt32(dst []int32, src []byte, bitWidth uint) {
	// For ARM64, we use simplified NEON for byte-aligned cases (8, 16 bits)
	// and scalar for other cases
	switch {
	case bitWidth <= 16:
		unpackInt32x1to16bitsNEON(dst, src, bitWidth)
	case bitWidth == 32:
		copy(dst, unsafecast.Slice[int32](src))
	default:
		unpackInt32Default(dst, src, bitWidth)
	}
}
