//go:build !purego

package bitpack

import (
	"fmt"
	"math/rand"
	"slices"
	"testing"
)

func TestUnpackInt32GeneratedAMD64MatchesDefault(t *testing.T) {
	values := make([]int32, 128)
	prng := rand.New(rand.NewSource(3))
	for i := range values {
		values[i] = int32(prng.Uint32())
	}

	for bitWidth := uint(1); bitWidth < 32; bitWidth++ {
		t.Run(fmt.Sprintf("bitWidth=%d", bitWidth), func(t *testing.T) {
			src := make([]byte, ByteCount(uint(len(values))*bitWidth)+PaddingInt32)
			packInt32Default(src, values, bitWidth)
			want := make([]int32, len(values))
			unpackInt32Default(want, src, bitWidth)
			got := make([]int32, len(values)+8)
			for i := len(values); i < len(got); i++ {
				got[i] = int32(i*31 + 7)
			}
			guard := slices.Clone(got[len(values):])

			unpackInt32GeneratedAMD64(got[:len(values)], src, bitWidth)
			if !slices.Equal(got[:len(values)], want) {
				t.Fatalf("unpacked values differ\nwant: %x\ngot:  %x", want, got[:len(values)])
			}
			if !slices.Equal(got[len(values):], guard) {
				t.Fatal("wrote beyond destination")
			}
		})
	}
}

func TestUnpackInt64GeneratedAMD64MatchesDefault(t *testing.T) {
	values := make([]int64, 128)
	prng := rand.New(rand.NewSource(4))
	for i := range values {
		values[i] = int64(prng.Uint64())
	}

	for bitWidth := uint(1); bitWidth < 64; bitWidth++ {
		t.Run(fmt.Sprintf("bitWidth=%d", bitWidth), func(t *testing.T) {
			src := make([]byte, ByteCount(uint(len(values))*bitWidth)+PaddingInt64)
			packInt64Default(src, values, bitWidth)
			want := make([]int64, len(values))
			unpackInt64Default(want, src, bitWidth)
			got := make([]int64, len(values)+8)
			for i := len(values); i < len(got); i++ {
				got[i] = int64(i*31 + 7)
			}
			guard := slices.Clone(got[len(values):])

			unpackInt64GeneratedAMD64(got[:len(values)], src, bitWidth)
			if !slices.Equal(got[:len(values)], want) {
				t.Fatalf("unpacked values differ\nwant: %x\ngot:  %x", want, got[:len(values)])
			}
			if !slices.Equal(got[len(values):], guard) {
				t.Fatal("wrote beyond destination")
			}
		})
	}
}
