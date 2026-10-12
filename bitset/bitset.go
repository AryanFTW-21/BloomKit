// Package bitset is a compact, thread-safe array of bits packed into 64-bit words.
package bitset

import (
	"encoding/binary"
	"fmt"
	"io"
	"math/bits"
	"sync/atomic"
)

// BitSet stores size bits in size/64 uint64 words.
// Set and Get use atomic operations, so they are safe to call concurrently.
type BitSet struct {
	words []uint64
	size  uint64
}

// New returns a BitSet holding size bits, all 0.
func New(size uint64) *BitSet {
	return &BitSet{words: make([]uint64, (size+63)/64), size: size}
}

// Set turns bit i on. i must be < Size().
func (b *BitSet) Set(i uint64) {
	atomic.OrUint64(&b.words[i/64], 1<<(i%64))
}

// Get reports whether bit i is on. i must be < Size().
func (b *BitSet) Get(i uint64) bool {
	return atomic.LoadUint64(&b.words[i/64])&(1<<(i%64)) != 0
}

// Size returns the number of bits.
func (b *BitSet) Size() uint64 { return b.size }

// Bytes returns the memory used by the bits.
func (b *BitSet) Bytes() uint64 { return uint64(len(b.words)) * 8 }

// Count returns how many bits are set.
func (b *BitSet) Count() uint64 {
	var n int
	for i := range b.words {
		n += bits.OnesCount64(atomic.LoadUint64(&b.words[i]))
	}
	return uint64(n)
}

// WriteTo writes the bitset in a portable little-endian format.
func (b *BitSet) WriteTo(w io.Writer) (int64, error) {
	buf := make([]byte, 8+8*len(b.words))
	binary.LittleEndian.PutUint64(buf, b.size)
	for i := range b.words {
		binary.LittleEndian.PutUint64(buf[8+8*i:], atomic.LoadUint64(&b.words[i]))
	}
	n, err := w.Write(buf)
	return int64(n), err
}

// ReadFrom reads a bitset written by WriteTo.
func ReadFrom(r io.Reader) (*BitSet, error) {
	var hdr [8]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, fmt.Errorf("bitset: reading size: %w", err)
	}
	size := binary.LittleEndian.Uint64(hdr[:])
	if size > 1<<40 {
		return nil, fmt.Errorf("bitset: size %d too large", size)
	}
	b := New(size)
	buf := make([]byte, 8*len(b.words))
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, fmt.Errorf("bitset: reading words: %w", err)
	}
	for i := range b.words {
		b.words[i] = binary.LittleEndian.Uint64(buf[8*i:])
	}
	return b, nil
}
