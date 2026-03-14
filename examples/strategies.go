package main

import "github.com/slicken/history"

// Engulfing test strategy
type Engulfing struct {
	history.BaseStrategy
}

// Name returns the strategy name
func (s *Engulfing) Name() string {
	return "Engulfing"
}

func NewEngulfing() *Engulfing {
	return &Engulfing{
		BaseStrategy: *history.NewBaseStrategy("Engulfing"),
	}
}

// Event EngulfingN..
func (s *Engulfing) OnBar(symbol string, bars history.Bars) (history.Event, bool) {
	if len(bars) < 20 {
		return history.Event{}, false
	}

	// close position on next bar
	if _, exists := s.GetPortfolioManager().Positions[symbol]; exists {
		return s.Close(), true
	}

	SMA := bars[0:20].SMA(history.C)
	ATR := bars[1:4].ATR()

	// MARKET_BUY signal
	if bars.LastBearIdx() < 5 &&
		bars[0].C()-SMA < 2*ATR &&
		bars[0].C() > bars[bars.LastBearIdx()].H() &&
		bars[0].O() < bars[bars.LastBearIdx()].H() &&
		bars[0].Body() > ATR &&
		bars[0].O()-SMA < 2*ATR &&
		bars[0].C() > SMA {

		return s.BuyEvent(s.GetPortfolioManager().GetStats().CurrentBalance*0.20, bars[0].Close), true
	}

	// MARKET_SELL signal
	if bars.LastBullIdx() < 5 &&
		bars[0].O() > bars[bars.LastBullIdx()].L() &&
		bars[0].C() < bars[bars.LastBullIdx()].L() &&
		bars[0].Body() > ATR &&
		bars[0].O()-SMA < 2*ATR &&
		bars[0].C() < SMA {

		return s.SellEvent(s.GetPortfolioManager().Balance*0.20, bars[0].C()), true
	}

	return history.Event{}, false
}

// PercScalper strategy
type PercScalper struct {
	history.BaseStrategy
	buy_perc    float64 // Buy on perc dips
	sell_perc   float64 // Sell on perc spikes
	trail_start float64 // Trail start percentage
	nClose      int     // Close after N bars
	maxPos      int     // Max positions
	climaxHi    float64 // Climax distance ma R
	climaxMove  float64 // Climax 3day move perc
}

// Name returns the strategy name
func (s *PercScalper) Name() string {
	return "PercScalper"
}

func NewPercScalper() *PercScalper {
	return &PercScalper{
		BaseStrategy: *history.NewBaseStrategy("PercScalper"),
		buy_perc:     2.5, // Buy on perc dips
		sell_perc:    0.0, // Sell on perc spikes
		trail_start:  0.2, // trail start (0=off)
		nClose:       0,   // close pos after N bars'
		maxPos:       1,   // max positions
		climaxHi:     2.8, // climax distance ma R (0=off)
		climaxMove:   0.0, // climax 3day move perc (0=off
	}
}

// Event PercScalper
func (s *PercScalper) OnBar(symbol string, bars history.Bars) (history.Event, bool) {
	if len(bars) < 20 {
		return history.Event{}, false
	}

	// Get current position
	portfolio := s.GetPortfolioManager()
	position, hasPosition := portfolio.Positions[symbol]

	// Calculate indicators
	ema10 := bars[0:10].EMA(history.C)
	atr := bars[0:14].ATR()

	// Calculate price movements
	price_dip := (bars[0].Open - bars[0].Close) / bars[0].Open * 100
	price_spike := 0.0
	if hasPosition {
		price_spike = (bars[0].Close - position.EntryPrice) / position.EntryPrice * 100
	}

	// Define conditions
	buy_condition := price_dip >= s.buy_perc
	sell_condition := s.sell_perc > 0 && price_spike >= s.sell_perc

	// Close after N bars
	if s.nClose > 0 && hasPosition {
		entryBar := position.EntryTime
		barCount := 0
		for _, bar := range bars {
			if bar.Time.After(entryBar) {
				barCount++
			}
		}
		if barCount >= s.nClose {
			return s.Close(), true
		}
	}

	// Buy logic - remove len(portfolio.Positions) check since we already use !hasPosition
	if buy_condition && !hasPosition { // !hasPosition already ensures max 1 position per symbol
		return s.BuyEvent(s.GetPortfolioManager().GetStats().CurrentBalance*0.20, bars[0].Close), true
	}

	// Sell logic
	if sell_condition && hasPosition {
		return s.Close(), true
	}

	// Trailing stop
	trailLong := hasPosition && bars[0].Close >= position.EntryPrice*(1+s.trail_start/100)
	trailFilter := bars[0].Open > ema10 && bars[0].Close < ema10
	if s.trail_start > 0 && trailLong && trailFilter {
		return s.Close(), true
	}

	// Climax high
	climaxHigh := bars[0].Close > ema10+s.climaxHi*atr
	if s.climaxHi > 0 && climaxHigh && hasPosition {
		return s.Close(), true
	}

	// Climax move
	if s.climaxMove > 0 && len(bars) >= 3 {
		climaxMoveUp := (bars[0].Close/bars[3].Close-1)*100 > s.climaxMove
		if climaxMoveUp && hasPosition {
			return s.Close(), true
		}
	}

	return history.Event{}, false
}
