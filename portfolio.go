package history

import (
	"log"
	"sync"
	"time"
)

// Position represents an open trading position
type Position struct {
	Symbol     string    // Trading pair
	Side       bool      // true for long, false for short
	EntryTime  time.Time // When position was opened
	EntryPrice float64   // Entry price
	Size       float64   // Position size in USD value
	Units      float64   // Actual units of the asset
	Current    float64   // Current price
	PnL        float64   // Current unrealized profit/loss
	OpenEvent  Event     // Event that opened this position
}

// PositionValue returns the current value of the position
func (p Position) PositionValue() float64 {
	return p.Size // Return size in USD value
}

// UnrealizedPnL returns the unrealized profit/loss of the position
func (p Position) UnrealizedPnL() float64 {
	// Calculate PnL based on actual units and price difference
	if p.Side {
		// Long position: profit = (current - entry) * units
		return (p.Current - p.EntryPrice) * p.Units
	}
	// Short position: profit = (entry - current) * units
	return (p.EntryPrice - p.Current) * p.Units
}

// PortfolioStats holds the portfolio performance metrics
type PortfolioStats struct {
	InitialBalance   float64
	Equity           float64 // Cash + open positions at mark-to-market
	Balance          float64 // Cash only (excludes open positions)
	OpenPositionsCnt int
	TotalPnL         float64
	UnrealizedPnL    float64
	RealizedPnL      float64
	TotalTrades      int
	WinningTrades    int
	LosingTrades     int
	WinRate          float64
	MaxDrawdown      float64
	HighWaterMark    float64
}

// PortfolioManager handles position tracking and P&L calculations
type PortfolioManager struct {
	Balance   float64                // Current balance
	Positions map[string][]*Position // Open positions by symbol (multiple per symbol for pyramiding)
	Stats     *PortfolioStats        // Trading statistics
	sync.RWMutex
}

// NewPortfolioManager creates a new portfolio manager with initial balance
func NewPortfolioManager() *PortfolioManager {
	initialBalance := 10000.0
	return &PortfolioManager{
		Balance:   initialBalance,
		Positions: make(map[string][]*Position),
		Stats: &PortfolioStats{
			InitialBalance: initialBalance,
			Equity:         initialBalance,
			Balance:        initialBalance,
			HighWaterMark:  initialBalance,
		},
	}
}

// PositionsForSymbol returns all open positions for a symbol
func (pm *PortfolioManager) PositionsForSymbol(symbol string) []*Position {
	pm.RLock()
	defer pm.RUnlock()
	return pm.Positions[symbol]
}

// TotalPositions returns the total number of open positions across all symbols
func (pm *PortfolioManager) TotalPositions() int {
	pm.RLock()
	defer pm.RUnlock()
	n := 0
	for _, positions := range pm.Positions {
		n += len(positions)
	}
	return n
}

// UpdatePosition updates the current price of all positions for a symbol and recalculates stats
func (pm *PortfolioManager) UpdatePosition(symbol string, currentPrice float64) {
	pm.Lock()
	defer pm.Unlock()
	for _, pos := range pm.Positions[symbol] {
		pos.Current = currentPrice
		pos.PnL = pos.UnrealizedPnL()
	}
	if len(pm.Positions[symbol]) > 0 {
		pm.updateStats()
	}
}

// ClosePosition closes a position and updates realized P&L
func (pm *PortfolioManager) ClosePosition(position *Position, closePrice float64, closeTime time.Time) float64 {
	if position == nil {
		return 0
	}

	var pnl float64
	if position.Side {
		pnl = (closePrice - position.EntryPrice) * position.Units
	} else {
		pnl = (position.EntryPrice - closePrice) * position.Units
	}

	// Return position size and P&L to cash
	pm.Balance += position.Size + pnl

	// Remove position from slice
	positions := pm.Positions[position.Symbol]
	for i, p := range positions {
		if p == position {
			pm.Positions[position.Symbol] = append(positions[:i], positions[i+1:]...)
			if len(pm.Positions[position.Symbol]) == 0 {
				delete(pm.Positions, position.Symbol)
			}
			break
		}
	}

	// Update stats to get correct Balance and Equity after close
	pm.Stats.RealizedPnL += pnl
	pm.Stats.TotalTrades++
	if pnl > 0 {
		pm.Stats.WinningTrades++
	} else if pnl < 0 {
		pm.Stats.LosingTrades++
	}
	pm.updateStats()

	barTime := closeTime.Format("2006/01/02 15:04")
	action := "SELL"
	if !position.Side {
		action = "BUY"
	}
	log.Printf("[TEST] #%d %s %s @%.2f -> %.2f PNL: %+.2f Balance: %.2f (Equity: %.2f)",
		pm.Stats.TotalTrades, barTime, action, position.EntryPrice, closePrice, pnl, pm.Stats.Balance, pm.Stats.Equity)

	return pnl
}

// updateStats recalculates portfolio statistics
func (pm *PortfolioManager) updateStats() {
	stats := pm.Stats
	unrealizedPnL := 0.0
	positionValue := 0.0

	// Calculate unrealized P&L and position value from open positions
	openCnt := 0
	for _, positions := range pm.Positions {
		for _, pos := range positions {
			unrealizedPnL += pos.UnrealizedPnL()
			positionValue += pos.Size
			openCnt++
		}
	}

	stats.UnrealizedPnL = unrealizedPnL
	stats.OpenPositionsCnt = openCnt
	stats.Balance = pm.Balance + positionValue   // total capital (cash + position at cost)
	stats.Equity = stats.Balance + unrealizedPnL // mark-to-market value
	stats.TotalPnL = stats.RealizedPnL + stats.UnrealizedPnL

	// Update win rate
	if stats.TotalTrades > 0 {
		stats.WinRate = float64(stats.WinningTrades) / float64(stats.TotalTrades)
	}

	// Update high water mark and drawdown
	if stats.Equity > stats.HighWaterMark {
		stats.HighWaterMark = stats.Equity
	}
	currentDrawdown := 0.0
	if stats.HighWaterMark > 0 {
		currentDrawdown = (stats.HighWaterMark - stats.Equity) / stats.HighWaterMark
	}
	if currentDrawdown > stats.MaxDrawdown {
		stats.MaxDrawdown = currentDrawdown
	}
}

// GetStats returns a copy of the current portfolio statistics
func (pm *PortfolioManager) GetStats() PortfolioStats {
	pm.RLock()
	defer pm.RUnlock()
	return *pm.Stats
}
