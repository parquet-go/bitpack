package bitpack_test

import (
	"fmt"
	"math/rand"
	"slices"
	"testing"

	"github.com/parquet-go/bitpack"
)

const (
	blockSize = 128
)

func TestUnpackInt32(t *testing.T) {
	for bitWidth := uint(1); bitWidth <= 32; bitWidth++ {
		t.Run(fmt.Sprintf("bitWidth=%d", bitWidth), func(t *testing.T) {
			block := [blockSize]int32{}
			bitMask := int32(bitWidth<<1) - 1

			prng := rand.New(rand.NewSource(0))
			for i := range block {
				block[i] = prng.Int31() & bitMask
			}

			size := (blockSize * bitWidth) / 8
			buf := make([]byte, size+bitpack.PaddingInt32)
			bitpack.PackInt32(buf, block[:], bitWidth)

			src := buf[:size]
			dst := make([]int32, blockSize)

			for n := 1; n <= blockSize; n++ {
				for i := range dst {
					dst[i] = 0
				}

				bitpack.UnpackInt32(dst[:n], src, bitWidth)

				if !slices.Equal(block[:n], dst[:n]) {
					t.Fatalf("values mismatch for length=%d\nwant: %v\ngot:  %v", n, block[:n], dst[:n])
				}
			}
		})
	}
}

func TestUnpackInt64(t *testing.T) {
	for bitWidth := uint(1); bitWidth <= 63; bitWidth++ {
		t.Run(fmt.Sprintf("bitWidth=%d", bitWidth), func(t *testing.T) {
			block := [blockSize]int64{}
			bitMask := int64(bitWidth<<1) - 1

			prng := rand.New(rand.NewSource(0))
			for i := range block {
				block[i] = prng.Int63() & bitMask
			}

			size := (blockSize * bitWidth) / 8
			buf := make([]byte, size+bitpack.PaddingInt64)
			bitpack.PackInt64(buf, block[:], bitWidth)

			src := buf[:size]
			dst := make([]int64, blockSize)

			for n := 1; n <= blockSize; n++ {
				for i := range dst {
					dst[i] = 0
				}

				bitpack.UnpackInt64(dst[:n], src, bitWidth)

				if !slices.Equal(block[:n], dst[:n]) {
					t.Fatalf("values mismatch for length=%d\nwant: %v\ngot:  %v", n, block[:n], dst[:n])
				}
			}
		})
	}
}

func FuzzUnpackUint64(f *testing.F) {
	// Add seed corpus
	f.Add(uint(10), uint(3), int64(6))
	f.Add(uint(20), uint(8), int64(0))
	f.Add(uint(30), uint(23), int64(-300))

	f.Fuzz(func(t *testing.T, size uint, bitWidth uint, seed int64) {
		if bitWidth == 0 || bitWidth > 64 {
			return
		}
		src := make([]int64, size)
		gen := rand.New(rand.NewSource(seed))
		bitMask := int64(bitWidth<<1) - 1
		for i := range src {
			src[i] = gen.Int63() & bitMask
		}

		packed := make([]byte, size*8+bitpack.PaddingInt64)
		bitpack.PackInt64(packed, src[:], bitWidth)

		unpacked := make([]int64, size)
		bitpack.UnpackInt64(unpacked[:], packed[:], bitWidth)

		if !slices.Equal(unpacked, src) {
			t.Fatalf("Roundtrip failed: got %v, want %v", unpacked, src)
		}
	})
}

func FuzzUnpackUint32(f *testing.F) {
	// Add seed corpus
	f.Add(uint(10), uint(3), int64(6))
	f.Add(uint(20), uint(8), int64(0))
	f.Add(uint(30), uint(23), int64(-300))

	f.Fuzz(func(t *testing.T, size uint, bitWidth uint, seed int64) {
		if bitWidth == 0 || bitWidth > 32 {
			return
		}
		src := make([]int32, size)
		gen := rand.New(rand.NewSource(seed))
		bitMask := int32(bitWidth<<1) - 1
		for i := range src {
			src[i] = gen.Int31() & bitMask
		}

		packed := make([]byte, size*8+bitpack.PaddingInt64)
		bitpack.PackInt32(packed, src[:], bitWidth)

		unpacked := make([]int32, size)
		bitpack.UnpackInt32(unpacked[:], packed[:], bitWidth)

		if !slices.Equal(unpacked, src) {
			t.Fatalf("Roundtrip failed: got %v, want %v", unpacked, src)
		}
	})
}

func BenchmarkUnpackInt32(b *testing.B) {
	for bitWidth := uint(1); bitWidth <= 32; bitWidth++ {
		block := [blockSize]int32{}
		buf := [4*blockSize + bitpack.PaddingInt32]byte{}
		bitpack.PackInt32(buf[:], block[:], bitWidth)

		b.Run(fmt.Sprintf("bitWidth=%d", bitWidth), func(b *testing.B) {
			dst := block[:]
			src := buf[:]

			for i := 0; i < b.N; i++ {
				bitpack.UnpackInt32(dst, src, bitWidth)
			}

			b.SetBytes(4 * blockSize)
		})
	}
}

func BenchmarkUnpackInt64(b *testing.B) {
	for bitWidth := uint(1); bitWidth <= 64; bitWidth++ {
		block := [blockSize]int64{}
		buf := [8*blockSize + bitpack.PaddingInt64]byte{}
		bitpack.PackInt64(buf[:], block[:], bitWidth)

		b.Run(fmt.Sprintf("bitWidth=%d", bitWidth), func(b *testing.B) {
			dst := block[:]
			src := buf[:]

			for i := 0; i < b.N; i++ {
				bitpack.UnpackInt64(dst, src, bitWidth)
			}

			b.SetBytes(4 * blockSize)
		})
	}
}
