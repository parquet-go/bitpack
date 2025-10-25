package bitpack_test

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/parquet-go/bitpack"
)

func BenchmarkPackInt32(b *testing.B) {
	for bitWidth := uint(1); bitWidth <= 32; bitWidth++ {
		block := [blockSize]int32{}
		buf := [4*blockSize + bitpack.PaddingInt32]byte{}

		// Initialize with random data
		prng := rand.New(rand.NewSource(0))
		bitMask := int32((1 << bitWidth) - 1)
		for i := range block {
			block[i] = prng.Int31() & bitMask
		}

		b.Run(fmt.Sprintf("bitWidth=%d", bitWidth), func(b *testing.B) {
			dst := buf[:]
			src := block[:]

			for i := 0; i < b.N; i++ {
				bitpack.PackInt32(dst, src, bitWidth)
			}

			b.SetBytes(4 * blockSize)
		})
	}
}

func BenchmarkPackInt64(b *testing.B) {
	for bitWidth := uint(1); bitWidth <= 64; bitWidth++ {
		block := [blockSize]int64{}
		buf := [8*blockSize + bitpack.PaddingInt64]byte{}

		// Initialize with random data
		prng := rand.New(rand.NewSource(0))
		bitMask := int64((1 << bitWidth) - 1)
		if bitWidth == 64 {
			bitMask = -1
		}
		for i := range block {
			block[i] = prng.Int63() & bitMask
		}

		b.Run(fmt.Sprintf("bitWidth=%d", bitWidth), func(b *testing.B) {
			dst := buf[:]
			src := block[:]

			for i := 0; i < b.N; i++ {
				bitpack.PackInt64(dst, src, bitWidth)
			}

			b.SetBytes(8 * blockSize)
		})
	}
}
