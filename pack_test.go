package bitpack_test

import (
	"fmt"
	"math/rand"
	"slices"
	"testing"

	"github.com/parquet-go/bitpack"
)

func TestPackInt32(t *testing.T) {
	for bitWidth := uint(1); bitWidth <= 32; bitWidth++ {
		t.Run(fmt.Sprintf("bitWidth=%d", bitWidth), func(t *testing.T) {
			block := [blockSize]int32{}
			bitMask := int32((1 << bitWidth) - 1)

			prng := rand.New(rand.NewSource(0))
			for i := range block {
				block[i] = prng.Int31() & bitMask
			}

			// Test various lengths to exercise NEON batch processing and remainders
			for n := 1; n <= blockSize; n++ {
				// Pack the values
				size := bitpack.ByteCount(uint(n) * bitWidth)
				packed := make([]byte, size+bitpack.PaddingInt32)
				bitpack.Pack(packed, block[:n], bitWidth)

				// Unpack and verify
				unpacked := make([]int32, n)
				bitpack.Unpack(unpacked, packed, bitWidth)

				if !slices.Equal(block[:n], unpacked) {
					t.Fatalf("values mismatch for length=%d\nwant: %v\ngot:  %v", n, block[:n], unpacked)
				}
			}
		})
	}
}

func TestPackInt64(t *testing.T) {
	for bitWidth := uint(1); bitWidth <= 63; bitWidth++ {
		t.Run(fmt.Sprintf("bitWidth=%d", bitWidth), func(t *testing.T) {
			block := [blockSize]int64{}
			bitMask := int64((1 << bitWidth) - 1)

			prng := rand.New(rand.NewSource(0))
			for i := range block {
				block[i] = prng.Int63() & bitMask
			}

			// Test various lengths to exercise NEON batch processing and remainders
			for n := 1; n <= blockSize; n++ {
				// Pack the values
				size := bitpack.ByteCount(uint(n) * bitWidth)
				packed := make([]byte, size+bitpack.PaddingInt64)
				bitpack.Pack(packed, block[:n], bitWidth)

				// Unpack and verify
				unpacked := make([]int64, n)
				bitpack.Unpack(unpacked, packed, bitWidth)

				if !slices.Equal(block[:n], unpacked) {
					t.Fatalf("values mismatch for length=%d\nwant: %v\ngot:  %v", n, block[:n], unpacked)
				}
			}
		})
	}
}

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
				bitpack.Pack(dst, src, bitWidth)
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
				bitpack.Pack(dst, src, bitWidth)
			}

			b.SetBytes(8 * blockSize)
		})
	}
}
