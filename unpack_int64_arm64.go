//go:build !purego

package bitpack

import (
	"github.com/parquet-go/bitpack/unsafecast"
)

//go:noescape
func unpackInt64Default(dst []int64, src []byte, bitWidth uint)

func unpackInt64(dst []int64, src []byte, bitWidth uint) {
	// For ARM64, we'll use NEON instructions
	// TODO: Implement NEON optimizations - using default for now
	switch {
	case bitWidth == 64:
		copy(dst, unsafecast.Slice[int64](src))
	default:
		unpackInt64Default(dst, src, bitWidth)
	}
}
