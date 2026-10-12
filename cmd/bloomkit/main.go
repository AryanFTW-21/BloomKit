// Command bloomkit is a small CLI for trying out and measuring Bloom filters.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/AryanFTW-21/bloomkit/bloom"
	"github.com/AryanFTW-21/bloomkit/hashing"
)

func usage() {
	fmt.Fprint(os.Stderr, `usage: bloomkit [flags] <command> [args]

commands:
  info              show the size of a filter for -n items at -p error rate
  fp                add n items, query n absent ones, report the false-positive rate
  compare           memory and speed of the filter vs a Go map for n items
  check NAME...     check names against the filter (use -load to fill it first)
  repl              interactive "add NAME" / "check NAME"

flags:
`)
	flag.PrintDefaults()
}

func main() {
	log.SetFlags(0)
	n := flag.Uint64("n", 1_000_000, "expected number of items")
	p := flag.Float64("p", 0.01, "target false-positive rate")
	hashName := flag.String("hash", "double", "hashing strategy: double or seeded")
	load := flag.String("load", "", "file with one item per line to add first")
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() == 0 {
		usage()
		os.Exit(2)
	}

	strategy, ok := hashing.ByName(*hashName)
	if !ok {
		log.Fatalf("unknown hash %q (use double or seeded)", *hashName)
	}
	f, err := bloom.New(*n, *p, bloom.WithStrategy(strategy))
	if err != nil {
		log.Fatal(err)
	}
	if *load != "" {
		count, err := loadFile(f, *load)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("loaded %d items from %s\n", count, *load)
	}

	switch cmd := flag.Arg(0); cmd {
	case "info":
		printInfo(f, *n, *p)
	case "fp":
		runFP(f, *n, *p)
	case "compare":
		runCompare(*n, *p)
	case "check":
		if flag.NArg() < 2 {
			log.Fatal("check needs at least one NAME")
		}
		for _, name := range flag.Args()[1:] {
			fmt.Printf("%-20s %s\n", name, verdict(f.ContainsString(name)))
		}
	case "repl":
		repl(f)
	default:
		log.Fatalf("unknown command %q", cmd)
	}
}

func printInfo(f *bloom.Filter, n uint64, p float64) {
	fmt.Printf("n=%d p=%v -> m=%d bits, k=%d hashes, %.2f MB, %.2f bits/item\n",
		n, p, f.M(), f.K(), float64(f.SizeBytes())/1e6, float64(f.M())/float64(n))
}

func runFP(f *bloom.Filter, n uint64, p float64) {
	printInfo(f, n, p)
	for i := uint64(0); i < n; i++ {
		f.AddString(fmt.Sprintf("added-%d", i))
	}
	fp := 0
	for i := uint64(0); i < n; i++ {
		if f.ContainsString(fmt.Sprintf("absent-%d", i)) {
			fp++
		}
	}
	fmt.Printf("target    %.4f%%\n", p*100)
	fmt.Printf("formula   %.4f%%\n", f.EstimatedFalsePositiveRate()*100)
	fmt.Printf("measured  %.4f%%  (%d / %d)\n", float64(fp)/float64(n)*100, fp, n)
	fmt.Printf("fill      %.1f%% of bits set\n", f.FillRatio()*100)
}

func heapBytes() uint64 {
	runtime.GC()
	runtime.GC()
	var s runtime.MemStats
	runtime.ReadMemStats(&s)
	return s.HeapAlloc
}

func runCompare(n uint64, p float64) {
	keys := make([]string, n)
	for i := range keys {
		keys[i] = fmt.Sprintf("user-%d@example.com", i)
	}

	before := heapBytes()
	m := make(map[string]struct{})
	for _, k := range keys {
		m[strings.Clone(k)] = struct{}{} // the map owns its own copy, as it would in a real service
	}
	mapBytes := heapBytes() - before
	start := time.Now()
	hits := 0
	for _, k := range keys {
		if _, ok := m[k]; ok {
			hits++
		}
	}
	mapNs := float64(time.Since(start).Nanoseconds()) / float64(n)
	runtime.KeepAlive(m)

	f, _ := bloom.New(n, p)
	for _, k := range keys {
		f.AddString(k)
	}
	start = time.Now()
	for _, k := range keys {
		if f.ContainsString(k) {
			hits++
		}
	}
	bloomNs := float64(time.Since(start).Nanoseconds()) / float64(n)

	fmt.Printf("%d keys like %q\n", n, keys[0])
	fmt.Printf("map[string]struct{}  %8.2f MB  %6.1f ns/lookup  (exact, stores the keys)\n", float64(mapBytes)/1e6, mapNs)
	fmt.Printf("bloom (p=%v)       %8.2f MB  %6.1f ns/lookup  (approximate)\n", p, float64(f.SizeBytes())/1e6, bloomNs)
	fmt.Printf("memory saved         %.0fx\n", float64(mapBytes)/float64(f.SizeBytes()))
	_ = hits
}

func repl(f *bloom.Filter) {
	fmt.Println(`commands: "add NAME", "check NAME", ctrl+D to quit`)
	sc := bufio.NewScanner(os.Stdin)
	for fmt.Print("> "); sc.Scan(); fmt.Print("> ") {
		fields := strings.Fields(sc.Text())
		switch {
		case len(fields) == 0:
		case len(fields) == 2 && fields[0] == "add":
			f.AddString(fields[1])
			fmt.Println("added")
		case len(fields) == 2 && fields[0] == "check":
			fmt.Println(verdict(f.ContainsString(fields[1])))
		default:
			fmt.Println(`expected "add NAME" or "check NAME"`)
		}
	}
	fmt.Println()
}

func loadFile(f *bloom.Filter, path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	count := 0
	sc := bufio.NewScanner(file)
	for sc.Scan() {
		if s := strings.TrimSpace(sc.Text()); s != "" {
			f.AddString(s)
			count++
		}
	}
	return count, sc.Err()
}

func verdict(maybe bool) string {
	if maybe {
		return "probably taken"
	}
	return "available (definitely never added)"
}
