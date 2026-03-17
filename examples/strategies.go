package main

import (
	"math"
	"time"

	"github.com/slicken/history"
)

// Engulfing test strategy with trailing stop (same logic as PercScalper)
type Engulfing struct {
	history.BaseStrategy
	trail_bars int // Trail Bars (0=off) - lowest(low) for long, highest(high) for short
}

// Name returns the strategy name
func (s *Engulfing) Name() string {
	return "Engulfing"
}

func NewEngulfing() *Engulfing {
	return &Engulfing{
		BaseStrategy: *history.NewBaseStrategy("Engulfing"),
		trail_bars:   1,
	}
}

// Event EngulfingN..
func (s *Engulfing) OnBar(symbol string, bars history.Bars) (history.Event, bool) {
	minBars := 20
	if s.trail_bars > 0 {
		minBars = s.trail_bars + 2
		if minBars < 20 {
			minBars = 20
		}
	}
	if len(bars) < minBars {
		return history.Event{}, false
	}

	portfolio := s.GetPortfolioManager()
	positions := portfolio.PositionsForSymbol(symbol)
	hasPosition := len(positions) > 0

	// --- Exit Logic: trailing stop (trails as soon as position exists) ---
	if hasPosition {
		position := positions[0]
		return s.CloseEvent(position.OpenEvent, bars[0].Close), true

		// if position.Side {
		// 	// Long: trail_stop = lowest(low, trail_bars)[1], exit when low <= trailStop
		// 	if s.trail_bars > 0 && len(bars) >= s.trail_bars+1 {
		// 		trailStopPrice := bars[1 : s.trail_bars+1].Lowest(history.L)
		// 		if bars[0].Low <= trailStopPrice {
		// 			return s.CloseEvent(position.OpenEvent, bars[0].Close), true
		// 		}
		// 	}
		// } else {
		// 	// Short: trail_stop = highest(high, trail_bars)[1], exit when high >= trailStop
		// 	if s.trail_bars > 0 && len(bars) >= s.trail_bars+1 {
		// 		trailStopPrice := bars[1 : s.trail_bars+1].Highest(history.H)
		// 		if bars[0].High >= trailStopPrice {
		// 			return s.CloseEvent(position.OpenEvent, bars[0].Close), true
		// 		}
		// 	}
		// }
	}

	// No new entry while in position
	if hasPosition {
		return history.Event{}, false
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

		return s.BuyEvent(1000, bars[0].Close), true
	}

	// MARKET_SELL signal
	if bars.LastBullIdx() < 5 &&
		bars[0].O() > bars[bars.LastBullIdx()].L() &&
		bars[0].C() < bars[bars.LastBullIdx()].L() &&
		bars[0].Body() > ATR &&
		bars[0].O()-SMA < 2*ATR &&
		bars[0].C() < SMA {

		return s.SellEvent(1000, bars[0].Close), true
	}

	return history.Event{}, false
}

// Turtle strategy - matches Pine Script slk_strategy_turle exactly
// Entry: 35-bar Donchian breakout, Exit: 20-bar Donchian, Stop: 2*ATR(20)
type Turtle struct {
	history.BaseStrategy
	entryLen int
	exitLen  int
	atrLen   int
}

func (s *Turtle) Name() string {
	return "Turtle"
}

func NewTurtle() *Turtle {
	return &Turtle{
		BaseStrategy: *history.NewBaseStrategy("Turtle"),
		entryLen:     35,
		exitLen:      20,
		atrLen:       20,
	}
}

