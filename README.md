# bloomkit

A Bloom filter library in Go: thread-safe, allocation-free, with counting,
scalable and Redis-backed variants.

A Bloom filter answers "have I seen this before?" using a tiny fraction of the
memory a hash set needs. It may occasionally say *yes* for something it has
never seen (a false positive, at a rate you choose), but it never says *no*
for something it has.

## Results

1M keys like `user-123@example.com`, target false-positive rate 1%:

| | Memory | Notes |
|---|---|---|
| `map[string]struct{}` | 79.9 MB | exact, stores every key |
| `bloom.Filter` | **1.20 MB** | **67x less**, 9.59 bits per item |

| Measurement | Result |
|---|---|
| False-positive rate (target 1%) | 1.009% measured, 1.004% predicted |
| False negatives | 0 |
| `Contains`, k seeded hashes | 150 ns |
| `Contains`, double hashing | **76 ns** (2x faster) |
| `Add`, k seeded hashes → double hashing | 166 ns → 73 ns |
| Allocations per operation | 0 |

Reproduce with `make compare`, `make fp` and `make bench`. Timings are medians
of 7 runs on one machine and will vary on yours.

## Packages

| Package | What it is |
|---|---|
| `bloom` | Classic Bloom filter. Lock-free and safe for concurrent use; save/load to disk |
| `counting` | Counting Bloom filter with 8-bit counters; supports `Remove` |
| `scalable` | Grows by adding layers as it fills, keeping the total error rate under the target |
| `redisbloom` | Bits stored in a Redis bitmap, so several servers share one filter |
| `bitset` | Bits packed into `uint64` words with atomic set/get |
| `hashing` | FNV-1a, plus double hashing and seeded strategies |
| `sizing` | Picks bit count `m` and hash count `k` from `n` and `p` |
| `internal/conformance` | Test suite every filter must pass |

## Usage

```go
f, err := bloom.New(1_000_000, 0.01) // expect 1M items, 1% false positives
if err != nil {
	log.Fatal(err)
}
f.AddString("aryan")
f.ContainsString("aryan") // true
f.ContainsString("zoya")  // false: definitely never added
```

Removing items:

```go
c, _ := counting.New(10_000, 0.01)
c.Add([]byte("session-42"))
_ = c.Remove([]byte("session-42"))
```

Unknown item count:

```go
s, _ := scalable.New(1_000, 0.01) // starts small, grows as needed
```

Shared across servers:

```go
rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
r, _ := redisbloom.New(ctx, rdb, "usernames", 1_000_000, 0.01)
_ = r.Add(ctx, []byte("aryan"))
ok, _ := r.Contains(ctx, []byte("aryan"))
```

## CLI

```
$ go run ./cmd/bloomkit info
n=1000000 p=0.01 -> m=9585059 bits, k=7 hashes, 1.20 MB, 9.59 bits/item

$ go run ./cmd/bloomkit -load testdata/usernames.txt check aryan zoya
aryan                probably taken
zoya                 available (definitely never added)
```

Other commands: `fp`, `compare`, `repl`. Run with no arguments for help.

## How it works

**Sizing.** For `n` items at false-positive rate `p`:

- bits: `m = -n·ln(p) / (ln 2)²`
- hashes: `k = (m/n)·ln 2`

At this `k`, about half the bits end up set when the filter is full, which
balances "too few bits checked" against "too many bits set".

**Double hashing.** Instead of running `k` separate hash functions over the
data, hash it once, derive two values `h1` and `h2`, and use `h1 + i·h2` as
the i-th position (Kirsch and Mitzenmacher, 2006). This reads the data once
instead of `k` times, with no measurable change in false-positive rate.

**Thread safety.** Bits are set with `atomic.OrUint64` and read with
`atomic.LoadUint64`, so concurrent `Add` and `Contains` calls need no lock.
Bits only ever go from 0 to 1, so there is no read-modify-write conflict to
protect against.

**Zero allocations.** Each call computes its positions into a fixed-size array
on the stack. The hashing strategy is an enum rather than a function value,
because calling through a function value hides the callee from Go's escape
analysis and forces that array onto the heap.

**Counting filter.** Each position is an 8-bit counter instead of a bit.
`Add` increments, `Remove` decrements. A counter that reaches 255 stays there
forever, which can only cause extra false positives, never false negatives.

**Scalable filter.** Layer `i` holds `n0·2^i` items at error rate
`p0·0.8^i`. Because those error rates form a geometric series, the total
false-positive rate stays below `p` however many layers are added
(Almeida et al., 2007).

**Redis.** The filter's bits are a Redis bitmap. Each `Add` or `Contains`
sends its `k` `SETBIT`/`GETBIT` commands in one pipelined round trip. The
parameters are stored next to the bitmap, and connecting with different ones
returns an error instead of silently breaking lookups.

## Development

```
make test     # all tests
make race     # all tests with the race detector
make bench    # benchmarks
```

The Redis tests use an in-memory Redis server, so no Redis install is needed.
