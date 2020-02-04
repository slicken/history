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

// IsPinBuy ...
func (bars Bars) IsPinBuy() bool {
	o0 := bars[0].Open
	o2 := bars[2].Open
	c0 := bars[0].Close
	c2 := bars[2].Close

	if bars[2].Body() > bars[1].Body() && bars[0].Body() > bars[1].Body() &&
		c2 < o2 && c0 > o0 && c0 > bars[1].BodyLow()+bars[1].Body()*0.5 {
		return true
	}
	return false
}

// IsPinSell ...
func (bars Bars) IsPinSell() bool {
	o0 := bars[0].Open
	o2 := bars[2].Open
	c0 := bars[0].Close
	c2 := bars[2].Close

	if bars[2].Body() > bars[1].Body() && bars[0].Body() > bars[1].Body() &&
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

// TD Sequential 9
func (bars Bars) TD() int {

	//	uc=isUp==1?nz(uc[1])==0?1:uc[1]==1?2:uc[1]==2?3:uc[1]==3?4:uc[1]==4?5:uc[1]==5?6:uc[1]==6?7:uc[1]==7?8:uc[1]==8?9:0:0
	//	dc=isDn==1?nz(dc[1])==0?1:dc[1]==1?2:dc[1]==2?3:dc[1]==3?4:dc[1]==4?5:dc[1]==5?6:dc[1]==6?7:dc[1]==7?8:dc[1]==8?9:0:0

	var uc []int = make([]int, len(bars))
	var dc []int = make([]int, len(bars))
	for i := len(bars) - 5; i >= 0; i-- {

		isUp := bars[i].Close > bars[i+4].Close
		isDn := bars[i].Close < bars[i+4].Close

		// UPCOUNT
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
			} /* else if uc[i] != 0 && bars[i].Close >= bars[i+4].Close {
				uc[i] = 0
			}
			*/
		}

		// DOWNCOUNT
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
			} /* else if dc[i] != 0 && bars[i].Close <= bars[i+4].Close {
				dc[i] = 0
			}
			*/
		}

		// debug: 	fmt.Printf("isUp\t%v \tuc\t%d \t\tisDn\t%v \t dc\t%d\t \n", isUp, uc[i], isDn, dc[i])
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

/*
func (bars Bars) SEQ() int {

	var bSetup, sSetup, bCountdown, sCountdown int
	var bSetupInd, sSetupInd bool

	for i := 100; i >= 0; i-- {

		//+------------------------------------------------------------------+
		//| Buy Setup                                                        |
		//+------------------------------------------------------------------+

		if bars[i].Close <= bars[i+4].Close && bars[i+1].Close >= bars[i+5].Close && bSetup == 0 {
			bSetup++
		}

		if bars[i].Close < bars[i+4].Close && bSetup != 0 {
			bSetup++

			if bSetup == 9 {
				bSetup = 0
				bSetupInd = true
				sSetupInd = false
				sCountdown = 0
			}
		}

		//if setup is completed look for criteria for perfect setup
		if bSetupInd {
			if bars[i+1].Low <= bars[i+3].Low || bars[i].Low <= bars[i+2].Low {
				bSetupInd = false
				//bPerfect = true
				bCountdown = 1
			}
		} else if bars[i].Close >= bars[i+4].Close && bSetup != 0 {
			bSetup = 0
		}

		//+------------------------------------------------------------------+
		//| Buy Countdown Setup                                              |
		//+------------------------------------------------------------------+

		if bCountdown == 13 && bars[i].Close <= bars[i+1].Close {
			bCountdown = 0
		}

		if bCountdown >= 1 && bCountdown < 13 && bars[i].Close <= bars[i+2].Close {
			bCountdown++
		}

		//+------------------------------------------------------------------+
		//| Sell Setup                                                       |
		//+------------------------------------------------------------------+

		if bars[i].Close >= bars[i+4].Close && bars[i+1].Close <= bars[i+5].Close && bSetup == 0 {
			sSetup++
		}

		if bars[i].Close >= bars[i+4].Close && sSetup != 0 {
			sSetup++
			if sSetup == 9 {
				sSetup = 0
				sSetupInd = true
				bSetupInd = false
				bCountdown = 0
			}
		}

		//Perfected Setup

		if sSetupInd {
			if bars[i+1].Low >= bars[i+3].Low || bars[i].Low >= bars[i+2].Low {
				sSetupInd = false
				//sPerfect = true
				sCountdown = 1
			}
		} else if bars[i].Close <= bars[i+4].Close && sSetup != 0 {
			sSetup = 0
		}

		//+------------------------------------------------------------------+
		//| Sell Countdown Setup                                             |
		//+------------------------------------------------------------------+

		if sCountdown == 13 && bars[i].Close >= bars[i+2].Close {
			sCountdown = 0
		}

		if sCountdown >= 1 && sCountdown < 13 && bars[i].Close >= bars[i+2].Close {
			sCountdown++
		}
	}
	return 0
}
*/
