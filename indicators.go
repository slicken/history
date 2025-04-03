package history

import (
	"math"
)

// SMA on bars
func (bars Bars) SMA(mode Price) float64 {
	var sum float64

	for _, b := range bars {
		sum += b.Mode(mode)
	}

	return sum / float64(len(bars))
}

// LWMA on bars
func (bars Bars) LWMA(mode Price) float64 {
	var period = len(bars)
	var sum, weight float64

	for i := period - 1; i >= 0; i-- {
		weight += float64(period - i)
		sum += bars[i].Mode(mode) * float64(period-i)
	}

	if weight > 0 {
		return sum / weight
	}

	return -1.
}

// EMA on bars
func (bars Bars) EMA(mode Price) float64 {
	period := len(bars)
	var last, k, sum float64

	k = 2 / (float64(period) + 1)
	sum = bars.SMA(mode)
	for i := period - 1; i >= 0; i-- {
		last = sum
		sum = bars[i].Mode(mode)*k + last*(1-k)
	}

	return sum
}

// ATR ..
func (bars Bars) ATR() float64 {
	var sum float64

	for _, b := range bars {
		sum += b.Range()
	}

	return sum / float64(len(bars))
}

// Standard Deviation
func (bars Bars) StDev(mode Price) float64 {
	if len(bars) == 0 {
		return 0
	}
	var sum float64
	sma := bars.SMA(mode)

	for _, bar := range bars {
		diff := bar.Mode(mode) - sma
		sum += diff * diff
	}

	return math.Sqrt(sum / float64(len(bars)))
}

// Range ..
func (bars Bars) Range() float64 {
	return bars.Highest(H) - bars.Lowest(L)
}

// Highest ..
func (bars Bars) Highest(mode Price) float64 {
	highest := -1.
	for _, b := range bars {
		if highest == -1. || b.Mode(mode) > highest {
			highest = b.Mode(mode)
		}
	}

	return highest
}

// HighestIdx ..
func (bars Bars) HighestIdx(mode Price) int {
	index := -1
	highest := -1.
	for i, b := range bars {
		if highest == -1. || b.Mode(mode) > highest {
			highest = b.Mode(mode)
			index = i
		}
	}

	return index
}

// Lowest ..
func (bars Bars) Lowest(mode Price) float64 {
	lowest := -1.
	for _, b := range bars {
		if lowest == -1. || b.Mode(mode) < lowest {
			lowest = b.Mode(mode)
		}
	}

	return lowest
}

// LowestIdx ..
func (bars Bars) LowestIdx(mode Price) int {
	index := -1
	lowest := -1.
	for i, b := range bars {
		if lowest == -1. || b.Mode(mode) < lowest {
			lowest = b.Mode(mode)
			index = i
		}
	}

	return index
}

// LastBullIdx ..
func (bars Bars) LastBullIdx() int {
	for i, b := range bars {
		if b.Bull() {
			return i
		}
	}

	return -1
}

// LastBearIdx
func (bars Bars) LastBearIdx() int {
	for i, b := range bars {
		if b.Bear() {
			return i
		}
	}

	return -1
}

// WithinRange
func WithinRange(src, dest, r float64) bool {
	return math.Abs(src-dest) < r
}

// CalcuPercentage
func CalcPercentage(n, total float64) float64 {
	return 100 * (n / total)
}

// IsEngulfBuy
func (bars Bars) IsEngulfBuy() bool {
	o0 := bars[0].Open
	o1 := bars[1].Open
	c0 := bars[0].Close
	c1 := bars[1].Close
	h1 := bars[1].BodyHigh()

	if c1 < o1 && c0 > o0 && bars[1].Body() < bars[0].Body() && c0 > h1 {
		return true
	}
	return false
}

// IsEngulfSell
func (bars Bars) IsEngulfSell() bool {
	o0 := bars[0].Open
	o1 := bars[1].Open
	c0 := bars[0].Close
	c1 := bars[1].Close
	l1 := bars[1].BodyLow()

	if c1 > o1 && c0 < o0 && bars[1].Body() < bars[0].Body() && c0 < l1 {
		return true
	}
	return false
}

