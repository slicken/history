package history

import (
	"math"
)

// SMA on bars
func (bars Bars) SMA(mode PriceMode) float64 {
	var sum float64

	for _, b := range bars {
		sum += b.Price(mode)
	}

	return sum / float64(len(bars))
}

// LWMA on bars
func (bars Bars) LWMA(mode PriceMode) float64 {
	var period = len(bars)
	var sum, weight float64

	for i := period - 1; i >= 0; i-- {
		weight += float64(period - i)
		sum += bars[i].Price(mode) * float64(period-i)
	}

	if weight > 0 {
		return sum / weight
	}

	return -1.
}

// EMA on bars
func (bars Bars) EMA(mode PriceMode) float64 {
	period := len(bars)
	var last, k, sum float64

	k = 2 / (float64(period) + 1)
	sum = bars.SMA(mode)
	for i := period - 1; i >= 0; i-- {
		last = sum
		sum = bars[i].Price(mode)*k + last*(1-k)
	}

	return sum
}

// VolumeAvg ..
func (bars Bars) VolumeAvg() float64 {
	var sum float64 = 1
	for _, b := range bars {
		sum += b.Volume
	}

	return sum / float64(len(bars))
}

// ATR ..
func (bars Bars) ATR(mode PriceMode) float64 {
	var sum float64

	for _, b := range bars {
		sum += b.Range()
	}

	return sum / float64(len(bars))
}

// Range ..
func (bars Bars) Range() float64 {
	return bars.Highest(H) - bars.Lowest(L)
}

// Highest ..
func (bars Bars) Highest(mode PriceMode) float64 {
	highest := -1.
	for _, b := range bars {
		if highest == -1. || b.Price(mode) > highest {
			highest = b.Price(mode)
		}
	}

	return highest
}

// HighestIdx ..
func (bars Bars) HighestIdx(mode PriceMode) int {
	index := -1
	highest := -1.
	for i, b := range bars {
		if highest == -1. || b.Price(mode) > highest {
			highest = b.Price(mode)
			index = i
		}
	}

	return index
}

// Lowest ..
func (bars Bars) Lowest(mode PriceMode) float64 {
	lowest := -1.
	for _, b := range bars {
		if lowest == -1. || b.Price(mode) < lowest {
			lowest = b.Price(mode)
		}
	}

	return lowest
}

// LowestIdx ..
func (bars Bars) LowestIdx(mode PriceMode) int {
	index := -1
	lowest := -1.
	for i, b := range bars {
		if lowest == -1. || b.Price(mode) < lowest {
			lowest = b.Price(mode)
			index = i
		}
	}

	return index
}

// fmt.Println("price", bars[0].Price(C))
// fmt.Println("sma", bars[0:30].SMA(C))
// fmt.Println("ema", bars[0:30].EMA(C))
// fmt.Println("lwma", bars[0:30].LWMA(C)
// fmt.Println("atr", bars.ATR(HL))
// fmt.Println("highest", bars.Highest(o))
// fmt.Println("highestIdx", bars.HighestIdx(o))
// fmt.Println("lowest", bars.Lowest(o))
// fmt.Println("lowestIdx", bars.LowestIdx(o))

type ClearStates struct {
	hh, ll, hl, lh, price float64
	state                 int
}

func (c *ClearStates) ClearState(bars Bars) int {

	high := bars[0].High
	low := bars[0].Low
	if c.state != -1 {
		c.hh = math.Max(high, c.hh)
		c.hl = math.Max(low, c.hl)
		// down swing
		if high < c.hl {
			c.state = -1
			c.ll = low
			c.lh = high
		}
	} else if c.state != 1 {
		c.ll = math.Min(low, c.ll)
		c.lh = math.Min(high, c.lh)
		// up swing
		if low > c.lh {
			c.state = 1
			c.hh = high
			c.hl = low
		}
	}

	// set state
	if c.state == -1 {
		c.price = c.lh
	} else {
		c.price = c.hl
	}

	return c.state
}

func (bars Bars) FractalHighIdx(per int) int {
	for i, _ := range bars[:len(bars)-per] {
		sh := i - per
		r := (per * 2) + 1
		if bars[sh:r].HighestIdx(H) == i {
			return i
		}
	}
	return -1
}

func (bars Bars) FractalLowIdx(per int) int {
	for i, _ := range bars[:len(bars)-per] {
		sh := i - per
		r := (per * 2) + 1
		if bars[sh:r].LowestIdx(L) == i {
			return i
		}
	}
	return -1
}

func (bars Bars) IsPinBuy() bool {

	if bars[0].Bull() && bars[0].Low < bars[bars[1:4].LowestIdx(L)].BodyLow() && bars[0].BodyLow()-bars[0].Low > bars[0].Body()*2 {
		return true
	}

	// o0 := bars[0].Open
	// o2 := bars[2].Open
	// c0 := bars[0].Close
	// c2 := bars[2].Close

	// if bars[2].Body() > bars[1].Body() && bars[0].Body() > bars[1].Body() &&
	// 	c2 < o2 && c0 > o0 && c0 > bars[1].BodyLow()+bars[1].Body()*0.5 {
	// 	return true
	// }
	return false
}

func (bars Bars) IsPinSell() bool {
	o0 := bars[0].Open
	o2 := bars[2].Open
	c0 := bars[0].Close
	c2 := bars[2].Close

	if bars[1].Body() > bars[1].Body() && bars[0].Body() > bars[1].Body() &&
		c2 > o2 && c0 < o0 && c0 < bars[1].BodyHigh()-bars[1].Body()*0.5 {
		return true
	}
	return false
}

// IsEngulfBuy ..
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

// IsEngulfSell ..
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
