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

// TestUnpackInt32MatchesBitReference compares Unpack with an independent
// little-endian bit reader. Padding bytes are 0xFF so a kernel that pulls in
// bits past the packed values fails the comparison. The non-simd and simd
// builds both run this test, so a passing pair of runs means those
// implementations agree with the same reference.
func TestUnpackInt32MatchesBitReference(t *testing.T) {
	lengths := []int{0, 1, 7, 8, 9, 15, 16, 17, 31, 32, 33, 63, 64, 65, 127, 128, 200}
	for bitWidth := uint(0); bitWidth <= 32; bitWidth++ {
		t.Run(fmt.Sprintf("bitWidth=%d", bitWidth), func(t *testing.T) {
			prng := rand.New(rand.NewSource(int64(bitWidth) + 1))
			mask := ^uint32(0)
			if bitWidth < 32 {
				mask = uint32(1)<<bitWidth - 1
			}
			for _, n := range lengths {
				values := make([]int32, n)
				raw := make([]uint64, n)
				for i := range values {
					values[i] = int32(prng.Uint32() & mask)
					raw[i] = uint64(uint32(values[i]))
				}
				size := bitpack.ByteCount(bitWidth * uint(n))
				buf := make([]byte, size+bitpack.PaddingInt32)
				writeBits(buf[:size], raw, bitWidth)
				for i := size; i < len(buf); i++ {
					buf[i] = 0xFF
				}
				got := make([]int32, n)
				// Pass the padded buffer. Unpack's contract is that those
				// bytes may be read; they must not affect the values.
				bitpack.Unpack(got, buf, bitWidth)
				if !slices.Equal(got, values) {
					t.Fatalf("length=%d\nwant: %v\ngot:  %v", n, values, got)
				}
			}
		})
	}
}

// TestUnpackInt64MatchesBitReference is the int64 counterpart of
// TestUnpackInt32MatchesBitReference.
func TestUnpackInt64MatchesBitReference(t *testing.T) {
	lengths := []int{0, 1, 7, 8, 9, 15, 16, 17, 31, 32, 33, 63, 64, 65, 127, 128, 200}
	for bitWidth := uint(0); bitWidth <= 64; bitWidth++ {
		t.Run(fmt.Sprintf("bitWidth=%d", bitWidth), func(t *testing.T) {
			prng := rand.New(rand.NewSource(int64(bitWidth) + 100))
			mask := ^uint64(0)
			if bitWidth < 64 {
				mask = uint64(1)<<bitWidth - 1
			}
			for _, n := range lengths {
				values := make([]int64, n)
				raw := make([]uint64, n)
				for i := range values {
					values[i] = int64(prng.Uint64() & mask)
					raw[i] = uint64(values[i])
				}
				size := bitpack.ByteCount(bitWidth * uint(n))
				buf := make([]byte, size+bitpack.PaddingInt64)
				writeBits(buf[:size], raw, bitWidth)
				for i := size; i < len(buf); i++ {
					buf[i] = 0xFF
				}
				got := make([]int64, n)
				// Pass the padded buffer. Unpack's contract is that those
				// bytes may be read; they must not affect the values.
				bitpack.Unpack(got, buf, bitWidth)
				if !slices.Equal(got, values) {
					t.Fatalf("length=%d\nwant: %v\ngot:  %v", n, values, got)
				}
			}
		})
	}
}

// writeBits stores each value in little-endian bit order, which is the
// layout Pack and Unpack use.
func writeBits(dst []byte, values []uint64, bitWidth uint) {
	for i, v := range values {
		bitOffset := uint(i) * bitWidth
		for b := uint(0); b < bitWidth; b++ {
			if v&(uint64(1)<<b) == 0 {
				continue
			}
			pos := bitOffset + b
			dst[pos/8] |= 1 << (pos % 8)
		}
	}
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
