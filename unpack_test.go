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
			bitMask := int32(uint32(1)<<bitWidth - 1)

			prng := rand.New(rand.NewSource(0))
			for i := range block {
				block[i] = int32(prng.Uint32()) & bitMask
			}

			size := (blockSize * bitWidth) / 8
			buf := make([]byte, size+bitpack.PaddingInt32)
			bitpack.Pack(buf, block[:], bitWidth)

			src := buf[:size]
			dst := make([]int32, blockSize+1)

			for n := 0; n <= blockSize; n++ {
				for i := range dst {
					dst[i] = -1
				}

				bitpack.Unpack(dst[:n], src, bitWidth)

				if !slices.Equal(block[:n], dst[:n]) {
					t.Fatalf("values mismatch for length=%d\nwant: %v\ngot:  %v", n, block[:n], dst[:n])
				}
				if dst[n] != -1 {
					t.Fatalf("wrote beyond destination for length=%d", n)
				}
			}
		})
	}
}

func TestUnpackInt64(t *testing.T) {
	for bitWidth := uint(1); bitWidth <= 64; bitWidth++ {
		t.Run(fmt.Sprintf("bitWidth=%d", bitWidth), func(t *testing.T) {
			block := [blockSize]int64{}
			bitMask := int64(uint64(1)<<bitWidth - 1)

			prng := rand.New(rand.NewSource(0))
			for i := range block {
				block[i] = int64(prng.Uint64()) & bitMask
			}

			size := (blockSize * bitWidth) / 8
			buf := make([]byte, size+bitpack.PaddingInt64)
			bitpack.Pack(buf, block[:], bitWidth)

			src := buf[:size]
			dst := make([]int64, blockSize+1)

			for n := 0; n <= blockSize; n++ {
				for i := range dst {
					dst[i] = -1
				}

				bitpack.Unpack(dst[:n], src, bitWidth)

				if !slices.Equal(block[:n], dst[:n]) {
					t.Fatalf("values mismatch for length=%d\nwant: %v\ngot:  %v", n, block[:n], dst[:n])
				}
				if dst[n] != -1 {
					t.Fatalf("wrote beyond destination for length=%d", n)
				}
			}
		})
	}
}

func TestUnpackZeroWidth(t *testing.T) {
	for _, size := range []int{0, 7, 8, 9, blockSize} {
		t.Run(fmt.Sprintf("int32/size=%d", size), func(t *testing.T) {
			dst := make([]int32, size)
			for i := range dst {
				dst[i] = -1
			}
			bitpack.Unpack(dst, make([]byte, bitpack.PaddingInt32), 0)
			if !slices.Equal(dst, make([]int32, size)) {
				t.Fatalf("expected zeros, got %v", dst)
			}
		})

		t.Run(fmt.Sprintf("int64/size=%d", size), func(t *testing.T) {
			dst := make([]int64, size)
			for i := range dst {
				dst[i] = -1
			}
			bitpack.Unpack(dst, make([]byte, bitpack.PaddingInt64), 0)
			if !slices.Equal(dst, make([]int64, size)) {
				t.Fatalf("expected zeros, got %v", dst)
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
		bitMask := int64(uint64(1)<<bitWidth - 1)
		for i := range src {
			src[i] = gen.Int63() & bitMask
		}

		packed := make([]byte, size*8+bitpack.PaddingInt64)
		bitpack.Pack(packed, src[:], bitWidth)

		unpacked := make([]int64, size)
		bitpack.Unpack(unpacked[:], packed[:], bitWidth)

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
		bitMask := int32(uint32(1)<<bitWidth - 1)
		for i := range src {
			src[i] = gen.Int31() & bitMask
		}

		packed := make([]byte, size*4+bitpack.PaddingInt32)
		bitpack.Pack(packed, src[:], bitWidth)

		unpacked := make([]int32, size)
		bitpack.Unpack(unpacked[:], packed[:], bitWidth)

		if !slices.Equal(unpacked, src) {
			t.Fatalf("Roundtrip failed: got %v, want %v", unpacked, src)
		}
	})
}

func BenchmarkUnpackInt32(b *testing.B) {
	for bitWidth := uint(1); bitWidth <= 32; bitWidth++ {
		block := [blockSize]int32{}
		buf := [4*blockSize + bitpack.PaddingInt32]byte{}
		bitpack.Pack(buf[:], block[:], bitWidth)

		b.Run(fmt.Sprintf("bitWidth=%d", bitWidth), func(b *testing.B) {
			dst := block[:]
			src := buf[:]

			for i := 0; i < b.N; i++ {
				bitpack.Unpack(dst, src, bitWidth)
			}

			b.SetBytes(4 * blockSize)
		})
	}
}

func BenchmarkUnpackInt64(b *testing.B) {
	for bitWidth := uint(1); bitWidth <= 64; bitWidth++ {
		block := [blockSize]int64{}
		buf := [8*blockSize + bitpack.PaddingInt64]byte{}
		bitpack.Pack(buf[:], block[:], bitWidth)

		b.Run(fmt.Sprintf("bitWidth=%d", bitWidth), func(b *testing.B) {
			dst := block[:]
			src := buf[:]

			for i := 0; i < b.N; i++ {
				bitpack.Unpack(dst, src, bitWidth)
			}

			b.SetBytes(8 * blockSize)
		})
	}
}
