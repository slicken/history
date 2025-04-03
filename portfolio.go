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
}

// PortfolioManager handles all portfolio-related operations
type PortfolioManager struct {
	sync.RWMutex
	Balance   float64             // Available balance
	Positions map[string]Position // Open positions by symbol
}

// NewPortfolioManager creates a new portfolio manager with initial balance
func NewPortfolioManager(initialBalance float64) *PortfolioManager {
	return &PortfolioManager{
		Balance:   initialBalance,
		Positions: make(map[string]Position),
	}
}
