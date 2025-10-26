package examples

import (
	"fmt"
	"github.com/parquet-go/bitpack"
)

func ExamplePack_int32() {
	// Pack int32 values
	values := []int32{1, 2, 3, 4, 5}
	bitWidth := uint(3)

	packedSize := bitpack.ByteCount(uint(len(values)) * bitWidth)
	// Allocate buffer with extra space for pack algorithm
	// (needs 3 bytes per value + padding for safe bit operations)
	dst := make([]byte, packedSize+bitpack.PaddingInt32)

	// Pack the values
	bitpack.Pack(dst, values, bitWidth)

	// Calculate actual packed size
	fmt.Printf("Packed %d values into %d bytes\n", len(values), packedSize)
	// Output: Packed 5 values into 2 bytes
}

func ExampleUnpack_int32() {
	// First, pack some values
	values := []int32{10, 20, 30, 40, 50}
	bitWidth := uint(6)

	packedSize := bitpack.ByteCount(uint(len(values)) * bitWidth)
	// Allocate buffer for packing
	packed := make([]byte, packedSize+bitpack.PaddingInt32)
	bitpack.Pack(packed, values, bitWidth)

	// Now unpack them
	dst := make([]int32, len(values))
	bitpack.Unpack(dst, packed, bitWidth)

	fmt.Printf("Unpacked values: %v\n", dst)
	// Output: Unpacked values: [10 20 30 40 50]
}

func ExamplePack_int64() {
	// Pack int64 values
	values := []int64{100, 200, 300, 400, 500}
	bitWidth := uint(9)

	packedSize := bitpack.ByteCount(uint(len(values)) * bitWidth)
	// Allocate buffer with extra space for pack algorithm
	// (needs packed size + padding for safe bit operations)
	dst := make([]byte, packedSize+bitpack.PaddingInt64)

	// Pack the values
	bitpack.Pack(dst, values, bitWidth)

	// Calculate actual packed size
	fmt.Printf("Packed %d values into %d bytes\n", len(values), packedSize)
	// Output: Packed 5 values into 6 bytes
}

func ExampleUnpack_int64() {
	// First, pack some values
	values := []int64{1000, 2000, 3000, 4000, 5000}
	bitWidth := uint(13)

	packedSize := bitpack.ByteCount(uint(len(values)) * bitWidth)
	// Allocate buffer for packing
	packed := make([]byte, packedSize+bitpack.PaddingInt64)
	bitpack.Pack(packed, values, bitWidth)

	// Now unpack them
	dst := make([]int64, len(values))
	bitpack.Unpack(dst, packed, bitWidth)

	fmt.Printf("Unpacked values: %v\n", dst)
	// Output: Unpacked values: [1000 2000 3000 4000 5000]
}
