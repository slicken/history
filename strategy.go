package history

import (
	"fmt"
	"time"
)

// Strategy interface defines the minimum requirements for implementing a trading strategy
type Strategy interface {
	// OnBar is called for each symbol and its bars, returns an event if strategy conditions are met
	OnBar(symbol string, bars Bars) (Event, bool)
	// Name returns the strategy name, used for identifying events
	Name() string
}

// BaseStrategy provides common functionality for strategies
type BaseStrategy struct {
	portfolio *PortfolioManager
	symbol    string    // Current symbol being processed
	time      time.Time // Current bar time
	price     float64   // Current bar price (close)
	name      string    // Strategy name
}

// NewBaseStrategy creates a new base strategy with portfolio management
func NewBaseStrategy(name string) *BaseStrategy {
	return &BaseStrategy{
		portfolio: NewPortfolioManager(),
		name:      name,
	}
}

// GetPortfolioManager returns the portfolio manager
func (s *BaseStrategy) GetPortfolioManager() *PortfolioManager {
	return s.portfolio
}

// SetContext sets the current trading context from the bar
func (s *BaseStrategy) SetContext(symbol string, bar Bar) {
	s.symbol = symbol
	s.time = bar.Time
	s.price = bar.Close
}

// Buy creates a buy event with default size of 1000 and current price
func (s *BaseStrategy) Buy() Event {
	return s.BuyEvent(1000, s.price)
}

// BuyEvent creates a buy event with specified size and price
func (s *BaseStrategy) BuyEvent(size float64, price float64) Event {
	event := Event{
		Symbol: s.symbol,
		Name:   s.name,
		Type:   MARKET_BUY,
		Time:   s.time,
		Price:  price,
		Size:   size,
		Text:   "Buy",
	}

	// Update portfolio
	if s.portfolio != nil {
		s.portfolio.Lock()
		defer s.portfolio.Unlock()

		// Open long position if we have enough balance
		if s.portfolio.Balance >= size {
			s.portfolio.Balance -= size // Deduct the position size from balance
			s.portfolio.Positions[s.symbol] = &Position{
				Symbol:     s.symbol,
				Side:       true, // long
				EntryTime:  s.time,
				EntryPrice: price,
				Size:       event.Size,
				Current:    price,
				OpenEvent:  event,
			}
		}
	}

	return event
}

// Sell creates a sell event with default size of 1000 and current price
func (s *BaseStrategy) Sell() Event {
	return s.SellEvent(1000, s.price)
}

// SellEvent creates a sell event with specified size and price
func (s *BaseStrategy) SellEvent(size float64, price float64) Event {
	event := Event{
		Symbol: s.symbol,
		Name:   s.name,
		Type:   MARKET_SELL,
		Time:   s.time,
		Price:  price,
		Size:   size,
		Text:   "Sell",
	}

	// Update portfolio
	if s.portfolio != nil {
		s.portfolio.Lock()
		defer s.portfolio.Unlock()

		// Open short position if we have enough balance
		if s.portfolio.Balance >= size {
			s.portfolio.Balance -= size // Deduct the position size from balance
			s.portfolio.Positions[s.symbol] = &Position{
				Symbol:     s.symbol,
				Side:       false, // short
				EntryTime:  s.time,
				EntryPrice: price,
				Size:       event.Size,
				Current:    price,
				OpenEvent:  event,
			}
		}
	}

	return event
}

// Close is a helper that finds the latest position and its opening event, then closes it at current price
func (s *BaseStrategy) Close() Event {
	if s.portfolio != nil {
		s.portfolio.Lock()
		defer s.portfolio.Unlock()

		if pos, exists := s.portfolio.Positions[s.symbol]; exists {
			return s.CloseEvent(pos.OpenEvent, s.price)
		}
	}

	// If no position found, return empty event
	return Event{
		Symbol: s.symbol,
		Name:   s.name,
		Type:   OTHER,
		Time:   s.time,
		Price:  s.price,
		Size:   0,
		Text:   "No Position to Close",
	}
}

// CloseEvent creates a close position event for a given opening event and closing price
func (s *BaseStrategy) CloseEvent(openEvent Event, closePrice float64) Event {
	// Validate that this is an event that could have opened a position
	if openEvent.Type != MARKET_BUY && openEvent.Type != MARKET_SELL &&
		openEvent.Type != LIMIT_BUY && openEvent.Type != LIMIT_SELL &&
		openEvent.Type != STOP_BUY && openEvent.Type != STOP_SELL {
		return Event{
			Symbol: s.symbol,
			Name:   s.name,
			Type:   OTHER,
			Time:   s.time,
			Price:  closePrice,
			Size:   0,
			Text:   "Invalid Event Type to Close",
		}
	}

	// Determine if this was a long or short position
	isLong := openEvent.Type == MARKET_BUY || openEvent.Type == LIMIT_BUY || openEvent.Type == STOP_BUY
	positionType := "Long"
	if !isLong {
		positionType = "Short"
	}

	// Calculate P&L before closing
	pnl := 0.0
	if s.portfolio != nil {
		if pos, exists := s.portfolio.Positions[openEvent.Symbol]; exists {
			pnl = s.portfolio.ClosePosition(pos, closePrice)
			s.portfolio.Balance += pnl
		}
	}

	event := Event{
		Symbol: openEvent.Symbol,
		Name:   s.name,
		Time:   s.time,
		Price:  closePrice,
		Size:   openEvent.Size,
		Type:   CLOSE,
		Text:   fmt.Sprintf("Close %s: Entry %.2f Exit %.2f Size %.0f P&L %.2f", positionType, openEvent.Price, closePrice, openEvent.Size, pnl),
	}

	return event
}

// Sit creates a neutral event when no action is taken
func (s *BaseStrategy) Sit() Event {
	return Event{
		Symbol: s.symbol,
		Name:   s.name,
		Type:   OTHER,
		Time:   s.time,
		Price:  s.price,
		Size:   0,
		Text:   "No Action",
	}
}

// Name returns the strategy name
func (s *BaseStrategy) Name() string {
	return s.name
}
