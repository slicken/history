package main

import (
	"fmt"
	"strconv"

	"github.com/slicken/history"
)

/*

	------ STRATEGIES ------

*/

type Preformance struct {
	Limit      int
	LowestOpen bool
}

// Event Signals ...
func (s *Preformance) Run(symbol string, bars history.Bars) (history.Event, bool) {
	var event = history.NewEvent(symbol)

	if s.Limit+1 > len(bars) {
		return event, false
	}

	var perc float64
	if s.LowestOpen {
		perc = 100 * ((bars[0].C() - bars[0:s.Limit].Lowest(history.O)) / bars[0:s.Limit].Lowest(history.O))
	} else {
		perc = 100 * ((bars[0].C() - bars[s.Limit].O()) / bars[s.Limit].O())
	}

	event.Type = history.OTHER
	event.Name = fmt.Sprintf("%.1f", perc)
	event.Time = bars.LastBar().T()
	event.Price = perc
	return event, true
}

// Engulfing Strategy
type Engulfe struct{}

// Run implements the Strategy interface
func (s *Engulfe) Run(symbol string, bars history.Bars) (history.Event, bool) {
	var event = history.NewEvent(symbol)

	// Need enough bars for calculation
	if len(bars) < 20 {
		return event, false
	}

	// EXCLUDE SYMBOLS PRICES MATCHING PREFIX "0.000000xx"
	if price := strconv.FormatFloat(bars[0].O(), 'f', -1, 64); len(price) >= 7 {
		if price[:7] == "0.00000" {
			return event, false
		}
	}

	// Calculate indicators
	sma20 := bars[0:20].SMA(history.C)
	atr := bars[0:14].ATR()
	avgVolume := bars[1:20].SMA(history.V)

	// Buy Signal
	if bars.IsEngulfBuy() && // Engulfing pattern
		bars[0].C() > sma20 && // Price above SMA20
		bars[0].Range() > atr*0.5 && // Significant move
		bars[0].Volume > avgVolume*1.2 { // Volume confirmation

		event.Type = history.MARKET_BUY
		event.Name = "ENGULF_BUY"
		event.Time = bars[0].T()
		event.Price = bars[0].C()
		event.Text = fmt.Sprintf("Range: %.8f", bars[0].Range())
		return event, true
	}

	// Sell Signal
	if bars.IsEngulfSell() && // Engulfing pattern
		bars[0].C() < sma20 && // Price below SMA20
		bars[0].Range() > atr*0.5 && // Significant move
		bars[0].Volume > avgVolume*1.2 { // Volume confirmation

		event.Type = history.MARKET_SELL
		event.Name = "ENGULF_SELL"
		event.Time = bars[0].T()
		event.Price = bars[0].C()
		event.Text = fmt.Sprintf("Range: %.8f", bars[0].Range())
		return event, true
	}

	return event, false
}

// EngulfingN
type Engulfing struct{}

// Event EngulfingN..
func (s *Engulfing) Run(symbol string, bars history.Bars) (history.Event, bool) {
	var event = history.NewEvent(symbol)

	if 21 > len(bars) {
		return event, false
	}
	// EXCLUDE SYMBOLS PRICES MATCHING PREFEX "0.000000xx"
	if price := strconv.FormatFloat(bars[0].O(), 'f', -1, 64); len(price) >= 7 {
		if price[:7] == "0.00000" {
			return event, false
		}
	}

	// --------------
	SMA := bars[0:20].SMA(history.C)
	ATR := bars[1:4].ATR()

	// MARKET_BUY
	if bars.LastBearIdx() < 5 &&
		bars[0].C()-SMA < 2*ATR &&
		bars[0].C() > bars[bars.LastBearIdx()].H() &&
		bars[0].O() < bars[bars.LastBearIdx()].H() &&
		bars[0].Body() > ATR &&
		bars[0].O()-SMA < 2*ATR &&
		bars[0].C() > SMA {

		event.Type = history.MARKET_BUY
		event.Name = "ENGULFING"
		event.Time = bars[0].T()
		event.Price = bars[0].C()
		return event, true
	}

	// MARKET_SELL
	if bars.LastBullIdx() < 5 &&
		bars[0].O() > bars[bars.LastBullIdx()].L() &&
		bars[0].C() < bars[bars.LastBullIdx()].L() &&
		bars[0].Body() > ATR &&
		bars[0].O()-SMA < 2*ATR &&
		bars[0].C() < SMA {

		event.Type = history.MARKET_SELL
		event.Name = "ENGULFING"
		event.Time = bars[0].T()
		event.Price = bars[0].C()
		return event, true
	}

	return event, false
}

// Power implements a more responsive version of the Power strategy
type Power struct {
	MA     int
	ATRLen int
}

// NewPower creates a new instance with default settings
func NewPower() *Power {
	return &Power{
		MA:     20, // Increased from 20 to 50 for better trend confirmation
		ATRLen: 14, // Standard ATR length
	}
}

