//go:build !purego

package bitpack

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"
)

func TestPackInt32AMD64MatchesScalar(t *testing.T) {
	values := make([]int32, 129)
	prng := rand.New(rand.NewSource(1))
	for i := range values {
		values[i] = int32(prng.Uint32())
	}

	for bitWidth := uint(1); bitWidth <= 32; bitWidth++ {
		t.Run(fmt.Sprintf("bitWidth=%d", bitWidth), func(t *testing.T) {
			for n := 0; n <= len(values); n++ {
				size := ByteCount(uint(n) * bitWidth)
				want := make([]byte, size)
				packInt32Default(want, values[:n], bitWidth)

				backing := make([]byte, size+32)
				for i := size; i < len(backing); i++ {
					backing[i] = byte(i*31 + 7)
				}
				guard := bytes.Clone(backing[size:])
				packInt32(backing[:size], values[:n], bitWidth)

				if !bytes.Equal(backing[:size], want) {
					t.Fatalf("packed bytes differ for length=%d\nwant: %x\ngot:  %x", n, want, backing[:size])
				}
				if !bytes.Equal(backing[size:], guard) {
					t.Fatalf("wrote beyond destination for length=%d", n)
				}
			}
		})
	}
}

func TestPackInt64AMD64MatchesScalar(t *testing.T) {
	values := make([]int64, 129)
	prng := rand.New(rand.NewSource(2))
	for i := range values {
		values[i] = int64(prng.Uint64())
	}

	for bitWidth := uint(1); bitWidth <= 64; bitWidth++ {
		t.Run(fmt.Sprintf("bitWidth=%d", bitWidth), func(t *testing.T) {
			for n := 0; n <= len(values); n++ {
				size := ByteCount(uint(n) * bitWidth)
				want := make([]byte, size)
				packInt64Default(want, values[:n], bitWidth)

				backing := make([]byte, size+32)
				for i := size; i < len(backing); i++ {
					backing[i] = byte(i*31 + 7)
				}
				guard := bytes.Clone(backing[size:])
				packInt64(backing[:size], values[:n], bitWidth)

				if !bytes.Equal(backing[:size], want) {
					t.Fatalf("packed bytes differ for length=%d\nwant: %x\ngot:  %x", n, want, backing[:size])
				}
				if !bytes.Equal(backing[size:], guard) {
					t.Fatalf("wrote beyond destination for length=%d", n)
				}
			}
		})
	}
}
