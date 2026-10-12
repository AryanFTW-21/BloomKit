// Package hashing turns an item into the k bit positions a Bloom filter uses.
package hashing

const (
	fnvOffset uint64 = 14695981039346656037
	fnvPrime  uint64 = 1099511628211
)

// FNV1a is the 64-bit FNV-1a hash. Mixing the seed into the starting value
// makes each seed behave like a different hash function.
func FNV1a(seed uint64, data []byte) uint64 {
	h := fnvOffset ^ seed
	for _, c := range data {
		h ^= uint64(c)
		h *= fnvPrime
	}
	return h
}

// Mix64 scrambles the bits of h (the MurmurHash3 finalizer) so that
// similar inputs produce very different outputs.
func Mix64(h uint64) uint64 {
	h ^= h >> 33
	h *= 0xff51afd7ed558ccd
	h ^= h >> 33
	h *= 0xc4ceb9fe1a85ec53
	h ^= h >> 33
	return h
}

// Strategy chooses how the k bit positions are derived from an item.
//
// It is an enum rather than a function value on purpose: calls through a
// function value hide the callee from Go's escape analysis, which forces the
// caller's position buffer onto the heap (one allocation per Add/Contains).
type Strategy uint8

const (
	// StrategyDouble hashes once and derives k positions (the default).
	StrategyDouble Strategy = iota
	// StrategySeeded runs k separately seeded hashes.
	StrategySeeded
)

// Positions appends the k positions in [0, m) for data to out and returns it.
func (s Strategy) Positions(data []byte, k, m uint64, out []uint64) []uint64 {
	if s == StrategySeeded {
		return Seeded(data, k, m, out)
	}
	return Double(data, k, m, out)
}

// String returns the strategy's CLI name.
func (s Strategy) String() string {
	if s == StrategySeeded {
		return "seeded"
	}
	return "double"
}

// Double uses Kirsch-Mitzenmacher double hashing: hash the data once,
// derive two values h1 and h2, and use h1 + i*h2 as the i-th position.
// One pass over the data instead of k, with the same false-positive rate.
func Double(data []byte, k, m uint64, out []uint64) []uint64 {
	h := FNV1a(0, data)
	h1 := Mix64(h)
	h2 := Mix64(h^0x9e3779b97f4a7c15) | 1 // odd, so it never gets stuck at one position
	for i := uint64(0); i < k; i++ {
		out = append(out, (h1+i*h2)%m)
	}
	return out
}

// Seeded runs k independently seeded hashes over the data.
// Simple and well understood, but reads the data k times.
func Seeded(data []byte, k, m uint64, out []uint64) []uint64 {
	for i := uint64(0); i < k; i++ {
		out = append(out, FNV1a(i, data)%m)
	}
	return out
}

// ByName looks up a strategy by its CLI name.
func ByName(name string) (Strategy, bool) {
	switch name {
	case "double":
		return StrategyDouble, true
	case "seeded":
		return StrategySeeded, true
	}
	return 0, false
}
