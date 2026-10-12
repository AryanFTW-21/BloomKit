// Package bloomkit defines the interfaces shared by every filter in this module.
//
// Filters live in sub-packages:
//
//	bloom      - classic Bloom filter, lock-free and thread-safe
//	counting   - counting Bloom filter that supports Remove
//	scalable   - grows automatically as items are added
//	redisbloom - Bloom filter stored in Redis, shared across processes
package bloomkit

// Filter is a probabilistic set. Contains never returns false for an item
// that was added, but may return true for an item that was not (a false positive).
type Filter interface {
	Add(data []byte)
	Contains(data []byte) bool
}

// Remover is a Filter that also supports deleting items.
type Remover interface {
	Filter
	Remove(data []byte) error
}
