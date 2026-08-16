package unsafecast_test

import (
	"testing"
	"unsafe"

	"github.com/parquet-go/bitpack/unsafecast"
	"golang.org/x/sys/cpu"
)

func TestUnsafeCastSlice(t *testing.T) {
	// Note: this test is currently disabled on Big-Endian architectures because
	// it assumes a Little-Endian memory layout.
	if cpu.IsBigEndian {
		t.Skip("skipping test on big-endian architecture")
	}

	a := make([]uint32, 4, 13)
	a[0] = 1
	a[1] = 0
	a[2] = 2
	a[3] = 0

	b := unsafecast.Slice[int64](a)
	if len(b) != 2 { // (4 * sizeof(uint32)) / sizeof(int64)
		t.Fatalf("length mismatch: want=2 got=%d", len(b))
	}
	if cap(b) != 6 { // (13 * sizeof(uint32)) / sizeof(int64)
		t.Fatalf("capacity mismatch: want=6 got=%d", cap(b))
	}
	if b[0] != 1 {
		t.Errorf("wrong value at index 0: want=1 got=%d", b[0])
	}
	if b[1] != 2 {
		t.Errorf("wrong value at index 1: want=2 got=%d", b[1])
	}

	c := unsafecast.Slice[uint32](b)
	if len(c) != 4 {
		t.Fatalf("length mismatch: want=2 got=%d", len(b))
	}
	if cap(c) != 12 {
		t.Fatalf("capacity mismatch: want=7 got=%d", cap(b))
	}
	for i := range c {
		if c[i] != a[i] {
			t.Errorf("wrong value at index %d: want=%d got=%d", i, a[i], c[i])
		}
	}
}

func TestUnsafeCastBytes(t *testing.T) {
	// The bytes must alias the string rather than copy it, which is the whole
	// reason the function exists.
	s := "hello, world"
	b := unsafecast.Bytes(s)
	if len(b) != len(s) {
		t.Fatalf("length mismatch: want=%d got=%d", len(s), len(b))
	}
	if cap(b) != len(s) {
		t.Fatalf("capacity mismatch: want=%d got=%d", len(s), cap(b))
	}
	if string(b) != s {
		t.Errorf("content mismatch: want=%q got=%q", s, string(b))
	}
	if unsafe.SliceData(b) != unsafe.StringData(s) {
		t.Error("Bytes copied the string instead of aliasing it")
	}
}

func TestUnsafeCastBytesEmpty(t *testing.T) {
	// The empty string has no backing array to point at; the result only has
	// to be a usable empty slice.
	b := unsafecast.Bytes("")
	if len(b) != 0 {
		t.Errorf("length mismatch: want=0 got=%d", len(b))
	}
	if len(b) != 0 || string(b) != "" {
		t.Errorf("content mismatch: want empty got=%q", string(b))
	}
}

func TestUnsafeCastBytesRoundTrip(t *testing.T) {
	// Bytes and String are inverses, and neither copies, so a round trip
	// through both lands back on the same backing array.
	for _, s := range []string{"", "a", "hello, world", "\x00\xff binary \x01"} {
		if got := unsafecast.String(unsafecast.Bytes(s)); got != s {
			t.Errorf("String(Bytes(%q)) = %q", s, got)
		}
	}

	data := []byte("mutable backing array")
	if got := unsafecast.Bytes(unsafecast.String(data)); unsafe.SliceData(got) != unsafe.SliceData(data) {
		t.Error("Bytes(String(data)) did not alias the original array")
	}
}
