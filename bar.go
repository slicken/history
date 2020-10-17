package history

import (
	"math"
	"sync"
	"time"
)

// Bar ..
type Bar struct {
	Time   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume float64
}

// // MarshalJSON compatible with json.Marshaler interface
// func (b *Bar) MarshalJSON() ([]byte, error) {
// 	return json.Marshal(
// 		&struct {
// 			Time   time.Time `json:"time"`
// 			Open   float64   `json:"open"`
// 			High   float64   `json:"high"`
// 			Low    float64   `json:"low"`
// 			Close  float64   `json:"close"`
// 			Volume float64   `json:"volume"`
// 		}{
// 			Time:   b.Time,
// 			Open:   b.Open,
// 			High:   b.High,
// 			Low:    b.Low,
// 			Close:  b.Close,
// 			Volume: b.Volume,
// 		},
// 	)
// }

// // UnmarshalJSON compatible with json.Unmarshaler interface
// func (b *Bar) UnmarshalJSON(data []byte) error {
// 	obj := struct {
// 		Time   time.Time `json:"time"`
// 		Open   float64   `json:"open"`
// 		High   float64   `json:"high"`
// 		Low    float64   `json:"low"`
// 		Close  float64   `json:"close"`
// 		Volume float64   `json:"volume"`
// 	}{}

// 	if err := json.Unmarshal(data, &obj); err != nil {
// 		return err
// 	}

// 	b.Time = obj.Time
// 	b.Open = obj.Open
// 	b.High = obj.High
// 	b.Low = obj.Low
// 	b.Close = obj.Close
// 	b.Volume = obj.Volume

// 	return nil
// }

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

// Timeframe ..
type Timeframe int

const (
	m1  Timeframe = 1
	m5  Timeframe = 5
	m15 Timeframe = 15
	h1  Timeframe = 60
	h4  Timeframe = 240
	h12 Timeframe = 960
	d1  Timeframe = 1440
	w1  Timeframe = 10080
)

// Tf formats timeframe
func Tf(tf string) Timeframe {
	switch tf {
	case "1m", "1M", "m1", "M1", "1":
		return m1
	case "5m", "5M", "m5", "M5", "5":
		return m5
	case "15m", "15M", "m15", "M15", "15":
		return m15
	case "1h", "1H", "h1", "H1", "h", "H", "60":
		return h1
	case "4h", "4H", "h4", "H4", "240":
		return h4
	case "12h", "12H", "h12", "H12", "960":
		return h12
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
	case m1:
		return "1m"
	case m5:
		return "5m"
	case m15:
		return "15m"
	case h1:
		return "1h"
	case h4:
		return "4h"
	case h12:
		return "12h"
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

// T ..
func (b Bar) T() time.Time {
	return b.Time
}

// O ..
func (b Bar) O() float64 {
	return b.Open
}

// H ..
func (b Bar) H() float64 {
	return b.High
}

// L ..
func (b Bar) L() float64 {
	return b.Low
}

// C ..
func (b Bar) C() float64 {
	return b.Close
}

// V ..
func (b Bar) V() float64 {
	return b.Volume
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

// Bullish that  closes upper 25%
func (b Bar) Bullish() bool {
	return b.Close > (b.High - b.Range()/5)
}

// Bearish that  closes bottom 25%
func (b Bar) Bearish() bool {
	return b.Close < (b.Low + b.Range()/5)
}

// WickUp ..
func (b Bar) WickUp() float64 {
	return b.High - b.BodyHigh()
}

// WickDn ..
func (b Bar) WickDn() float64 {
	return b.BodyLow() - b.Low
}

// PercMove ..
func (b Bar) PercMove() float64 {
	return 100 * ((b.Close - b.Open) / b.Open)
}
