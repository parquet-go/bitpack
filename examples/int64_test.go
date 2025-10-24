package examples

import (
	"fmt"
	"github.com/parquet-go/bitpack"
)

func ExamplePackInt64() {
	// Pack int64 values
	values := []int64{100, 200, 300, 400, 500}
	bitWidth := uint(9)

	packedSize := bitpack.ByteCount(uint(len(values)) * bitWidth)
	// Allocate buffer with extra space for pack algorithm
	// (needs packed size + padding for safe bit operations)
	dst := make([]byte, packedSize+bitpack.PaddingInt64)

	// Pack the values
	bitpack.PackInt64(dst, values, bitWidth)

	// Calculate actual packed size
	fmt.Printf("Packed %d values into %d bytes\n", len(values), packedSize)
	// Output: Packed 5 values into 6 bytes
}

func ExampleUnpackInt64() {
	// First, pack some values
	values := []int64{1000, 2000, 3000, 4000, 5000}
	bitWidth := uint(13)

	packedSize := bitpack.ByteCount(uint(len(values)) * bitWidth)
	// Allocate buffer for packing
	packed := make([]byte, packedSize+bitpack.PaddingInt64)
	bitpack.PackInt64(packed, values, bitWidth)

	// Now unpack them
	dst := make([]int64, len(values))
	bitpack.UnpackInt64(dst, packed, bitWidth)

	fmt.Printf("Unpacked values: %v\n", dst)
	// Output: Unpacked values: [1000 2000 3000 4000 5000]
}
