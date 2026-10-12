package counting

import (
	"errors"
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

func TestRemove(t *testing.T) {
	f, _ := New(1000, 0.01)
	f.Add([]byte("alice"))
	f.Add([]byte("bob"))
	if err := f.Remove([]byte("alice")); err != nil {
		t.Fatal(err)
	}
	if f.Contains([]byte("alice")) {
		t.Fatal("alice still present after Remove")
	}
	if !f.Contains([]byte("bob")) {
		t.Fatal("removing alice removed bob")
	}
}

func TestRemoveAbsent(t *testing.T) {
	f, _ := New(1000, 0.01)
	if err := f.Remove([]byte("ghost")); !errors.Is(err, ErrNotPresent) {
		t.Fatalf("Remove(absent) = %v, want ErrNotPresent", err)
	}
}

func TestDuplicateAddsNeedMatchingRemoves(t *testing.T) {
	f, _ := New(1000, 0.01)
	f.Add([]byte("x"))
	f.Add([]byte("x"))
	_ = f.Remove([]byte("x"))
	if !f.Contains([]byte("x")) {
		t.Fatal("x added twice but gone after one Remove")
	}
	_ = f.Remove([]byte("x"))
	if f.Contains([]byte("x")) {
		t.Fatal("x still present after two Removes")
	}
}

// Removing half the items must not cause false negatives for the other half.
func TestRemoveHalf(t *testing.T) {
	const n = 20_000
	f, _ := New(n, 0.01)
	for i := 0; i < n; i++ {
		f.Add([]byte(fmt.Sprintf("k%d", i)))
	}
	for i := 0; i < n; i += 2 {
		if err := f.Remove([]byte(fmt.Sprintf("k%d", i))); err != nil {
			t.Fatal(err)
		}
	}
	for i := 1; i < n; i += 2 {
		if !f.Contains([]byte(fmt.Sprintf("k%d", i))) {
			t.Fatalf("false negative for k%d after removing others", i)
		}
	}
}
