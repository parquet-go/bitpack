package bitpack

// PackInt32 packs values from src to dst, each value is packed into the given
// bit width regardless of how many bits are needed to represent it.
//
// The function panics if dst is too short to hold the bit packed values.
func PackInt32(dst []byte, src []int32, bitWidth uint) {
	assertPack(dst, len(src), bitWidth)
	packInt32(dst, src, bitWidth)
}

// PackInt64 packs values from src to dst, each value is packed into the given
// bit width regardless of how many bits are needed to represent it.
//
// The function panics if dst is too short to hold the bit packed values.
func PackInt64(dst []byte, src []int64, bitWidth uint) {
	assertPack(dst, len(src), bitWidth)
	packInt64(dst, src, bitWidth)
}

func assertPack(dst []byte, count int, bitWidth uint) {
	_ = dst[:ByteCount(bitWidth*uint(count))]
}
