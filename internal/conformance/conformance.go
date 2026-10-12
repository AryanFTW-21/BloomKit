// Package conformance is a test suite every filter in bloomkit must pass.
package conformance

import (
	"fmt"
	"sync"
	"testing"

	"github.com/AryanFTW-21/bloomkit"
)

// NewFunc builds an empty filter sized for n items at false-positive rate p.
type NewFunc func(t *testing.T, n uint64, p float64) bloomkit.Filter

// Options turns off checks that do not apply to a filter.
type Options struct {
	SkipConcurrency bool
	Items           int // items for the false-positive test; default 100_000
}

func key(prefix string, i int) []byte { return []byte(fmt.Sprintf("%s-%d", prefix, i)) }

// Run runs the suite against filters created by newFilter.
func Run(t *testing.T, newFilter NewFunc, opts Options) {
	n := opts.Items
	if n == 0 {
		n = 100_000
	}

	t.Run("EmptyContainsNothing", func(t *testing.T) {
		f := newFilter(t, 1000, 0.01)
		for i := 0; i < 1000; i++ {
			if f.Contains(key("absent", i)) {
				t.Fatalf("empty filter contains %q", key("absent", i))
			}
		}
	})

	t.Run("NoFalseNegatives", func(t *testing.T) {
		f := newFilter(t, uint64(n), 0.01)
		for i := 0; i < n; i++ {
			f.Add(key("item", i))
		}
		for i := 0; i < n; i++ {
			if !f.Contains(key("item", i)) {
				t.Fatalf("false negative for %q", key("item", i))
			}
		}
	})

	t.Run("FalsePositiveRate", func(t *testing.T) {
		for _, p := range []float64{0.1, 0.01, 0.001} {
			t.Run(fmt.Sprintf("p=%v", p), func(t *testing.T) {
				f := newFilter(t, uint64(n), p)
				for i := 0; i < n; i++ {
					f.Add(key("item", i))
				}
				fp := 0
				for i := 0; i < n; i++ {
					if f.Contains(key("absent", i)) {
						fp++
					}
				}
				got := float64(fp) / float64(n)
				t.Logf("target %.4f, measured %.4f", p, got)
				if got > p*1.5 { // slack for sampling noise
					t.Errorf("false-positive rate %.4f is over 1.5x the target %.4f", got, p)
				}
			})
		}
	})

	if !opts.SkipConcurrency {
		t.Run("ConcurrentAdd", func(t *testing.T) {
			const workers, per = 8, 5000
			f := newFilter(t, workers*per, 0.01)
			var wg sync.WaitGroup
			for w := 0; w < workers; w++ {
				wg.Add(1)
				go func(w int) {
					defer wg.Done()
					for i := 0; i < per; i++ {
						f.Add(key(fmt.Sprintf("w%d", w), i))
						f.Contains(key("absent", i))
					}
				}(w)
			}
			wg.Wait()
			for w := 0; w < workers; w++ {
				for i := 0; i < per; i++ {
					if !f.Contains(key(fmt.Sprintf("w%d", w), i)) {
						t.Fatalf("lost concurrent add: worker %d item %d", w, i)
					}
				}
			}
		})
	}
}
