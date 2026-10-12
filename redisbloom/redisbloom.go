// Package redisbloom is a Bloom filter stored in a Redis bitmap, so many
// processes or servers can share one filter.
//
// Each Add or Contains is a single pipelined round trip: k SETBIT or GETBIT
// commands sent together. Hashing happens in the client using the same code
// as the in-memory filter.
package redisbloom

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"

	"github.com/AryanFTW-21/bloomkit/hashing"
	"github.com/AryanFTW-21/bloomkit/sizing"
)

// Redis strings are at most 512 MB, so a bitmap holds at most 2^32 bits.
const maxBits = 1 << 32

// ErrParamsMismatch means the key already holds a filter with different
// parameters. Mixing parameters would silently break lookups.
var ErrParamsMismatch = errors.New("redisbloom: key already holds a filter with different m/k")

// Filter is a Bloom filter whose bits live in Redis.
type Filter struct {
	client redis.Cmdable
	key    string
	m, k   uint64
}

// New connects a filter sized for n items at false-positive rate p to key.
// Parameters are stored in "<key>:meta"; every process using the same key
// must use the same n and p, and New returns ErrParamsMismatch otherwise.
func New(ctx context.Context, client redis.Cmdable, key string, n uint64, p float64) (*Filter, error) {
	m, k, err := sizing.Optimal(n, p)
	if err != nil {
		return nil, err
	}
	if m > maxBits {
		return nil, fmt.Errorf("redisbloom: needs %d bits, Redis allows at most %d", m, uint64(maxBits))
	}
	meta := key + ":meta"
	if _, err := client.HSetNX(ctx, meta, "m", m).Result(); err != nil {
		return nil, err
	}
	if _, err := client.HSetNX(ctx, meta, "k", k).Result(); err != nil {
		return nil, err
	}
	stored, err := client.HMGet(ctx, meta, "m", "k").Result()
	if err != nil {
		return nil, err
	}
	if fmt.Sprint(stored[0]) != strconv.FormatUint(m, 10) || fmt.Sprint(stored[1]) != strconv.FormatUint(k, 10) {
		return nil, ErrParamsMismatch
	}
	return &Filter{client: client, key: key, m: m, k: k}, nil
}

func (f *Filter) positions(data []byte, buf *[32]uint64) []uint64 {
	return hashing.Double(data, f.k, f.m, buf[:0])
}

// Add inserts data.
func (f *Filter) Add(ctx context.Context, data []byte) error {
	var buf [32]uint64
	pipe := f.client.Pipeline()
	for _, p := range f.positions(data, &buf) {
		pipe.SetBit(ctx, f.key, int64(p), 1)
	}
	_, err := pipe.Exec(ctx)
	return err
}

// Contains reports whether data might have been added.
func (f *Filter) Contains(ctx context.Context, data []byte) (bool, error) {
	var buf [32]uint64
	pipe := f.client.Pipeline()
	pos := f.positions(data, &buf)
	cmds := make([]*redis.IntCmd, len(pos))
	for i, p := range pos {
		cmds[i] = pipe.GetBit(ctx, f.key, int64(p))
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return false, err
	}
	for _, c := range cmds {
		if c.Val() == 0 {
			return false, nil
		}
	}
	return true, nil
}

// Clear deletes the filter and its metadata from Redis.
func (f *Filter) Clear(ctx context.Context) error {
	return f.client.Del(ctx, f.key, f.key+":meta").Err()
}

// M returns the number of bits; K the number of hashes.
func (f *Filter) M() uint64 { return f.m }
func (f *Filter) K() uint64 { return f.k }
