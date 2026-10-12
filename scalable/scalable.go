// Package scalable is a Bloom filter that grows as items are added, so you
// do not need to know the item count up front (Almeida et al., 2007).
//
// It keeps a list of plain Bloom filters ("layers"). When the newest layer
// fills up, a bigger one is added with a tighter error rate. The error rates
// form a geometric series, so the total false-positive rate stays below the target.
package scalable

import (
	"sync"

	"github.com/AryanFTW-21/bloomkit"
	"github.com/AryanFTW-21/bloomkit/bloom"
)

const (
	growth     = 2   // each layer holds twice as many items as the last
	tightening = 0.8 // each layer's error rate is 0.8x the last
)

type layer struct {
	f        *bloom.Filter
	capacity uint64
}

// Filter is a scalable Bloom filter. It is safe for concurrent use.
type Filter struct {
	mu        sync.RWMutex
	layers    []layer
	nextCap   uint64
	nextP     float64
	itemCount uint64
}

var _ bloomkit.Filter = (*Filter)(nil)

// New returns a filter that starts with room for initial items and keeps the
// overall false-positive rate below p no matter how many items are added.
func New(initial uint64, p float64) (*Filter, error) {
	f := &Filter{nextCap: initial, nextP: p * (1 - tightening)}
	if err := f.grow(); err != nil {
		return nil, err
	}
	return f, nil
}

// grow appends a new layer. Callers must hold mu for writing.
func (f *Filter) grow() error {
	b, err := bloom.New(f.nextCap, f.nextP)
	if err != nil {
		return err
	}
	f.layers = append(f.layers, layer{f: b, capacity: f.nextCap})
	f.nextCap *= growth
	f.nextP *= tightening
	return nil
}

// Add inserts data, growing the filter if the current layer is full.
// Items that already appear present are skipped so duplicates do not use up capacity.
func (f *Filter) Add(data []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.containsLocked(data) {
		return
	}
	cur := f.layers[len(f.layers)-1]
	if cur.f.Added() >= cur.capacity {
		if err := f.grow(); err != nil {
			panic(err) // parameters were validated in New, so this cannot happen
		}
		cur = f.layers[len(f.layers)-1]
	}
	cur.f.Add(data)
	f.itemCount++
}

// Contains reports whether data might have been added.
func (f *Filter) Contains(data []byte) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.containsLocked(data)
}

func (f *Filter) containsLocked(data []byte) bool {
	// Newest layers hold the most items, so check them first.
	for i := len(f.layers) - 1; i >= 0; i-- {
		if f.layers[i].f.Contains(data) {
			return true
		}
	}
	return false
}

// Layers returns how many layers the filter has grown to.
func (f *Filter) Layers() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return len(f.layers)
}

// Count returns the number of distinct items added (approximately; a false
// positive on Add is counted as a duplicate).
func (f *Filter) Count() uint64 {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.itemCount
}

// SizeBytes returns the memory used by all layers.
func (f *Filter) SizeBytes() uint64 {
	f.mu.RLock()
	defer f.mu.RUnlock()
	var total uint64
	for _, l := range f.layers {
		total += l.f.SizeBytes()
	}
	return total
}
