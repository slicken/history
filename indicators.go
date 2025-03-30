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

// TD Sequential 9
func (bars Bars) TD() int {
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

// // FractalHighIdx
// func (bars Bars) FractalHighIdx(per int) int {
// 	for i, _ := range bars[:len(bars)-per] {
// 		sh := i - per
// 		r := (per * 2) + 1
// 		if bars[sh:r].HighestIdx(H) == i {
// 			return i
// 		}
// 	}
// 	return -1
// }

// // FractalLowIdx
// func (bars Bars) FractalLowIdx(per int) int {
// 	for i, _ := range bars[:len(bars)-per] {
// 		sh := i - per
// 		r := (per * 2) + 1
// 		if bars[sh:r].LowestIdx(L) == i {
// 			return i
// 		}
// 	}
// 	return -1
// }

// // IsPinBuy ...
// func (bars Bars) IsPinBuy() bool {
// 	o0 := bars[0].Open
// 	o2 := bars[2].Open
// 	c0 := bars[0].Close
// 	c2 := bars[2].Close

// 	if bars[2].Body() > bars[1].Body() && bars[0].Body() > bars[1].Body() &&
// 		c2 < o2 && c0 > o0 && c0 > bars[1].BodyLow()+bars[1].Body()*0.5 {
// 		return true
// 	}
// 	return false
// }

// // IsPinSell ...
// func (bars Bars) IsPinSell() bool {
// 	o0 := bars[0].Open
// 	o2 := bars[2].Open
// 	c0 := bars[0].Close
// 	c2 := bars[2].Close

// 	if bars[2].Body() > bars[1].Body() && bars[0].Body() > bars[1].Body() &&
// 		c2 > o2 && c0 < o0 && c0 < bars[1].BodyHigh()-bars[1].Body()*0.5 {
// 		return true
// 	}
// 	return false
// }
