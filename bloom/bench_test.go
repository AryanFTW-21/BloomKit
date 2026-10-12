package bloom

import (
	"fmt"
	"testing"

	"github.com/AryanFTW-21/bloomkit/hashing"
)

const benchN = 1_000_000

var benchKeys = func() [][]byte {
	k := make([][]byte, benchN)
	for i := range k {
		k[i] = []byte(fmt.Sprintf("user-%d@example.com", i))
	}
	return k
}()

var sink bool

func benchFilter(b *testing.B, s hashing.Strategy) *Filter {
	f, err := New(benchN, 0.01, WithStrategy(s))
	if err != nil {
		b.Fatal(err)
	}
	for _, k := range benchKeys[:benchN/2] {
		f.Add(k)
	}
	return f
}

func BenchmarkAdd(b *testing.B) {
	for name, s := range map[string]hashing.Strategy{"seeded": hashing.StrategySeeded, "double": hashing.StrategyDouble} {
		b.Run(name, func(b *testing.B) {
			f, _ := New(benchN, 0.01, WithStrategy(s))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				f.Add(benchKeys[i%benchN])
			}
		})
	}
}

func BenchmarkContains(b *testing.B) {
	for name, s := range map[string]hashing.Strategy{"seeded": hashing.StrategySeeded, "double": hashing.StrategyDouble} {
		b.Run(name, func(b *testing.B) {
			f := benchFilter(b, s)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				sink = f.Contains(benchKeys[i%benchN])
			}
		})
	}
}

func BenchmarkContainsParallel(b *testing.B) {
	f := benchFilter(b, hashing.StrategyDouble)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			sink = f.Contains(benchKeys[i%benchN])
			i++
		}
	})
}
