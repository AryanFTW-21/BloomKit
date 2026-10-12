package bloom

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/AryanFTW-21/bloomkit"
	"github.com/AryanFTW-21/bloomkit/hashing"
	"github.com/AryanFTW-21/bloomkit/internal/conformance"
)

func TestConformanceDouble(t *testing.T) {
	conformance.Run(t, func(t *testing.T, n uint64, p float64) bloomkit.Filter {
		f, err := New(n, p)
		if err != nil {
			t.Fatal(err)
		}
		return f
	}, conformance.Options{})
}

func TestConformanceSeeded(t *testing.T) {
	conformance.Run(t, func(t *testing.T, n uint64, p float64) bloomkit.Filter {
		f, err := New(n, p, WithStrategy(hashing.StrategySeeded))
		if err != nil {
			t.Fatal(err)
		}
		return f
	}, conformance.Options{})
}

func TestNewRejectsBadInput(t *testing.T) {
	if _, err := New(0, 0.01); err == nil {
		t.Fatal("expected error for n=0")
	}
}

func TestStats(t *testing.T) {
	f, _ := New(10_000, 0.01)
	for i := 0; i < 10_000; i++ {
		f.AddString(fmt.Sprintf("user-%d", i))
	}
	if f.Added() != 10_000 {
		t.Fatalf("Added() = %d", f.Added())
	}
	// At designed capacity, about half the bits are set.
	if r := f.FillRatio(); r < 0.45 || r > 0.55 {
		t.Fatalf("FillRatio() = %.3f, want about 0.5", r)
	}
	if !f.ContainsString("user-42") {
		t.Fatal("ContainsString missed an added key")
	}
}

func TestWriteReadRoundTrip(t *testing.T) {
	f, _ := New(1000, 0.01)
	for i := 0; i < 1000; i++ {
		f.AddString(fmt.Sprintf("k%d", i))
	}
	var buf bytes.Buffer
	if _, err := f.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}
	g, err := ReadFrom(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if g.M() != f.M() || g.K() != f.K() || g.Added() != f.Added() {
		t.Fatal("parameters changed in round trip")
	}
	for i := 0; i < 1000; i++ {
		if !g.ContainsString(fmt.Sprintf("k%d", i)) {
			t.Fatalf("k%d lost in round trip", i)
		}
	}
}

func TestReadFromRejectsGarbage(t *testing.T) {
	if _, err := ReadFrom(bytes.NewReader([]byte("not a filter at all, sorry"))); err == nil {
		t.Fatal("expected error")
	}
}
