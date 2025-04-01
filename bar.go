package history

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// Bar
type Bar struct {
	Time   time.Time `json:"time"`
	Open   float64   `json:"open"`
	High   float64   `json:"high"`
	Low    float64   `json:"low"`
	Close  float64   `json:"close"`
	Volume float64   `json:"volume,omitempty"`
}

func (b Bar) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"time":   b.Time.Unix(),
		"open":   b.Open,
		"high":   b.High,
		"low":    b.Low,
		"close":  b.Close,
		"volume": b.Volume,
	}

	return json.Marshal(m)
}

func (b *Bar) UnmarshalJSON(data []byte) error {
	var err error
	m := make(map[string]interface{})
	if err = json.Unmarshal(data, &m); err != nil {
		return err
	}

	b.Time = time.Unix(int64(m["time"].(float64)), 0)
	if b.Open, err = strconv.ParseFloat(fmt.Sprintf("%v", m["open"]), 64); err != nil {
		return err
	}
	if b.High, err = strconv.ParseFloat(fmt.Sprintf("%v", m["high"]), 64); err != nil {
		return err
	}
	if b.Low, err = strconv.ParseFloat(fmt.Sprintf("%v", m["low"]), 64); err != nil {
		return err
	}
	if b.Close, err = strconv.ParseFloat(fmt.Sprintf("%v", m["close"]), 64); err != nil {
		return err
	}
	if b.Volume, err = strconv.ParseFloat(fmt.Sprintf("%v", m["volume"]), 64); err != nil {
		return err
	}

	return err
}

// Price Mode
func (b Bar) Mode(mode Price) float64 {
	switch mode {
	case O:
		return b.Open
	case H:
		return b.High
	case L:
		return b.Low
	case C:
		return b.Close
	case HL2:
		return b.High + b.Low/2
	case HLC3:
		return b.High + b.Low + b.Close/3
	case OHLC4:
		return b.Open + b.High + b.Low + b.Close/4
	case V:
		return b.Volume
	default:
		return 0
	}
}

// Price Mode
type Price int

const (
	O     Price = iota // O price open
	H                  // H price open
	L                  // L price high
	C                  // C price close
	HL2                // HL2 price volume
	HLC3               // HLC3 price median
	OHLC4              // OHLC4 price median
	V                  // V price volume
)

// Timeframe
type Timeframe int

const (
	m1  Timeframe = 1
	m3  Timeframe = 3
	m5  Timeframe = 5
	m15 Timeframe = 15
	m30 Timeframe = 30
	h1  Timeframe = 60
	h4  Timeframe = 240
	h6  Timeframe = 360
	h8  Timeframe = 480
	h12 Timeframe = 960
	d1  Timeframe = 1440
	d3  Timeframe = 4320
	w1  Timeframe = 10080
	M   Timeframe = 70560
)

// TFInterval formats timeframe
func TFInterval(tf string) Timeframe {
	switch strings.ToLower(tf) {
	case "1m", "1M", "m1", "M1", "1":
		return m1
	case "3m", "3M", "m3", "M3", "3":
		return m3
	case "5m", "5M", "m5", "M5", "5":
		return m5
	case "15m", "15M", "m15", "M15", "15":
		return m15
	case "30m", "30M", "m30", "M30", "30":
		return m30
	case "1h", "1H", "h1", "H1", "h", "H", "60":
		return h1
	case "4h", "4H", "h4", "H4", "240":
		return h4
	case "6h", "6H", "h6", "H6", "360":
		return h6
	case "8h", "8H", "h8", "H8", "480":
		return h8
	case "12h", "12H", "h12", "H12", "960":
		return h12
	case "1d", "1D", "d1", "D1", "d", "D", "1440":
		return d1
	case "3d", "3D", "d3", "D3", "4320":
		return d3
	case "1w", "1W", "w1", "W1", "w", "W", "10080":
		return w1
	case "m", "M", "70560":
		return M
	default:
		return 0
	}
}

// TFString formats timeframe
func TFString(tf Timeframe) string {
	switch tf {
	case m1:
		return "1m"
	case m3:
		return "3m"
	case m5:
		return "5m"
	case m15:
		return "15m"
	case m30:
		return "30m"
	case h1:
		return "1h"
	case h4:
		return "4h"
	case h6:
		return "6h"
	case h8:
		return "8h"
	case h12:
		return "12h"
	case d1:
		return "1d"
	case d3:
		return "3d"
	case w1:
		return "1w"
	case M:
		return "M"
	default:
		return ""
	}
}

// Is timeframe string valid
func TFIsValid(tf string) bool {
	return TFString(TFInterval(tf)) != ""
}

// T returns bar time
func (b Bar) T() time.Time {
	return b.Time
}

// O returns bar open price
func (b Bar) O() float64 {
	return b.Open
}

// H returns bar high price
func (b Bar) H() float64 {
	return b.High
}

// L returns bar low price
func (b Bar) L() float64 {
	return b.Low
}

// C returns bar close price
func (b Bar) C() float64 {
	return b.Close
}

// HL2 returns bar 'mid' price
func (b Bar) HL2() float64 {
	return (b.High + b.Low) / 2
}

// HLC3 returns bar (high+low+close)/3 price
func (b Bar) HLC3() float64 {
	return (b.High + b.Low + b.Close) / 3
}

// OHLC3 reurns bar (open+high+low+close)/4 price
func (b Bar) OHLC4() float64 {
	return (b.Open + b.High + b.Low + b.Close) / 4
}

// V returns bar volume
func (b Bar) V() float64 {
	return b.Volume
}

// Range retuens bar rage (high-low)
func (b Bar) Range() float64 {
	return b.High - b.Low
}

// Body
func (b Bar) Body() float64 {
	return math.Max(b.Open, b.Close) - math.Min(b.Open, b.Close)
}

// BodyHigh
func (b Bar) BodyHigh() float64 {
	return math.Max(b.Open, b.Close)
}

// BodyLow
func (b Bar) BodyLow() float64 {
	return math.Min(b.Open, b.Close)
}

// Bull
func (b Bar) Bull() bool {
	return b.Close > b.Open
}

// Bear
func (b Bar) Bear() bool {
	return b.Open > b.Close
}

// Bullish that  closes upper 33%
func (b Bar) Bullish() bool {
	return b.Close >= (b.High - b.Range()/3)
}

// Bearish that  closes bottom 33%
func (b Bar) Bearish() bool {
	return b.Close <= (b.Low + b.Range()/3)
}

// WickUp
func (b Bar) WickUp() float64 {
	return b.High - b.BodyHigh()
}

// WickDn
func (b Bar) WickDn() float64 {
	return b.BodyLow() - b.Low
}

// PercMove
func (b Bar) PercMove() float64 {
	return 100 * ((b.Close - b.Open) / b.Open)
}
