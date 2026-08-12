//go:build goexperiment.simd

package bitpack

import (
	"encoding/binary"
	"simd/archsimd"
	"unsafe"

	"github.com/parquet-go/bitpack/unsafecast"
)

// This file provides implementations of the int64 bit unpacking algorithms
// based on the simd/archsimd package, replacing the hand-written assembly of
// unpack_int64_amd64.s and unpack_int64_{1,2,4,8}bit_amd64.s when
// GOEXPERIMENT=simd is set.
//
// Bit widths 1 to 32 use the vectorized algorithm of the assembly: a
// cross-lane 32 bit permutation (VPERMD) places the two words containing
// each value in a 64 bit lane, giving a window that always contains the
// full value, and a per-lane logical right shift + mask extracts it. Unlike
// the assembly, which runs scalar specializations for bit widths 1-8, 16
// and 32, the vector kernel is used for the whole 1-32 range: it is faster
// than those scalar loops (the "AVX2" kernels of the specialized .s files
// perform no vector work despite their names).
//
// Bit widths 33 to 63 fall back to the scalar loop, and 64 is a copy.

// unpackInt64Permute holds the permutation and shift vectors used to unpack
// 8 values of a given bit width from 32 bytes of input. The value at index
// i starts at bit i*bitWidth: the permutation loads words i*bitWidth/32 and
// i*bitWidth/32+1 into a 64 bit lane, aligned by shifting right by
// i*bitWidth%32.
//
// This is the formula gen_int64_masks.go uses to generate the assembly
// tables in masks_int64_amd64.s.
type unpackInt64Permute struct {
	perm0  [8]uint32
	perm1  [8]uint32
	shift0 [4]uint64
	shift1 [4]uint64
}

var unpackInt64Permutes [33]unpackInt64Permute

func init() {
	for bitWidth := uint(1); bitWidth <= 32; bitWidth++ {
		m := &unpackInt64Permutes[bitWidth]
		for lane := uint(0); lane < 8; lane++ {
			// At bitWidth 32, lane 7 computes word+1 == 8; VPERMD only uses
			// the low 3 bits of each index so it wraps to word 0, and the
			// bits it contributes are cleared by the 32 bit mask.
			word := (lane * bitWidth) / 32
			shift := (lane * bitWidth) % 32
			if lane < 4 {
				m.perm0[2*lane+0] = uint32(word)
				m.perm0[2*lane+1] = uint32(word + 1)
				m.shift0[lane] = uint64(shift)
			} else {
				m.perm1[2*(lane-4)+0] = uint32(word)
				m.perm1[2*(lane-4)+1] = uint32(word + 1)
				m.shift1[lane-4] = uint64(shift)
			}
		}
	}
}

func unpackInt64(dst []int64, src []byte, bitWidth uint) {
	if bitWidth == 64 {
		copy(dst, unsafecast.Slice[int64](src))
		return
	}
	// The Unpack contract guarantees PaddingInt64 bytes of capacity after the
	// packed values; extend the length over them so that the full-width vector
	// loads of the last iteration and the word reads of the scalar tail stay
	// in bounds.
	src = src[:ByteCount(bitWidth*uint(len(dst)))+PaddingInt64]
	switch {
	case archsimd.X86.AVX2() && 1 <= bitWidth && bitWidth <= 32:
		unpackInt64x1to32bits(dst, src, bitWidth)
	case bitWidth <= 8:
		unpackInt64x1to8bits(dst, src, bitWidth)
	default:
		unpackInt64Default(dst, src, bitWidth)
	}
}

// unpackInt64x1to8bits unpacks 8 values per iteration from a single 64 bit
// word with scalar shifts and masks; it is only used when AVX2 is not
// available.
func unpackInt64x1to8bits(dst []int64, src []byte, bitWidth uint) {
	bitMask := uint64(1)<<bitWidth - 1
	n := (len(dst) / 8) * 8
	i, j := 0, 0
	for i < n {
		d := dst[i : i+8 : i+8]
		w := binary.LittleEndian.Uint64(src[j:])
		d[0] = int64(w & bitMask)
		d[1] = int64((w >> (1 * bitWidth)) & bitMask)
		d[2] = int64((w >> (2 * bitWidth)) & bitMask)
		d[3] = int64((w >> (3 * bitWidth)) & bitMask)
		d[4] = int64((w >> (4 * bitWidth)) & bitMask)
		d[5] = int64((w >> (5 * bitWidth)) & bitMask)
		d[6] = int64((w >> (6 * bitWidth)) & bitMask)
		d[7] = int64((w >> (7 * bitWidth)) & bitMask)
		i += 8
		j += int(bitWidth)
	}
	if i < len(dst) {
		unpackInt64Default(dst[i:], src[j:], bitWidth)
	}
}

// unpackInt64x1to32bits unpacks 8 values per iteration from a 32 byte load
// using a cross-lane word permutation and per-lane shifts.
func unpackInt64x1to32bits(dst []int64, src []byte, bitWidth uint) {
	n := (len(dst) / 8) * 8
	if n > 0 {
		m := &unpackInt64Permutes[bitWidth]
		perm0 := archsimd.LoadUint32x8(&m.perm0)
		perm1 := archsimd.LoadUint32x8(&m.perm1)
		shift0 := archsimd.LoadUint64x4(&m.shift0)
		shift1 := archsimd.LoadUint64x4(&m.shift1)
		bitMask := archsimd.BroadcastUint64x4(uint64(1)<<bitWidth - 1)

		// The loop walks raw pointers: slice expressions on src and dst keep
		// enough values live across the vector ops that the loop spills to
		// the stack and re-checks bounds on every iteration. Every load is in
		// bounds of the padded src length established by unpackInt64.
		in := unsafe.Pointer(unsafe.SliceData(src))
		op := unsafe.Pointer(unsafe.SliceData(dst))
		for range n / 8 {
			w := archsimd.LoadUint8x32((*[32]uint8)(in)).AsUint32x8()
			w.Permute(perm0).AsUint64x4().ShiftRight(shift0).And(bitMask).Store((*[4]uint64)(op))
			w.Permute(perm1).AsUint64x4().ShiftRight(shift1).And(bitMask).Store((*[4]uint64)(unsafe.Add(op, 32)))
			in = unsafe.Add(in, bitWidth)
			op = unsafe.Add(op, 64)
		}
		archsimd.ClearAVXUpperBits()
	}
	if n < len(dst) {
		unpackInt64Default(dst[n:], src[(uint(n)/8)*bitWidth:], bitWidth)
	}
}

func unpackInt64Default(dst []int64, src []byte, bitWidth uint) {
	words := unsafecast.Slice[uint64](src)
	bitMask := uint64(1)<<bitWidth - 1
	bitOffset := uint(0)

	for n := range dst {
		i := bitOffset / 64
		j := bitOffset % 64
		d := (words[i] >> j) & bitMask
		if j+bitWidth > 64 {
			k := 64 - j
			d |= (words[i+1] & (bitMask >> k)) << k
		}
		dst[n] = int64(d)
		bitOffset += bitWidth
	}
}