// TD Sequential
func (bars Bars) TDSequential() int {
	var uc []int = make([]int, len(bars))
	var dc []int = make([]int, len(bars))
	for i := len(bars) - 5; i >= 0; i-- {

		isUp := bars[i].Close > bars[i+4].Close
		isDn := bars[i].Close < bars[i+4].Close

		// UP COUNT
		if isUp {
			dc[i] = 0
			if uc[i+1] < 9 {
				uc[i] = uc[i+1] + 1
			} else {
				uc[i] = 0
			}

			// PERFECT BUY
			if uc[i] == 9 {
				if bars[i+1].Low <= bars[i+3].Low || bars[i].Low <= bars[i+2].Low {
					uc[i] = 10
				}
			}
		}

		// DOWN COUNT
		if isDn {
			uc[i] = 0
			if dc[i+1] < 9 {
				dc[i] = dc[i+1] + 1
			} else {
				dc[i] = 0
			}

			// PERFECT SELL
			if dc[i] == 9 {
				if bars[i+1].Low >= bars[i+3].Low || bars[i].Low >= bars[i+2].Low {
					dc[i] = 10
				}
			}
		}
	}
	if uc[0] == 9 {
		return 1
	}
	if uc[0] == 10 { // PERFECT SELL
		return 2
	}
	if dc[0] == 9 {
		return -1
	}
	if dc[0] == 10 { // PERFECT BUY
		return -2
	}

	return 0
}

// RSI calculates the Relative Strength Index for the given period
func (bars Bars) RSI(period int) float64 {
	if len(bars) < period+1 {
		return 0
	}

	var gains, losses float64
	for i := 1; i <= period; i++ {
		change := bars[i-1].Close - bars[i].Close
		if change > 0 {
			gains += change
		} else {
			losses -= change
		}
	}

	if losses == 0 {
		return 100
	}

	rs := gains / losses
	return 100 - (100 / (1 + rs))
}

// Stochastic calculates the Stochastic oscillator
// Returns %K (fast stochastic) and %D (slow stochastic)
func (bars Bars) Stochastic(period int) (k, d float64) {
	if len(bars) < period {
		return 0, 0
	}

	// Calculate %K
	currentClose := bars[0].Close
	lowestLow := bars[0].Low
	highestHigh := bars[0].High

	for i := 0; i < period; i++ {
		if bars[i].Low < lowestLow {
			lowestLow = bars[i].Low
		}
		if bars[i].High > highestHigh {
			highestHigh = bars[i].High
		}
	}

	if highestHigh-lowestLow == 0 {
		k = 0
	} else {
		k = 100 * ((currentClose - lowestLow) / (highestHigh - lowestLow))
	}

	// Calculate %D (3-period SMA of %K)
	if len(bars) < period+2 {
		return k, k
	}

	k2, _ := bars[1:].Stochastic(period)
	k3, _ := bars[2:].Stochastic(period)
	d = (k + k2 + k3) / 3

	return k, d
}

// IsPinbarBuy checks if the bar is a bullish pinbar
func (bars Bars) IsPinbarBuy() bool {
	if len(bars) < 1 {
		return false
	}

	bar := bars[0]

	// Check for long lower wick
	wickRatio := bar.WickDn() / bar.Body()
	upperWickRatio := bar.WickUp() / bar.Body()

	// Conditions for bullish pinbar:
	// 1. Lower wick should be at least 2x the body
	// 2. Upper wick should be small (less than body)
	// 3. Close should be in the upper third of the range
	return wickRatio >= 2.0 &&
		upperWickRatio < 1.0 &&
		bar.Bullish()
}

// IsPinbarSell checks if the bar is a bearish pinbar
func (bars Bars) IsPinbarSell() bool {
	if len(bars) < 1 {
		return false
	}

	bar := bars[0]

	// Check for long upper wick
	wickRatio := bar.WickUp() / bar.Body()
	lowerWickRatio := bar.WickDn() / bar.Body()

	// Conditions for bearish pinbar:
	// 1. Upper wick should be at least 2x the body
	// 2. Lower wick should be small (less than body)
	// 3. Close should be in the lower third of the range
	return wickRatio >= 2.0 &&
		lowerWickRatio < 1.0 &&
		bar.Bearish()
}
