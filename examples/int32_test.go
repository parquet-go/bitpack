package examples

import (
	"fmt"
	"github.com/parquet-go/bitpack"
)

func ExamplePackInt32() {
	// Pack int32 values
	values := []int32{1, 2, 3, 4, 5}
	bitWidth := uint(3)

	packedSize := bitpack.ByteCount(uint(len(values)) * bitWidth)
	// Allocate buffer with extra space for pack algorithm
	// (needs 3 bytes per value + padding for safe bit operations)
	dst := make([]byte, packedSize+bitpack.PaddingInt32)

	// Pack the values
	bitpack.PackInt32(dst, values, bitWidth)

	// Calculate actual packed size
	fmt.Printf("Packed %d values into %d bytes\n", len(values), packedSize)
	// Output: Packed 5 values into 2 bytes
}

func ExampleUnpackInt32() {
	// First, pack some values
	values := []int32{10, 20, 30, 40, 50}
	bitWidth := uint(6)

	packedSize := bitpack.ByteCount(uint(len(values)) * bitWidth)
	// Allocate buffer for packing
	packed := make([]byte, packedSize+bitpack.PaddingInt32)
	bitpack.PackInt32(packed, values, bitWidth)

	// Now unpack them
	dst := make([]int32, len(values))
	bitpack.UnpackInt32(dst, packed, bitWidth)

	fmt.Printf("Unpacked values: %v\n", dst)
	// Output: Unpacked values: [10 20 30 40 50]
}
