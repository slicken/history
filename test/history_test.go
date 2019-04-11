package main

import (
	"fmt"
	"testing"
	"time"
)

//go test -timeout 30s github.com/slicken/hc -run ^(TestData)$
var (
	sym = "BTCUSDT"
	tf  = "1d"

	bars, _ = loadBars(sym, tf)
	new, _  = download(sym, tf, 5000)
)

func TestTime(t *testing.T) {
	nowUnix := toUnix(time.Now())
	fmt.Printf("%d\t %v\n", nowUnix, toTime(nowUnix))

	barUnix := bars[len(bars)-1].Time
	fmt.Printf("%d\t %v\n", barUnix, toTime(barUnix))

}

func TestDataset(t *testing.T) {
	if len(bars) == 0 {
		t.Fatalf("no bars")
	}

	// if !isReverse(bars) {
	// 	reverse(bars)
	// }

	for i, b := range bars[:10] {
		fmt.Printf("%d\t %v\n", i, b.DateTime())

		// stream = append(stream, bars[i])

	}

	// v := time.Unix(time.Now()/1000, 0).UTC()

	// bars = append(bars, Bar{Time: v})
	// fmt.Printf("%d\t %v\n", -1, bars[0].DateTime())

}

func TestMerge1(t *testing.T) {
	bt := bars[len(bars)-1].Time
	fmt.Printf("bars %d\t %v\n", len(bars), toTime(bt))

	nt := new[len(new)-1].Time
	fmt.Printf("new %d\t %v\n", len(new), toTime(nt))

	merged := merge(bars, new)
	fmt.Printf("merged %d\n", len(merged))
}

func TestHandler(t *testing.T) {
	s := New()
	s.UpdateBars("BTCUSDT", "1d", new)

	for tf := range s.Hist {
		last := bars[0].DateTime()
		if 0 > time.Until(last.Add(time.Duration(tf)*time.Minute)) {

			// update
		}
	}
}

// ----------------------------------------

// reverse data
func reverse(bars []Bar) {
	for i := len(bars)/2 - 1; i >= 0; i-- {
		v := len(bars) - 1 - i
		bars[i], bars[v] = bars[v], bars[i]
	}
}

// reverse data
func reverse2(bars []Bar) {
	for left, right := 0, len(bars)-1; left < right; left, right = left+1, right-1 {
		bars[left], bars[right] = bars[right], bars[left]
	}
}

func isReverse(bars []Bar) bool {
	return bars[0].Time > bars[len(bars)-1].Time
}

// Benchmarks -----------------------------

func BenchmarkReverse(b *testing.B) {
	for n := 0; n < b.N; n++ {
		reverse(bars)
	}
}

func BenchmarkReverse2(b *testing.B) {
	for n := 0; n < b.N; n++ {
		reverse2(bars)
	}
}

// Benchmarks -----------------------------

func BenchmarkMerge1(b *testing.B) {
	for n := 0; n < b.N; n++ {
		_ = merge(bars, new)
	}
}
