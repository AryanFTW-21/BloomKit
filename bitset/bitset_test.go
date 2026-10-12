package bitset

import (
	"bytes"
	"sync"
	"testing"
)

func TestNewIsEmpty(t *testing.T) {
	b := New(1000)
	for i := uint64(0); i < 1000; i++ {
		if b.Get(i) {
			t.Fatalf("bit %d set in new bitset", i)
		}
	}
	if b.Count() != 0 {
		t.Fatalf("Count() = %d, want 0", b.Count())
	}
}

func TestSetAndGet(t *testing.T) {
	b := New(200)
	for _, i := range []uint64{0, 1, 63, 64, 127, 199} {
		b.Set(i)
		if !b.Get(i) {
			t.Fatalf("bit %d not set after Set", i)
		}
	}
	if b.Get(2) || b.Get(65) {
		t.Fatal("unrelated bits were set")
	}
	if b.Count() != 6 {
		t.Fatalf("Count() = %d, want 6", b.Count())
	}
}

func TestBytes(t *testing.T) {
	if got := New(65).Bytes(); got != 16 {
		t.Fatalf("New(65).Bytes() = %d, want 16", got)
	}
}

// Many goroutines set different bits in the same words. Without atomics,
// concurrent read-modify-write would lose some of them.
func TestConcurrentSet(t *testing.T) {
	const size = 64 * 100
	b := New(size)
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := uint64(g); i < size; i += 8 {
				b.Set(i)
			}
		}(g)
	}
	wg.Wait()
	if b.Count() != size {
		t.Fatalf("Count() = %d, want %d (lost updates)", b.Count(), size)
	}
}

func TestWriteReadRoundTrip(t *testing.T) {
	b := New(1000)
	for _, i := range []uint64{0, 7, 64, 500, 999} {
		b.Set(i)
	}
	var buf bytes.Buffer
	if _, err := b.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}
	got, err := ReadFrom(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if got.Size() != b.Size() || got.Count() != b.Count() {
		t.Fatalf("round trip: size %d/%d count %d/%d", got.Size(), b.Size(), got.Count(), b.Count())
	}
	for _, i := range []uint64{0, 7, 64, 500, 999} {
		if !got.Get(i) {
			t.Fatalf("bit %d lost in round trip", i)
		}
	}
}

func TestReadFromTruncated(t *testing.T) {
	if _, err := ReadFrom(bytes.NewReader([]byte{1, 2, 3})); err == nil {
		t.Fatal("expected error for truncated input")
	}
}
