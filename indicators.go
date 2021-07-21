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

// ATR ..
func (bars Bars) ATR() float64 {
	var sum float64

	for _, b := range bars {
		sum += b.Range()
	}

	return sum / float64(len(bars))
}

// StDev ..
func (bars Bars) StDev(mode PriceMode) float64 {
	var v float64
	sma := bars.SMA(mode)

	for i := range bars {
		v += math.Pow(bars[i:i+len(bars)].SMA(mode)-sma, 2)
		// v += math.Pow(bars.SMA(O)-sma, 2)
	}

	return math.Sqrt(v / 20)
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

// LastBullIdx ..
func (bars Bars) LastBullIdx() int {
	for i, b := range bars {
		if b.Bull() {
			return i
		}
	}

	return -1
}

// LastBearIdx ..
func (bars Bars) LastBearIdx() int {
	for i, b := range bars {
		if b.Bear() {
			return i
		}
	}

	return -1
}

// ClearStates ..
type ClearStates struct {
	hh, ll, hl, lh, price float64
	state                 int
}

// ClearState ..
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

// // FractalHighIdx ..
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

// // FractalLowIdx ..
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

// TD Sequential 9
func (bars Bars) TD() int {
	var upCount []int = make([]int, len(bars))
	var dc []int = make([]int, len(bars))
	for i := len(bars) - 5; i >= 0; i-- {

		isUp := bars[i].Close > bars[i+4].Close
		isDn := bars[i].Close < bars[i+4].Close

		// UPCOUNT
		if isUp {
			dc[i] = 0
			if upCount[i+1] < 9 {
				upCount[i] = upCount[i+1] + 1
			} else {
				upCount[i] = 0
			}

			// PERFECT BUY
			if upCount[i] == 9 {
				if bars[i+1].Low <= bars[i+3].Low || bars[i].Low <= bars[i+2].Low {
					upCount[i] = 10
				}
			}
		}

		// DOWNCOUNT
		if isDn {
			upCount[i] = 0
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

		// debug: 	fmt.Printf("isUp\t%v \tupCount\t%d \t\tisDn\t%v \t dc\t%d\t \n", isUp, upCount[i], isDn, dc[i])
	}
	if upCount[0] == 9 {
		return 1
	}
	if upCount[0] == 10 { // PERFECT SELL
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

// HtTrendline - Hilbert Transform - Instantaneous Trendline (lookback=63)
func HtTrendline(data []float64) []float64 {

	outReal := make([]float64, len(data))
	a := 0.0962
	b := 0.5769
	detrenderOdd := make([]float64, 3)
	detrenderEven := make([]float64, 3)
	q1Odd := make([]float64, 3)
	q1Even := make([]float64, 3)
	jIOdd := make([]float64, 3)
	jIEven := make([]float64, 3)
	jQOdd := make([]float64, 3)
	jQEven := make([]float64, 3)
	smoothPriceIdx := 0
	maxIdxSmoothPrice := (50 - 1)
	smoothPrice := make([]float64, maxIdxSmoothPrice+1)
	iTrend1 := 0.0
	iTrend2 := 0.0
	iTrend3 := 0.0
	tempReal := math.Atan(1)
	rad2Deg := 45.0 / tempReal
	lookbackTotal := 63
	startIdx := lookbackTotal
	trailingWMAIdx := startIdx - lookbackTotal
	today := trailingWMAIdx
	tempReal = data[today]
	today++
	periodWMASub := tempReal
	periodWMASum := tempReal
	tempReal = data[today]
	today++
	periodWMASub += tempReal
	periodWMASum += tempReal * 2.0
	tempReal = data[today]
	today++
	periodWMASub += tempReal
	periodWMASum += tempReal * 3.0
	trailingWMAValue := 0.0
	i := 34
	for ok := true; ok; {
		tempReal = data[today]
		today++
		periodWMASub += tempReal
		periodWMASub -= trailingWMAValue
		periodWMASum += tempReal * 4.0
		trailingWMAValue = data[trailingWMAIdx]
		trailingWMAIdx++
		//smoothedValue := periodWMASum * 0.1
		periodWMASum -= periodWMASub
		i--
		ok = i != 0
	}
	hilbertIdx := 0
	detrender := 0.0
	prevDetrenderOdd := 0.0
	prevDetrenderEven := 0.0
	prevDetrenderInputOdd := 0.0
	prevDetrenderInputEven := 0.0
	q1 := 0.0
	prevq1Odd := 0.0
	prevq1Even := 0.0
	prevq1InputOdd := 0.0
	prevq1InputEven := 0.0
	jI := 0.0
	prevJIOdd := 0.0
	prevJIEven := 0.0
	prevJIInputOdd := 0.0
	prevJIInputEven := 0.0
	jQ := 0.0
	prevJQOdd := 0.0
	prevJQEven := 0.0
	prevJQInputOdd := 0.0
	prevJQInputEven := 0.0
	period := 0.0
	outIdx := 63
	previ2 := 0.0
	prevq2 := 0.0
	Re := 0.0
	Im := 0.0
	i1ForOddPrev3 := 0.0
	i1ForEvenPrev3 := 0.0
	i1ForOddPrev2 := 0.0
	i1ForEvenPrev2 := 0.0
	smoothPeriod := 0.0
	q2 := 0.0
	i2 := 0.0
	for today < len(data) {
		adjustedPrevPeriod := (0.075 * period) + 0.54
		todayValue := data[today]
		periodWMASub += todayValue
		periodWMASub -= trailingWMAValue
		periodWMASum += todayValue * 4.0
		trailingWMAValue = data[trailingWMAIdx]
		trailingWMAIdx++
		smoothedValue := periodWMASum * 0.1
		periodWMASum -= periodWMASub
		smoothPrice[smoothPriceIdx] = smoothedValue
		if (today % 2) == 0 {
			hilbertTempReal := a * smoothedValue
			detrender = -detrenderEven[hilbertIdx]
			detrenderEven[hilbertIdx] = hilbertTempReal
			detrender += hilbertTempReal
			detrender -= prevDetrenderEven
			prevDetrenderEven = b * prevDetrenderInputEven
			detrender += prevDetrenderEven
			prevDetrenderInputEven = smoothedValue
			detrender *= adjustedPrevPeriod
			hilbertTempReal = a * detrender
			q1 = -q1Even[hilbertIdx]
			q1Even[hilbertIdx] = hilbertTempReal
			q1 += hilbertTempReal
			q1 -= prevq1Even
			prevq1Even = b * prevq1InputEven
			q1 += prevq1Even
			prevq1InputEven = detrender
			q1 *= adjustedPrevPeriod
			hilbertTempReal = a * i1ForEvenPrev3
			jI = -jIEven[hilbertIdx]
			jIEven[hilbertIdx] = hilbertTempReal
			jI += hilbertTempReal
			jI -= prevJIEven
			prevJIEven = b * prevJIInputEven
			jI += prevJIEven
			prevJIInputEven = i1ForEvenPrev3
			jI *= adjustedPrevPeriod
			hilbertTempReal = a * q1
			jQ = -jQEven[hilbertIdx]
			jQEven[hilbertIdx] = hilbertTempReal
			jQ += hilbertTempReal
			jQ -= prevJQEven
			prevJQEven = b * prevJQInputEven
			jQ += prevJQEven
			prevJQInputEven = q1
			jQ *= adjustedPrevPeriod
			hilbertIdx++
			if hilbertIdx == 3 {
				hilbertIdx = 0
			}
			q2 = (0.2 * (q1 + jI)) + (0.8 * prevq2)
			i2 = (0.2 * (i1ForEvenPrev3 - jQ)) + (0.8 * previ2)
			i1ForOddPrev3 = i1ForOddPrev2
			i1ForOddPrev2 = detrender
		} else {
			hilbertTempReal := a * smoothedValue
			detrender = -detrenderOdd[hilbertIdx]
			detrenderOdd[hilbertIdx] = hilbertTempReal
			detrender += hilbertTempReal
			detrender -= prevDetrenderOdd
			prevDetrenderOdd = b * prevDetrenderInputOdd
			detrender += prevDetrenderOdd
			prevDetrenderInputOdd = smoothedValue
			detrender *= adjustedPrevPeriod
			hilbertTempReal = a * detrender
			q1 = -q1Odd[hilbertIdx]
			q1Odd[hilbertIdx] = hilbertTempReal
			q1 += hilbertTempReal
			q1 -= prevq1Odd
			prevq1Odd = b * prevq1InputOdd
			q1 += prevq1Odd
			prevq1InputOdd = detrender
			q1 *= adjustedPrevPeriod
			hilbertTempReal = a * i1ForOddPrev3
			jI = -jIOdd[hilbertIdx]
			jIOdd[hilbertIdx] = hilbertTempReal
			jI += hilbertTempReal
			jI -= prevJIOdd
			prevJIOdd = b * prevJIInputOdd
			jI += prevJIOdd
			prevJIInputOdd = i1ForOddPrev3
			jI *= adjustedPrevPeriod
			hilbertTempReal = a * q1
			jQ = -jQOdd[hilbertIdx]
			jQOdd[hilbertIdx] = hilbertTempReal
			jQ += hilbertTempReal
			jQ -= prevJQOdd
			prevJQOdd = b * prevJQInputOdd
			jQ += prevJQOdd
			prevJQInputOdd = q1
			jQ *= adjustedPrevPeriod
			q2 = (0.2 * (q1 + jI)) + (0.8 * prevq2)
			i2 = (0.2 * (i1ForOddPrev3 - jQ)) + (0.8 * previ2)
			i1ForEvenPrev3 = i1ForEvenPrev2
			i1ForEvenPrev2 = detrender
		}
		Re = (0.2 * ((i2 * previ2) + (q2 * prevq2))) + (0.8 * Re)
		Im = (0.2 * ((i2 * prevq2) - (q2 * previ2))) + (0.8 * Im)
		prevq2 = q2
		previ2 = i2
		tempReal = period
		if (Im != 0.0) && (Re != 0.0) {
			period = 360.0 / (math.Atan(Im/Re) * rad2Deg)
		}
		tempReal2 := 1.5 * tempReal
		if period > tempReal2 {
			period = tempReal2
		}
		tempReal2 = 0.67 * tempReal
		if period < tempReal2 {
			period = tempReal2
		}
		if period < 6 {
			period = 6
		} else if period > 50 {
			period = 50
		}
		period = (0.2 * period) + (0.8 * tempReal)
		smoothPeriod = (0.33 * period) + (0.67 * smoothPeriod)
		DCPeriod := smoothPeriod + 0.5
		DCPeriodInt := math.Floor(DCPeriod)
		idx := today
		tempReal = 0.0
		for i := 0; i < int(DCPeriodInt); i++ {
			tempReal += data[idx]
			idx--
		}
		if DCPeriodInt > 0 {
			tempReal = tempReal / (DCPeriodInt * 1.0)
		}
		tempReal2 = (4.0*tempReal + 3.0*iTrend1 + 2.0*iTrend2 + iTrend3) / 10.0
		iTrend3 = iTrend2
		iTrend2 = iTrend1
		iTrend1 = tempReal
		if today >= startIdx {
			outReal[outIdx] = tempReal2
			outIdx++
		}
		smoothPriceIdx++
		if smoothPriceIdx > maxIdxSmoothPrice {
			smoothPriceIdx = 0
		}

		today++
	}
	return outReal
}
