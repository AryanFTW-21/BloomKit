package sizing

import (
	"math"
	"testing"
)

func TestOptimalKnownValues(t *testing.T) {
	// 1M items at 1% needs about 9.59 bits per item and 7 hashes.
	m, k, err := Optimal(1_000_000, 0.01)
	if err != nil {
		t.Fatal(err)
	}
	if m != 9_585_059 || k != 7 {
		t.Fatalf("Optimal(1e6, 0.01) = m %d, k %d; want 9585059, 7", m, k)
	}
}

func TestFalsePositiveRateMatchesTarget(t *testing.T) {
	for _, p := range []float64{0.1, 0.01, 0.001} {
		m, k, _ := Optimal(100_000, p)
		got := FalsePositiveRate(m, k, 100_000)
		if math.Abs(got-p)/p > 0.1 {
			t.Errorf("p=%v: formula gives %v", p, got)
		}
	}
}

func TestOptimalRejectsBadInput(t *testing.T) {
	for _, tc := range []struct {
		n uint64
		p float64
	}{{0, 0.01}, {10, 0}, {10, 1}, {10, -0.5}} {
		if _, _, err := Optimal(tc.n, tc.p); err == nil {
			t.Errorf("Optimal(%d, %v): expected error", tc.n, tc.p)
		}
	}
}
