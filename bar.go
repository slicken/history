package history

import (
	"math"
	"sync"
	"time"
)

// Bar ..
type Bar struct {
	// event
	Time   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume float64
}

// Price ..
func (b Bar) Price(mode PriceMode) float64 {
	switch mode {
	case O:
		return b.Open
	case H:
		return b.High
	case L:
		return b.Low
	case C:
		return b.Close
	case M:
		return b.Open + b.High + b.Low + b.Close/4
	case HL:
		return b.High + b.Low/2
	case V:
		return b.Volume
	default:
		return 0
	}
}

// PriceMode ..
type PriceMode int

const (
	// O price open
	O PriceMode = iota
	// H price open
	H
	// L price high
	L
	// C price close
	C
	// M price median
	M
	// HL price volume
	HL
	// V price volume
	V
)

// // SetEvent ..
// func (b *Bar) SetEvent(price float64, info string) {
// 	b.event.price = price
// 	b.event.info = info
// }

// // IsEvent ..
// func (b *Bar) IsEvent() bool {
// 	if b.event.price == 0 {
// 		return false
// 	}
// 	return true
// }

// Timeframe ..
type Timeframe int

const (
	m5  Timeframe = 5
	m15 Timeframe = 15
	h1  Timeframe = 60
	h4  Timeframe = 240
	d1  Timeframe = 1440
	w1  Timeframe = 5760
)

// Tf formats timeframe
func Tf(tf string) Timeframe {
	switch tf {
	case "5m", "5M", "m5", "M5", "5":
		return m5
	case "15m", "15M", "m15", "M15", "15":
		return m15
	case "1h", "1H", "h1", "H1", "h", "H", "60":
		return h1
	case "4h", "4H", "h4", "H4", "240":
		return h4
	case "1d", "1D", "d1", "D1", "d", "D", "1440":
		return d1
	case "1w", "1W", "w1", "W1", "w", "W":
		return w1
	default:
		return 0
	}
}

// Tfs formats timeframe
func Tfs(tf Timeframe) string {
	switch tf {
	case m5:
		return "5m"
	case m15:
		return "15m"
	case h1:
		return "1h"
	case h4:
		return "4h"
	case d1:
		return "1d"
	case w1:
		return "1w"
	default:
		return ""
	}
}

func toUnix(t time.Time) int64 {
	return t.UnixNano() / 1e6
}

// merge bars itterate function
func merge(bars ...*Bars) *Bars {
	c := make(chan Bar)

	go func() {
		var wg sync.WaitGroup
		for _, li := range bars {

			wg.Add(1)
			go func(l *Bars) {
				defer wg.Done()

				for _, b := range *l {
					c <- b
				}
			}(li)
		}

		wg.Wait()
		close(c)
	}()

	// make merged
	var f = make(map[time.Time]bool, 0)
	var merged = make(Bars, 0)

	for b := range c {
		if _, ok := f[b.Time]; !ok {
			merged = append(merged, b)
		}
		f[b.Time] = true
	}

	// sort it
	merged = merged.Sort()

	return &merged
}

// Range ..
func (b Bar) Range() float64 {
	return b.High - b.Low
}

// Body ..
func (b Bar) Body() float64 {
	return math.Max(b.Open, b.Close) - math.Min(b.Open, b.Close)
}

// BodyHigh ..
func (b Bar) BodyHigh() float64 {
	return math.Max(b.Open, b.Close)
}

// BodyLow ..
func (b Bar) BodyLow() float64 {
	return math.Min(b.Open, b.Close)
}

// Bull ..
func (b Bar) Bull() bool {
	return b.Close > b.Open
}

// Bear ..
func (b Bar) Bear() bool {
	return b.Open > b.Close
}