func (s *Turtle) OnBar(symbol string, bars history.Bars) (history.Event, bool) {
	entryLen, exitLen, atrLen := s.entryLen, s.exitLen, s.atrLen
	minBars := entryLen + 2
	if len(bars) < minBars {
		return history.Event{}, false
	}

	// Donchian channels (exclude current bar: bars[1:n+1])
	upperEntry := bars[1 : entryLen+1].Highest(history.H)
	lowerEntry := bars[1 : entryLen+1].Lowest(history.L)
	upperExit := bars[1 : exitLen+1].Highest(history.H)
	lowerExit := bars[1 : exitLen+1].Lowest(history.L)

	N := bars[1 : atrLen+1].ATRWilder()

	portfolio := s.GetPortfolioManager()
	positions := portfolio.PositionsForSymbol(symbol)
	hasPosition := len(positions) > 0
	var position *history.Position
	if hasPosition {
		position = positions[0]
	}
	currHigh, currLow := bars[0].High, bars[0].Low
	prevHigh, prevLow := bars[1].High, bars[1].Low

	// Exit logic (check first - Turtles exit before new entry)
	if hasPosition {
		if position.Side {
			// Long: exit on low < lowerExit
			if currLow < lowerExit {
				return s.Close(), true
			}
			// Stop loss: entry - 2*N
			stopPrice := position.EntryPrice - 2*N
			if currLow <= stopPrice {
				return s.CloseEvent(position.OpenEvent, stopPrice), true
			}
		} else {
			// Short: exit on high > upperExit
			if currHigh > upperExit {
				return s.Close(), true
			}
			// Stop loss: entry + 2*N
			stopPrice := position.EntryPrice + 2*N
			if currHigh >= stopPrice {
				return s.CloseEvent(position.OpenEvent, stopPrice), true
			}
		}
	}

	// Entry logic (crossover/crossunder)
	longCondition := currHigh > upperEntry && prevHigh <= bars[2:entryLen+2].Highest(history.H)
	shortCondition := currLow < lowerEntry && prevLow >= bars[2:entryLen+2].Lowest(history.L)

	// Entry logic (Pine: pyramiding=0, so reversal = close + open opposite)
	if longCondition {
		if hasPosition && !position.Side {
			return s.Close(), true // close short first, next bar/call can enter long
		}
		if !hasPosition {
			size := portfolio.GetStats().Equity * 0.50
			return s.BuyEvent(size, bars[0].Close), true
		}
	}
	if shortCondition {
		if hasPosition && position.Side {
			return s.Close(), true // close long first, next bar/call can enter short
		}
		if !hasPosition {
			size := portfolio.GetStats().Equity * 0.50
			return s.SellEvent(size, bars[0].Close), true
		}
	}

	return history.Event{}, false
}

// PercScalper strategy - matches Pine slk_strategy_percs (excl. date range)
type PercScalper struct {
	history.BaseStrategy
	buy_perc          float64 // Buy on perc dips
	sell_perc         float64 // Sell on % profit
	trail_start       float64 // Trail Start % Profit - trail activates when profit >= this
	trail_bars        int     // Trail Bars (0=off) - lowest(low, trail_bars)[1]
	nClose            int     // Close after N bars
	maxPos            int     // Max positions
	single_order_calc bool    // true=single (each pos), false=multi (collective)
	closeAllPending   bool    // close_all: close remaining positions on same bar
}

// Name returns the strategy name
func (s *PercScalper) Name() string {
	return "PercScalper"
}

func NewPercScalper() *PercScalper {
	return &PercScalper{
		BaseStrategy:      *history.NewBaseStrategy("PercScalper"),
		buy_perc:          2.5,   // Buy on perc dips (Pine default)
		sell_perc:         0.0,   // Sell on % profit (Pine default)
		trail_start:       1.0,   // Trail Start % Profit (Pine default)
		trail_bars:        4,     // Trail Bars (Pine default)
		nClose:            0,     // close pos after N bars
		maxPos:            20,    // max positions
		single_order_calc: false, // "single" or "multi"
	}
}

