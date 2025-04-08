package history

import (
	"sync"
	"time"
)

// Position represents an open trading position
type Position struct {
	Symbol     string    // Trading pair
	Side       bool      // true for long, false for short
	EntryTime  time.Time // When position was opened
	EntryPrice float64   // Entry price
	Size       float64   // Position size
	Current    float64   // Current price
	PnL        float64   // Current unrealized profit/loss
	OpenEvent  Event     // Event that opened this position
}

// PositionValue returns the current value of the position
func (p Position) PositionValue() float64 {
	return p.Size // Always 1000 since we use fixed size
}

// UnrealizedPnL returns the unrealized profit/loss of the position
func (p Position) UnrealizedPnL() float64 {
	// Scale factor is position size relative to initial balance (10000)
	scaleFactor := p.Size / 10000.0

	if p.Side {
		// Long position: profit = (current - entry) scaled by size/balance
		return (p.Current - p.EntryPrice) * scaleFactor
	}
	// Short position: profit = (entry - current) scaled by size/balance
	return (p.EntryPrice - p.Current) * scaleFactor
}

// PortfolioStats holds the portfolio performance metrics
type PortfolioStats struct {
	InitialBalance float64
	CurrentBalance float64
	TotalPnL       float64
	UnrealizedPnL  float64
	RealizedPnL    float64
	TotalTrades    int
	WinningTrades  int
	LosingTrades   int
	WinRate        float64
	MaxDrawdown    float64
	HighWaterMark  float64
}

// PortfolioManager handles position tracking and P&L calculations
type PortfolioManager struct {
	Balance   float64              // Current balance
	Positions map[string]*Position // Open positions by symbol
	Stats     *PortfolioStats      // Trading statistics
	sync.RWMutex
}

// NewPortfolioManager creates a new portfolio manager with initial balance
func NewPortfolioManager() *PortfolioManager {
	initialBalance := 10000.0
	return &PortfolioManager{
		Balance:   initialBalance,
		Positions: make(map[string]*Position),
		Stats: &PortfolioStats{
			InitialBalance: initialBalance,
			CurrentBalance: initialBalance,
			HighWaterMark:  initialBalance,
		},
	}
}

// UpdatePosition updates the current price of a position and recalculates stats
func (pm *PortfolioManager) UpdatePosition(symbol string, currentPrice float64) {
	if pos, exists := pm.Positions[symbol]; exists {
		// Update position's current price
		pos.Current = currentPrice

		// Calculate unrealized P&L
		pos.PnL = pos.UnrealizedPnL()

		// Update stats
		pm.updateStats()
	}
}

// ClosePosition closes a position and updates realized P&L
func (pm *PortfolioManager) ClosePosition(position *Position, closePrice float64) float64 {
	if position == nil {
		return 0
	}

	// Return position size to balance
	pm.Balance += position.Size

	// Calculate P&L
	var pnl float64
	// Scale factor is position size relative to initial balance (10000)
	scaleFactor := position.Size / 10000.0

	if position.Side {
		// Long position: profit = (close - entry)
		pnl = (closePrice - position.EntryPrice) * scaleFactor
	} else {
		// Short position: profit = (entry - close)
		pnl = (position.EntryPrice - closePrice) * scaleFactor
	}

	// Add PnL to balance
	pm.Balance += pnl

	// Update stats
	pm.Stats.RealizedPnL += pnl
	pm.Stats.TotalTrades++
	if pnl > 0 {
		pm.Stats.WinningTrades++
	} else if pnl < 0 {
		pm.Stats.LosingTrades++
	}

	delete(pm.Positions, position.Symbol)
	pm.updateStats()
	return pnl
}

// updateStats recalculates portfolio statistics
func (pm *PortfolioManager) updateStats() {
	stats := pm.Stats
	unrealizedPnL := 0.0

	// Calculate unrealized P&L from open positions
	for _, pos := range pm.Positions {
		unrealizedPnL += pos.UnrealizedPnL()
	}

	stats.UnrealizedPnL = unrealizedPnL
	stats.CurrentBalance = pm.Balance + unrealizedPnL
	stats.TotalPnL = stats.RealizedPnL + stats.UnrealizedPnL

	// Update win rate
	if stats.TotalTrades > 0 {
		stats.WinRate = float64(stats.WinningTrades) / float64(stats.TotalTrades)
	}

	// Update high water mark and drawdown
	if stats.CurrentBalance > stats.HighWaterMark {
		stats.HighWaterMark = stats.CurrentBalance
	}
	currentDrawdown := 0.0
	if stats.HighWaterMark > 0 {
		currentDrawdown = (stats.HighWaterMark - stats.CurrentBalance) / stats.HighWaterMark
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
