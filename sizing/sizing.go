// Package sizing computes how big a Bloom filter needs to be.
package sizing

import (
	"fmt"
	"math"
)

// Optimal returns the number of bits m and hash functions k for n items
// at false-positive rate p:
//
//	m = -n * ln(p) / ln(2)^2
//	k = (m / n) * ln(2)
func Optimal(n uint64, p float64) (m, k uint64, err error) {
	if n == 0 {
		return 0, 0, fmt.Errorf("sizing: n must be > 0")
	}
	if p <= 0 || p >= 1 {
		return 0, 0, fmt.Errorf("sizing: p must be in (0, 1), got %v", p)
	}
	mf := math.Ceil(-float64(n) * math.Log(p) / (math.Ln2 * math.Ln2))
	kf := math.Round(mf / float64(n) * math.Ln2)
	if kf < 1 {
		kf = 1
	}
	return uint64(mf), uint64(kf), nil
}

// FalsePositiveRate is the expected false-positive rate after n items
// are added to a filter with m bits and k hashes: (1 - e^(-kn/m))^k.
func FalsePositiveRate(m, k, n uint64) float64 {
	return math.Pow(1-math.Exp(-float64(k)*float64(n)/float64(m)), float64(k))
}