// Event PercScalper - matches Pine slk_strategy_percs (excl. date range)
func (s *PercScalper) OnBar(symbol string, bars history.Bars) (history.Event, bool) {
	minBars := s.trail_bars + 2
	if s.trail_bars == 0 {
		minBars = 2
	}
	if len(bars) < minBars {
		return history.Event{}, false
	}

	portfolio := s.GetPortfolioManager()
	positions := portfolio.PositionsForSymbol(symbol)
	hasPosition := len(positions) > 0
	totalPositions := portfolio.TotalPositions()

	// price_dip = (open - close) / open * 100
	price_dip := (bars[0].Open - bars[0].Close) / bars[0].Open * 100

	// trail_stop_price[1] = lowest(low, trail_bars) from previous bar
	var trailStopPrice float64
	if s.trail_start > 0 && len(bars) >= s.trail_bars+1 {
		trailStopPrice = bars[1 : s.trail_bars+1].Lowest(history.L)
	}

	// closeAllPending: close remaining positions (multi close_all)
	if !hasPosition {
		s.closeAllPending = false
	}
	if s.closeAllPending && hasPosition {
		return s.Close(), true
	}

	// --- Exit Logic (when position_size > 0) ---
	if hasPosition {
		if s.single_order_calc {
			// Single: check each position
			for _, pos := range positions {
				current_profit_perc := (bars[0].Close - pos.EntryPrice) / pos.EntryPrice * 100
				barCount := 0
				for _, b := range bars {
					if b.Time.After(pos.EntryTime) {
						barCount++
					}
				}
				time_exit := s.nClose > 0 && barCount >= s.nClose
				trail_active := s.trail_bars > 0 && current_profit_perc >= s.trail_start
				trail_hit := trail_active && bars[0].Low <= trailStopPrice

				if (s.sell_perc > 0 && current_profit_perc >= s.sell_perc) || time_exit || trail_hit {
					return s.CloseEvent(pos.OpenEvent, bars[0].Close), true
				}
			}
		} else {
			// Multi (collective): avg_entry, total_profit_perc
			var totalCost, totalUnits float64
			for _, pos := range positions {
				totalCost += pos.EntryPrice * pos.Units
				totalUnits += pos.Units
			}
			avg_entry := totalCost / totalUnits
			total_profit_perc := (bars[0].Close - avg_entry) / avg_entry * 100

			trail_active := s.trail_bars > 0 && total_profit_perc >= s.trail_start
			trail_hit := trail_active && bars[0].Low <= trailStopPrice

			if s.sell_perc > 0 && total_profit_perc >= s.sell_perc {
				s.closeAllPending = true
				return s.Close(), true
			}
			if trail_hit {
				s.closeAllPending = true
				return s.Close(), true
			}
			if s.nClose > 0 {
				oldest := positions[0]
				barCount := 0
				for _, b := range bars {
					if b.Time.After(oldest.EntryTime) {
						barCount++
					}
				}
				if barCount >= s.nClose {
					s.closeAllPending = true
					return s.Close(), true
				}
			}
		}
	}

	// --- Buy logic ---
	if price_dip >= s.buy_perc && totalPositions < s.maxPos {
		return s.BuyEvent(1000, bars[0].Close), true
	}

	return history.Event{}, false
}

// Ratings strategy - matches Pine slk_strategy_ratings
// Entry: new day only, when MA rating avg > StrongBound (long) or < -StrongBound (short)
// Exit: trailing stop (loss 3*ATR, trail 5*ATR, trail_offset 2*ATR)
type Ratings struct {
	history.BaseStrategy
	strongBound float64 // Strong Rating Bound (default 0.5)
	useOpen     bool    // true=Open, false=Close for execution price
}

func (s *Ratings) Name() string {
	return "Ratings"
}

func NewRatings() *Ratings {
	return &Ratings{
		BaseStrategy: *history.NewBaseStrategy("Ratings"),
		strongBound:  0.5,
		useOpen:      false,
	}
}

// calcRatingMA returns sign(src - ma): -1, 0, or 1
func calcRatingMA(ma, src float64) float64 {
	if src > ma {
		return 1
	}
	if src < ma {
		return -1
	}
	return 0
}

