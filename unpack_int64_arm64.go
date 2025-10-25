//go:build !purego

package bitpack

import (
	"github.com/parquet-go/bitpack/unsafecast"
)

//go:noescape
func unpackInt64Default(dst []int64, src []byte, bitWidth uint)

//go:noescape
func unpackInt64x1to32bitsARM64(dst []int64, src []byte, bitWidth uint)

func unpackInt64(dst []int64, src []byte, bitWidth uint) {
	// For ARM64, use optimized scalar operations for common bit widths
	switch {
	case bitWidth <= 32:
		unpackInt64x1to32bitsARM64(dst, src, bitWidth)
	case bitWidth == 64:
		copy(dst, unsafecast.Slice[int64](src))
	default:
		unpackInt64Default(dst, src, bitWidth)
	}
}
