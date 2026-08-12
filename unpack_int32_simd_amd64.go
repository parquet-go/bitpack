//go:build goexperiment.simd

package bitpack

import (
	"simd/archsimd"
	"unsafe"

	"github.com/parquet-go/bitpack/unsafecast"
)

// This file provides implementations of the int32 bit unpacking algorithms
// based on the simd/archsimd package, replacing the hand-written assembly of
// unpack_int32_amd64.s when GOEXPERIMENT=simd is set.
//
// The algorithms are adaptations of the "horizontal" method described in
// "Decoding billions of integers per second through vectorization" by
// D. Lemire & L. Boytsov (see unpack_int32_amd64.s for the full history):
// each iteration unpacks 8 values by dispatching the packed bytes into the
// four byte positions of each 32 bit lane (VPSHUFB), aligning the values on
// the lane boundaries with per-lane logical shifts (VPSRLVD/VPSLLVD), and
// masking off the extra high bits.
//
// The port differs from the assembly in how lanes 4-7 read their input: the
// assembly loads two consecutive 16 byte registers and merges two shuffles
// (bit widths 17 and up straddle the boundary between the two), while here
// the second 16 byte load is offset by the byte position of value 4, so the
// bytes of lanes 4-7 always fit a single shuffle of that register. This
// drops one shuffle+or from the 17-26 bit path and two from the 27-31 bit
// path, at the cost of the loads overlapping for bit widths below 32.
//
// The assembly loads its shuffle masks and shift vectors from tables
// declared in masks_int32_amd64.s; the equivalent tables here are computed
// once at package initialization from the layout formulas, indexed by bit
// width.

// unpackInt32Shuffle holds the shuffle masks and shift vectors used to
// unpack 8 values of a given bit width.
//
// The value at index i starts at bit i*bitWidth: in byte i*bitWidth/8 of
// the input, offset by shift = i*bitWidth%8 bits. Lanes 0-3 index their
// bytes in src[j:j+16]; lanes 4-7 in src[j+g:j+g+16] where g is the byte
// position of value 4 (g = 4*bitWidth/8), which keeps every index below 16
// for all bit widths up to 31.
//
// For bit widths 27 to 31, a value plus its leading shift may span 5 bytes,
// one more than a 32 bit lane can hold. The 4 low bytes are shifted right
// into place, and the 5th byte is dispatched to the top byte of a second
// lane which is shifted left by 8-shift, so its bits land at position
// 32-shift.
type unpackInt32Shuffle struct {
	lo0 [16]int8 // low word bytes of lanes 0-3, indexed in src[j:]
	lo1 [16]int8 // low word bytes of lanes 4-7, indexed in src[j+g:]
	hi0 [16]int8 // 5th byte of lanes 0-3, indexed in src[j:]
	hi1 [16]int8 // 5th byte of lanes 4-7, indexed in src[j+g:]
	sr0 [4]uint32
	sr1 [4]uint32
	sl0 [4]uint32
	sl1 [4]uint32
}

var unpackInt32Shuffles [32]unpackInt32Shuffle

func init() {
	for bitWidth := uint(1); bitWidth <= 31; bitWidth++ {
		m := &unpackInt32Shuffles[bitWidth]
		for i := range 16 {
			m.lo0[i], m.lo1[i], m.hi0[i], m.hi1[i] = -128, -128, -128, -128
		}
		// For bit widths up to 16 all 8 values fit in the first 16 bytes, so
		// lanes 4-7 index src[j:] directly and the kernel performs a single
		// load; wider values straddle the boundary, so lanes 4-7 index a
		// second load at src[j+g:].
		g := uint(0)
		if bitWidth > 16 {
			g = (4 * bitWidth) / 8
		}
		for lane := uint(0); lane < 8; lane++ {
			bitOffset := lane * bitWidth
			firstByte := bitOffset / 8
			shift := bitOffset % 8
			byteCount := (shift + bitWidth + 7) / 8
			lo, hi, base := &m.lo0, &m.hi0, uint(0)
			if lane >= 4 {
				lo, hi, base = &m.lo1, &m.hi1, g
			}
			for k := uint(0); k < min(byteCount, 4); k++ {
				lo[(lane%4)*4+k] = int8(firstByte + k - base)
			}
			if byteCount == 5 {
				hi[(lane%4)*4+3] = int8(firstByte + 4 - base)
			}
			if lane < 4 {
				m.sr0[lane] = uint32(shift)
				m.sl0[lane] = uint32(8 - shift)
			} else {
				m.sr1[lane-4] = uint32(shift)
				m.sl1[lane-4] = uint32(8 - shift)
			}
		}
	}
}