// calcMAOnlyRating returns average of 14 MA ratings (uses previous bar's close for MAs)
func calcMAOnlyRating(bars history.Bars) float64 {
	if len(bars) < 201 {
		return 0
	}
	// Use bars[1] as "current" for rating (Pine: ratingMA[1] = previous bar)
	// MAs computed on bars[1:n+1] for each period
	var sum float64
	close := bars[1].Close

	sum += calcRatingMA(bars[1:11].SMA(history.C), close)      // SMA 10
	sum += calcRatingMA(bars[1:21].SMA(history.C), close)      // SMA 20
	sum += calcRatingMA(bars[1:31].SMA(history.C), close)      // SMA 30
	sum += calcRatingMA(bars[1:51].SMA(history.C), close)      // SMA 50
	sum += calcRatingMA(bars[1:101].SMA(history.C), close)     // SMA 100
	sum += calcRatingMA(bars[1:201].SMA(history.C), close)     // SMA 200
	sum += calcRatingMA(bars[1:11].EMA(history.C), close)      // EMA 10
	sum += calcRatingMA(bars[1:21].EMA(history.C), close)      // EMA 20
	sum += calcRatingMA(bars[1:31].EMA(history.C), close)      // EMA 30
	sum += calcRatingMA(bars[1:51].EMA(history.C), close)      // EMA 50
	sum += calcRatingMA(bars[1:101].EMA(history.C), close)     // EMA 100
	sum += calcRatingMA(bars[1:201].EMA(history.C), close)     // EMA 200
	sum += calcRatingMA(bars[1:12].HMA(history.C, 9), close)   // HMA 9 (needs 11 bars)
	sum += calcRatingMA(bars[1:21].VWMA(history.C, 20), close) // VWMA 20

	return sum / 14
}

func (s *Ratings) OnBar(symbol string, bars history.Bars) (history.Event, bool) {
	if len(bars) < 202 {
		return history.Event{}, false
	}

	ratingMA := calcMAOnlyRating(bars)
	atr := bars[1:15].ATRWilder()
	if atr <= 0 {
		atr = bars[0].Range()
	}

	// New day check: bars[0] date != bars[1] date
	isNewDay := !bars[0].Time.Truncate(24 * time.Hour).Equal(bars[1].Time.Truncate(24 * time.Hour))

	portfolio := s.GetPortfolioManager()
	positions := portfolio.PositionsForSymbol(symbol)
	hasPosition := len(positions) > 0
	var position *history.Position
	if hasPosition {
		position = positions[0]
	}

	execPrice := bars[0].Close
	if s.useOpen {
		execPrice = bars[0].Open
	}

	// Exit logic: trailing stop (loss 3*ATR, trail 5*ATR, offset 2*ATR)
	if hasPosition {
		entryPrice := position.EntryPrice
		currHigh, currLow := bars[0].High, bars[0].Low

		// Find highest high / lowest low since entry
		var highestSince, lowestSince float64
		for _, b := range bars {
			if !b.Time.Before(position.EntryTime) {
				if highestSince == 0 || b.High > highestSince {
					highestSince = b.High
				}
				if lowestSince == 0 || b.Low < lowestSince {
					lowestSince = b.Low
				}
			}
		}
		if highestSince == 0 {
			highestSince = currHigh
		}
		if lowestSince == 0 {
			lowestSince = currLow
		}

		if position.Side {
			// Long: initial stop entry-3*ATR; trail activates at entry+2*ATR, trails 5*ATR below high
			stopPrice := entryPrice - 3*atr
			if highestSince >= entryPrice+2*atr {
				trailStop := highestSince - 5*atr
				if trailStop > stopPrice {
					stopPrice = trailStop
				}
			}
			if currLow <= stopPrice {
				return s.CloseEvent(position.OpenEvent, math.Max(stopPrice, currLow)), true
			}
		} else {
			// Short: initial stop entry+3*ATR; trail activates at entry-2*ATR
			stopPrice := entryPrice + 3*atr
			if lowestSince <= entryPrice-2*atr {
				trailStop := lowestSince + 5*atr
				if trailStop < stopPrice {
					stopPrice = trailStop
				}
			}
			if currHigh >= stopPrice {
				return s.CloseEvent(position.OpenEvent, math.Min(stopPrice, currHigh)), true
			}
		}
	}

	// Entry logic: only on new day, no position (pyramiding=0)
	size := portfolio.GetStats().Equity * 0.15 // 15% of equity (Pine default)

	if isNewDay && !hasPosition {
		if ratingMA > s.strongBound {
			return s.BuyEvent(size, execPrice), true
		}
		if ratingMA < -s.strongBound {
			return s.SellEvent(size, execPrice), true
		}
	}

	return history.Event{}, false
}
