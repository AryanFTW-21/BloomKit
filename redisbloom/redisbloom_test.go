package redisbloom

import (
	"context"
	"errors"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/AryanFTW-21/bloomkit"
	"github.com/AryanFTW-21/bloomkit/internal/conformance"
)

// adapter lets the Redis filter run through the shared conformance suite,
// failing the test on any Redis error.
type adapter struct {
	t *testing.T
	f *Filter
}

func (a adapter) Add(d []byte) {
	if err := a.f.Add(context.Background(), d); err != nil {
		a.t.Fatal(err)
	}
}

func (a adapter) Contains(d []byte) bool {
	ok, err := a.f.Contains(context.Background(), d)
	if err != nil {
		a.t.Fatal(err)
	}
	return ok
}

func newClient(t *testing.T) *redis.Client {
	t.Helper()
	srv := miniredis.RunT(t) // in-memory Redis, no server needed
	c := redis.NewClient(&redis.Options{Addr: srv.Addr()})
	t.Cleanup(func() { c.Close() })
	return c
}

func TestConformance(t *testing.T) {
	conformance.Run(t, func(t *testing.T, n uint64, p float64) bloomkit.Filter {
		f, err := New(context.Background(), newClient(t), "test", n, p)
		if err != nil {
			t.Fatal(err)
		}
		return adapter{t, f}
	}, conformance.Options{Items: 10_000, SkipConcurrency: true})
}

// Two separate clients (think: two servers) share one filter.
func TestSharedAcrossClients(t *testing.T) {
	ctx := context.Background()
	srv := miniredis.RunT(t)
	c1 := redis.NewClient(&redis.Options{Addr: srv.Addr()})
	c2 := redis.NewClient(&redis.Options{Addr: srv.Addr()})
	defer c1.Close()
	defer c2.Close()

	f1, err := New(ctx, c1, "users", 1000, 0.01)
	if err != nil {
		t.Fatal(err)
	}
	f2, err := New(ctx, c2, "users", 1000, 0.01)
	if err != nil {
		t.Fatal(err)
	}
	if err := f1.Add(ctx, []byte("alice")); err != nil {
		t.Fatal(err)
	}
	ok, err := f2.Contains(ctx, []byte("alice"))
	if err != nil || !ok {
		t.Fatalf("second client did not see alice: ok=%v err=%v", ok, err)
	}
}

func TestParamsMismatch(t *testing.T) {
	ctx := context.Background()
	c := newClient(t)
	if _, err := New(ctx, c, "users", 1000, 0.01); err != nil {
		t.Fatal(err)
	}
	if _, err := New(ctx, c, "users", 5000, 0.01); !errors.Is(err, ErrParamsMismatch) {
		t.Fatalf("got %v, want ErrParamsMismatch", err)
	}
}

func TestClear(t *testing.T) {
	ctx := context.Background()
	c := newClient(t)
	f, _ := New(ctx, c, "users", 1000, 0.01)
	_ = f.Add(ctx, []byte("alice"))
	if err := f.Clear(ctx); err != nil {
		t.Fatal(err)
	}
	if ok, _ := f.Contains(ctx, []byte("alice")); ok {
		t.Fatal("alice still present after Clear")
	}
}