func unpackInt32(dst []int32, src []byte, bitWidth uint) {
	if bitWidth == 32 {
		copy(dst, unsafecast.Slice[int32](src))
		return
	}
	// The Unpack contract guarantees PaddingInt32 bytes of capacity after the
	// packed values; extend the length over them so that the full-width vector
	// loads of the last iteration and the word reads of the scalar tail stay
	// in bounds.
	src = src[:ByteCount(bitWidth*uint(len(dst)))+PaddingInt32]
	hasAVX2 := archsimd.X86.AVX2()
	switch {
	case hasAVX2 && 1 <= bitWidth && bitWidth <= 16:
		unpackInt32x1to16bits(dst, src, bitWidth)
	case hasAVX2 && bitWidth <= 26:
		unpackInt32x17to26bits(dst, src, bitWidth)
	case hasAVX2 && bitWidth <= 31:
		unpackInt32x27to31bits(dst, src, bitWidth)
	default:
		unpackInt32Default(dst, src, bitWidth)
	}
}

// unpackInt32x1to16bits unpacks values of bit widths 1 to 16, for which the
// 8 values of an iteration fit in a single 16 byte load.
func unpackInt32x1to16bits(dst []int32, src []byte, bitWidth uint) {
	n := (len(dst) / 8) * 8
	if n > 0 {
		m := &unpackInt32Shuffles[bitWidth]
		lo0 := archsimd.LoadInt8x16(&m.lo0)
		lo1 := archsimd.LoadInt8x16(&m.lo1)
		sr0 := archsimd.LoadUint32x4(&m.sr0)
		sr1 := archsimd.LoadUint32x4(&m.sr1)
		bitMask := archsimd.BroadcastUint32x4(uint32(1)<<bitWidth - 1)

		// See unpackInt32x17to26bits for why the loop walks raw pointers.
		in := unsafe.Pointer(unsafe.SliceData(src))
		op := unsafe.Pointer(unsafe.SliceData(dst))
		for range n / 8 {
			w0 := archsimd.LoadUint8x16((*[16]uint8)(in))
			w0.PermuteOrZero(lo0).AsUint32x4().ShiftRight(sr0).And(bitMask).Store((*[4]uint32)(op))
			w0.PermuteOrZero(lo1).AsUint32x4().ShiftRight(sr1).And(bitMask).Store((*[4]uint32)(unsafe.Add(op, 16)))
			in = unsafe.Add(in, bitWidth)
			op = unsafe.Add(op, 32)
		}
	}
	if n < len(dst) {
		unpackInt32Default(dst[n:], src[(uint(n)/8)*bitWidth:], bitWidth)
	}
}

