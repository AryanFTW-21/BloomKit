// Package counting is a counting Bloom filter: each position holds a small
// counter instead of a single bit, so items can be removed.
package counting

import (
	"errors"
	"sync"

	"github.com/AryanFTW-21/bloomkit"
	"github.com/AryanFTW-21/bloomkit/hashing"
	"github.com/AryanFTW-21/bloomkit/sizing"
)

// ErrNotPresent is returned by Remove for an item that is definitely not in the filter.
var ErrNotPresent = errors.New("counting: item not present")

const maxCount = 255

// Filter is a counting Bloom filter with 8-bit counters.
// It uses 8x the memory of a plain Bloom filter in exchange for Remove.
// A counter that reaches 255 is "stuck" and never decremented, so it can
// never cause a false negative; this only happens with extreme overloading.
type Filter struct {
	mu       sync.RWMutex
	counters []uint8
	m, k     uint64
}

var _ bloomkit.Remover = (*Filter)(nil)

// New returns a filter sized for n items at false-positive rate p.
func New(n uint64, p float64) (*Filter, error) {
	m, k, err := sizing.Optimal(n, p)
	if err != nil {
		return nil, err
	}
	return &Filter{counters: make([]uint8, m), m: m, k: k}, nil
}

func (f *Filter) positions(data []byte, buf *[32]uint64) []uint64 {
	return hashing.Double(data, f.k, f.m, buf[:0])
}

// Add inserts data.
func (f *Filter) Add(data []byte) {
	var buf [32]uint64
	pos := f.positions(data, &buf)
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, p := range pos {
		if f.counters[p] < maxCount {
			f.counters[p]++
		}
	}
}

// Contains reports whether data might be in the filter.
func (f *Filter) Contains(data []byte) bool {
	var buf [32]uint64
	pos := f.positions(data, &buf)
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.allNonZero(pos)
}

func (f *Filter) allNonZero(pos []uint64) bool {
	for _, p := range pos {
		if f.counters[p] == 0 {
			return false
		}
	}
	return true
}

// Remove deletes one occurrence of data. Only remove items you added:
// removing a false positive would decrement other items' counters and
// could cause false negatives for them.
func (f *Filter) Remove(data []byte) error {
	var buf [32]uint64
	pos := f.positions(data, &buf)
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.allNonZero(pos) {
		return ErrNotPresent
	}
	for _, p := range pos {
		if f.counters[p] < maxCount {
			f.counters[p]--
		}
	}
	return nil
}

// SizeBytes returns the memory used by the counters.
func (f *Filter) SizeBytes() uint64 { return f.m }
