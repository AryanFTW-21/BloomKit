// Package bloom is a classic Bloom filter.
// Add and Contains are lock-free and safe to call from many goroutines.
package bloom

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync/atomic"

	"github.com/AryanFTW-21/bloomkit"
	"github.com/AryanFTW-21/bloomkit/bitset"
	"github.com/AryanFTW-21/bloomkit/hashing"
	"github.com/AryanFTW-21/bloomkit/sizing"
)

// Filter is a Bloom filter backed by a thread-safe bitset.
type Filter struct {
	bits     *bitset.BitSet
	m, k     uint64
	strategy hashing.Strategy
	added    atomic.Uint64
}

var _ bloomkit.Filter = (*Filter)(nil)

// Option configures a Filter.
type Option func(*Filter)

// WithStrategy picks how bit positions are computed. The default is
// hashing.StrategyDouble.
func WithStrategy(s hashing.Strategy) Option {
	return func(f *Filter) { f.strategy = s }
}

// New returns a filter sized for n items at false-positive rate p.
func New(n uint64, p float64, opts ...Option) (*Filter, error) {
	m, k, err := sizing.Optimal(n, p)
	if err != nil {
		return nil, err
	}
	return NewWithSize(m, k, opts...), nil
}

// NewWithSize returns a filter with exactly m bits and k hashes.
func NewWithSize(m, k uint64, opts ...Option) *Filter {
	f := &Filter{bits: bitset.New(m), m: m, k: k, strategy: hashing.StrategyDouble}
	for _, o := range opts {
		o(f)
	}
	return f
}

// positions computes the k bit positions for data. buf is a stack array in
// the caller, so this does not allocate for k <= 32.
func (f *Filter) positions(data []byte, buf *[32]uint64) []uint64 {
	return f.strategy.Positions(data, f.k, f.m, buf[:0])
}

// Add inserts data.
func (f *Filter) Add(data []byte) {
	var buf [32]uint64
	for _, p := range f.positions(data, &buf) {
		f.bits.Set(p)
	}
	f.added.Add(1)
}

// Contains reports whether data might have been added.
// false means definitely not added; true means probably added.
func (f *Filter) Contains(data []byte) bool {
	var buf [32]uint64
	for _, p := range f.positions(data, &buf) {
		if !f.bits.Get(p) {
			return false
		}
	}
	return true
}

// AddString and ContainsString are conveniences for string keys.
func (f *Filter) AddString(s string) { f.Add([]byte(s)) }

func (f *Filter) ContainsString(s string) bool { return f.Contains([]byte(s)) }

// M returns the number of bits.
func (f *Filter) M() uint64 { return f.m }

// K returns the number of hash functions.
func (f *Filter) K() uint64 { return f.k }

// Added returns how many times Add has been called.
func (f *Filter) Added() uint64 { return f.added.Load() }

// SizeBytes returns the memory used by the bit array.
func (f *Filter) SizeBytes() uint64 { return f.bits.Bytes() }

// FillRatio is the fraction of bits set. Around 0.5 means the filter is at
// its designed capacity; much higher means false positives are climbing.
func (f *Filter) FillRatio() float64 { return float64(f.bits.Count()) / float64(f.m) }

// EstimatedFalsePositiveRate is the expected rate given how much has been added.
func (f *Filter) EstimatedFalsePositiveRate() float64 {
	return sizing.FalsePositiveRate(f.m, f.k, f.Added())
}

const magic = "BKF1"

// WriteTo saves the filter so it can be loaded later with ReadFrom.
// The hashing strategy is not saved; load with the same strategy you used.
func (f *Filter) WriteTo(w io.Writer) (int64, error) {
	var hdr [4 + 8*3]byte
	copy(hdr[:], magic)
	binary.LittleEndian.PutUint64(hdr[4:], f.m)
	binary.LittleEndian.PutUint64(hdr[12:], f.k)
	binary.LittleEndian.PutUint64(hdr[20:], f.Added())
	n, err := w.Write(hdr[:])
	if err != nil {
		return int64(n), err
	}
	n2, err := f.bits.WriteTo(w)
	return int64(n) + n2, err
}

// ReadFrom loads a filter saved with WriteTo.
func ReadFrom(r io.Reader, opts ...Option) (*Filter, error) {
	var hdr [4 + 8*3]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, fmt.Errorf("bloom: reading header: %w", err)
	}
	if string(hdr[:4]) != magic {
		return nil, errors.New("bloom: not a bloomkit filter file")
	}
	m := binary.LittleEndian.Uint64(hdr[4:])
	k := binary.LittleEndian.Uint64(hdr[12:])
	bits, err := bitset.ReadFrom(r)
	if err != nil {
		return nil, err
	}
	if bits.Size() != m || k == 0 || k > 32 {
		return nil, errors.New("bloom: corrupt filter header")
	}
	f := NewWithSize(m, k, opts...)
	f.bits = bits
	f.added.Store(binary.LittleEndian.Uint64(hdr[20:]))
	return f, nil
}