// unpackInt32x17to26bits unpacks values of bit widths 17 to 26, for which a
// value plus its leading shift spans at most 4 bytes: one shuffle + shift +
// mask per group of 4 lanes, with lanes 4-7 reading from a second load
// offset by the byte position of value 4.
func unpackInt32x17to26bits(dst []int32, src []byte, bitWidth uint) {
	n := (len(dst) / 8) * 8
	if n > 0 {
		m := &unpackInt32Shuffles[bitWidth]
		lo0 := archsimd.LoadInt8x16(&m.lo0)
		lo1 := archsimd.LoadInt8x16(&m.lo1)
		sr0 := archsimd.LoadUint32x4(&m.sr0)
		sr1 := archsimd.LoadUint32x4(&m.sr1)
		bitMask := archsimd.BroadcastUint32x4(uint32(1)<<bitWidth - 1)
		g := (4 * bitWidth) / 8

		// The loop walks raw pointers: slice expressions on src and dst keep
		// enough values live across the vector ops that the loop spills to
		// the stack and re-checks bounds on every iteration. Every load is in
		// bounds of the padded src length established by unpackInt32.
		in := unsafe.Pointer(unsafe.SliceData(src))
		op := unsafe.Pointer(unsafe.SliceData(dst))
		for range n / 8 {
			w0 := archsimd.LoadUint8x16((*[16]uint8)(in))
			w1 := archsimd.LoadUint8x16((*[16]uint8)(unsafe.Add(in, g)))
			w0.PermuteOrZero(lo0).AsUint32x4().ShiftRight(sr0).And(bitMask).Store((*[4]uint32)(op))
			w1.PermuteOrZero(lo1).AsUint32x4().ShiftRight(sr1).And(bitMask).Store((*[4]uint32)(unsafe.Add(op, 16)))
			in = unsafe.Add(in, bitWidth)
			op = unsafe.Add(op, 32)
		}
	}
	if n < len(dst) {
		unpackInt32Default(dst[n:], src[(uint(n)/8)*bitWidth:], bitWidth)
	}
}

// unpackInt32x27to31bits unpacks values of bit widths 27 to 31, which may
// span 5 bytes: the low 4 bytes are shifted right into position, and the
// 5th byte is shifted left by 8-shift to contribute the bits above
// 32-shift.
func unpackInt32x27to31bits(dst []int32, src []byte, bitWidth uint) {
	n := (len(dst) / 8) * 8
	if n > 0 {
		m := &unpackInt32Shuffles[bitWidth]
		lo0 := archsimd.LoadInt8x16(&m.lo0)
		lo1 := archsimd.LoadInt8x16(&m.lo1)
		hi0 := archsimd.LoadInt8x16(&m.hi0)
		hi1 := archsimd.LoadInt8x16(&m.hi1)
		sr0 := archsimd.LoadUint32x4(&m.sr0)
		sr1 := archsimd.LoadUint32x4(&m.sr1)
		sl0 := archsimd.LoadUint32x4(&m.sl0)
		sl1 := archsimd.LoadUint32x4(&m.sl1)
		bitMask := archsimd.BroadcastUint32x4(uint32(1)<<bitWidth - 1)
		g := (4 * bitWidth) / 8

		// See unpackInt32x17to26bits for why the loop walks raw pointers.
		in := unsafe.Pointer(unsafe.SliceData(src))
		op := unsafe.Pointer(unsafe.SliceData(dst))
		for range n / 8 {
			w0 := archsimd.LoadUint8x16((*[16]uint8)(in))
			w1 := archsimd.LoadUint8x16((*[16]uint8)(unsafe.Add(in, g)))
			w0.PermuteOrZero(lo0).AsUint32x4().ShiftRight(sr0).
				Or(w0.PermuteOrZero(hi0).AsUint32x4().ShiftLeft(sl0)).
				And(bitMask).Store((*[4]uint32)(op))
			w1.PermuteOrZero(lo1).AsUint32x4().ShiftRight(sr1).
				Or(w1.PermuteOrZero(hi1).AsUint32x4().ShiftLeft(sl1)).
				And(bitMask).Store((*[4]uint32)(unsafe.Add(op, 16)))
			in = unsafe.Add(in, bitWidth)
			op = unsafe.Add(op, 32)
		}
	}
	if n < len(dst) {
		unpackInt32Default(dst[n:], src[(uint(n)/8)*bitWidth:], bitWidth)
	}
}

func unpackInt32Default(dst []int32, src []byte, bitWidth uint) {
	bits := unsafecastBytesToUint32(src)
	bitMask := uint32(1<<bitWidth) - 1
	bitOffset := uint(0)

	for n := range dst {
		i := bitOffset / 32
		j := bitOffset % 32
		d := (bits[i] & (bitMask << j)) >> j
		if j+bitWidth > 32 {
			k := 32 - j
			d |= (bits[i+1] & (bitMask >> k)) << k
		}
		dst[n] = int32(d)
		bitOffset += bitWidth
	}
}
