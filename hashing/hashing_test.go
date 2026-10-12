package hashing

import (
	"fmt"
	"testing"
)

// Reference values from the FNV-1a specification.
func TestFNV1aKnownValues(t *testing.T) {
	cases := map[string]uint64{
		"":       0xcbf29ce484222325,
		"a":      0xaf63dc4c8601ec8c,
		"foobar": 0x85944171f73967e8,
	}
	for in, want := range cases {
		if got := FNV1a(0, []byte(in)); got != want {
			t.Errorf("FNV1a(%q) = %#x, want %#x", in, got, want)
		}
	}
}

func TestSeedsDiffer(t *testing.T) {
	if FNV1a(0, []byte("alice")) == FNV1a(1, []byte("alice")) {
		t.Fatal("different seeds gave the same hash")
	}
}

func TestStrategies(t *testing.T) {
	for name, s := range map[string]Strategy{"double": StrategyDouble, "seeded": StrategySeeded} {
		t.Run(name, func(t *testing.T) {
			const k, m = 7, 1000
			a := s.Positions([]byte("alice"), k, m, nil)
			b := s.Positions([]byte("alice"), k, m, nil)
			if len(a) != k {
				t.Fatalf("got %d positions, want %d", len(a), k)
			}
			for i := range a {
				if a[i] >= m {
					t.Fatalf("position %d out of range", a[i])
				}
				if a[i] != b[i] {
					t.Fatal("same input gave different positions")
				}
			}
		})
	}
}

// Positions should spread evenly over the bit array; a bad hash
// would pile items into a few buckets and raise the false-positive rate.
func TestDistribution(t *testing.T) {
	for name, s := range map[string]Strategy{"double": StrategyDouble, "seeded": StrategySeeded} {
		t.Run(name, func(t *testing.T) {
			const buckets, items, k = 64, 64_000, 4
			counts := make([]int, buckets)
			var out []uint64
			for i := 0; i < items; i++ {
				out = s.Positions([]byte(fmt.Sprintf("key-%d", i)), k, buckets, out[:0])
				for _, p := range out {
					counts[p]++
				}
			}
			want := items * k / buckets
			for b, c := range counts {
				if c < want*9/10 || c > want*11/10 {
					t.Errorf("bucket %d has %d hits, want about %d", b, c, want)
				}
			}
		})
	}
}

func TestByName(t *testing.T) {
	if _, ok := ByName("double"); !ok {
		t.Fatal("double not found")
	}
	if _, ok := ByName("nope"); ok {
		t.Fatal("unknown name found")
	}
}