// Run implements the Strategy interface
func (s *Power) Run(symbol string, bars history.Bars) (history.Event, bool) {
	var event = history.NewEvent(symbol)

	// Need enough bars for calculation
	if len(bars) < s.MA+10 {
		return event, false
	}

	// EXCLUDE SYMBOLS PRICES MATCHING PREFIX "0.000000xx"
	if price := strconv.FormatFloat(bars[0].O(), 'f', -1, 64); len(price) >= 7 {
		if price[:7] == "0.00000" {
			return event, false
		}
	}

	// Calculate main indicators
	MA := bars[0:s.MA].SMA(history.C)
	prevMA := bars[1 : s.MA+1].SMA(history.C)
	atr := bars[0:s.ATRLen].ATR()

	// Buy Conditions
	if bars[0].Bullish() && // Current bar is bullish
		bars[0].C() > MA && // Price above MA
		prevMA < bars[0].C() && // Strong move above MA
		bars[0].C() > bars[1:10].Highest(history.H) && // Breaking 10-bar high (increased from 5)
		bars[0].Range() > atr*1.2 && // Increased move significance
		bars[0].Volume > bars[1:20].SMA(history.V)*2.0 && // Increased volume requirement
		bars[0].Body()/bars[0].Range() > 0.6 { // Strong bullish candle

		event.Type = history.MARKET_BUY
		event.Name = "POWER_BUY"
		event.Time = bars[0].T()
		event.Price = bars[0].C()
		event.Text = fmt.Sprintf("ATR: %.8f", atr)
		return event, true
	}

	// Sell Conditions
	if !bars[0].Bullish() && // Current bar is bearish
		bars[0].C() < MA && // Price below MA
		prevMA > bars[0].C() && // Strong move below MA
		bars[0].C() < bars[1:10].Lowest(history.L) && // Breaking 10-bar low (increased from 5)
		bars[0].Range() > atr*1.2 && // Increased move significance
		bars[0].Volume > bars[1:20].SMA(history.V)*2.0 && // Increased volume requirement
		bars[0].Body()/bars[0].Range() > 0.6 { // Strong bearish candle

		event.Type = history.MARKET_SELL
		event.Name = "POWER_SELL"
		event.Time = bars[0].T()
		event.Price = bars[0].C()
		event.Text = fmt.Sprintf("ATR: %.8f", atr)
		return event, true
	}

	return event, false
}

// DoubleWick strategy looks for two significant wicks within a close range
type DoubleWick struct {
	WickRatio float64 // Minimum wick to body ratio
}

// NewDoubleWick creates a new instance with default settings
func NewDoubleWick() *DoubleWick {
	return &DoubleWick{
		WickRatio: 1.5, // Wick should be 1.5x the body size
	}
}

// Run implements the Strategy interface
func (s *DoubleWick) Run(symbol string, bars history.Bars) (history.Event, bool) {
	var event = history.NewEvent(symbol)

	// Need enough bars for calculation
	if len(bars) < 20 {
		return event, false
	}

	// EXCLUDE SYMBOLS PRICES MATCHING PREFIX "0.000000xx"
	if price := strconv.FormatFloat(bars[0].O(), 'f', -1, 64); len(price) >= 7 {
		if price[:7] == "0.00000" {
			return event, false
		}
	}

	// Calculate indicators for confirmation
	sma20 := bars[0:20].SMA(history.C)
	atr := bars[0:14].ATR()

	// Check only adjacent bars
	if len(bars) <= 1 {
		return event, false
	}

	bar1 := bars[0]
	bar2 := bars[1]

	// Skip if either bar's body is too small (doji)
	if bar1.Body() < atr*0.1 || bar2.Body() < atr*0.1 {
		return event, false
	}

	// Buy Signal - Look for large upper wicks (potential reversal down)
	if bar1.WickUp() > bar1.Body()*s.WickRatio && // First bar has significant upper wick
		bar2.WickUp() > bar2.Body()*s.WickRatio && // Second bar has significant upper wick
		bar1.High > bar2.High*0.995 && bar1.High < bar2.High*1.005 && // Similar highs
		bar1.C() > sma20 && // Price above MA (overbought)
		bar1.Range() > atr*0.8 && // Decent volatility
		bar1.WickUp() > bar1.WickDn() && // Upper wick larger than lower
		bar2.WickUp() > bar2.WickDn() { // Upper wick larger than lower

		event.Type = history.MARKET_SELL
		event.Name = "DOUBLE_WICK_SELL"
		event.Time = bars[0].T()
		event.Price = bars[0].C()
		event.Text = fmt.Sprintf("Upper wicks: %.8f, %.8f", bar1.WickUp(), bar2.WickUp())
		return event, true
	}

	// Sell Signal - Look for large lower wicks (potential reversal up)
	if bar1.WickDn() > bar1.Body()*s.WickRatio && // First bar has significant lower wick
		bar2.WickDn() > bar2.Body()*s.WickRatio && // Second bar has significant lower wick
		bar1.Low > bar2.Low*0.995 && bar1.Low < bar2.Low*1.005 && // Similar lows
		bar1.C() < sma20 && // Price below MA (oversold)
		bar1.Range() > atr*0.8 && // Decent volatility
		bar1.WickDn() > bar1.WickUp() && // Lower wick larger than upper
		bar2.WickDn() > bar2.WickUp() { // Lower wick larger than upper

		event.Type = history.MARKET_BUY
		event.Name = "DOUBLE_WICK_BUY"
		event.Time = bars[0].T()
		event.Price = bars[0].C()
		event.Text = fmt.Sprintf("Lower wicks: %.8f, %.8f", bar1.WickDn(), bar2.WickDn())
		return event, true
	}

	return event, false
}
