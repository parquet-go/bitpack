//go:build !purego

package bitpack

import (
	"github.com/parquet-go/bitpack/unsafecast"
)

//go:noescape
func unpackInt32Default(dst []int32, src []byte, bitWidth uint)

func unpackInt32(dst []int32, src []byte, bitWidth uint) {
	// For ARM64, we'll use NEON instructions which are similar to AVX2
	// but operate on 128-bit registers instead of 256-bit
	// TODO: Implement NEON optimizations - using default for now
	switch {
	case bitWidth == 32:
		copy(dst, unsafecast.Slice[int32](src))
	default:
		unpackInt32Default(dst, src, bitWidth)
	}
}
