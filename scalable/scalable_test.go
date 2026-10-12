package scalable

import (
	"fmt"
	"testing"

	"github.com/AryanFTW-21/bloomkit"
	"github.com/AryanFTW-21/bloomkit/internal/conformance"
)

func TestConformance(t *testing.T) {
	conformance.Run(t, func(t *testing.T, n uint64, p float64) bloomkit.Filter {
		f, err := New(n, p)
		if err != nil {
			t.Fatal(err)
		}
		return f
	}, conformance.Options{})
}

// Start tiny and add 100x more items than the first layer holds.
// The filter must grow and still keep the false-positive rate under target.
func TestGrowsAndKeepsErrorBound(t *testing.T) {
	const initial, n, p = 1000, 100_000, 0.01
	f, err := New(initial, p)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < n; i++ {
		f.Add([]byte(fmt.Sprintf("item-%d", i)))
	}
	if f.Layers() < 5 {
		t.Fatalf("expected the filter to grow, got %d layers", f.Layers())
	}
	for i := 0; i < n; i++ {
		if !f.Contains([]byte(fmt.Sprintf("item-%d", i))) {
			t.Fatalf("false negative for item-%d", i)
		}
	}
	fp := 0
	for i := 0; i < n; i++ {
		if f.Contains([]byte(fmt.Sprintf("absent-%d", i))) {
			fp++
		}
	}
	rate := float64(fp) / n
	t.Logf("%d layers, %.1f KB, false-positive rate %.4f (target %.2f)",
		f.Layers(), float64(f.SizeBytes())/1024, rate, p)
	if rate > p {
		t.Fatalf("false-positive rate %.4f above target %.2f", rate, p)
	}
}

func TestDuplicatesDoNotGrow(t *testing.T) {
	f, _ := New(100, 0.01)
	for i := 0; i < 10_000; i++ {
		f.Add([]byte("same"))
	}
	if f.Layers() != 1 || f.Count() != 1 {
		t.Fatalf("duplicates grew the filter: %d layers, count %d", f.Layers(), f.Count())
	}
}
