package history

import (
	"fmt"
	"time"
)

// Strategy interface defines the minimum requirements for implementing a trading strategy
type Strategy interface {
	// OnBar is main function that computes strategy for each bar and returns events
	// called for each barsymbol and its bars, returns an event if strategy conditions are met
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

// BuyEvent creates a buy event with specified size and price
func (s *BaseStrategy) BuyEvent(riskAmount float64, price float64) Event {
	event := Event{
		Symbol: s.symbol,
		Name:   s.name,
		Type:   MARKET_BUY,
		Time:   s.time,
		Price:  price,
		Size:   riskAmount,
		Text:   "Buy",
	}

	// Update portfolio
	if s.portfolio != nil {
		s.portfolio.Lock()
		defer s.portfolio.Unlock()

		// Calculate position size based on available balance
		positionSize := riskAmount
		units := positionSize / price // Calculate actual units to buy

		// Open long position if we have enough balance
		if s.portfolio.Balance >= positionSize {
			s.portfolio.Balance -= positionSize // Deduct the position size from balance
			pos := &Position{
				Symbol:     s.symbol,
				Side:       true, // long
				EntryTime:  s.time,
				EntryPrice: price,
				Size:       positionSize,
				Units:      units,
				Current:    price,
				OpenEvent:  event,
			}
			s.portfolio.Positions[s.symbol] = append(s.portfolio.Positions[s.symbol], pos)
		}
	}

	return event
}

// SellEvent creates a sell event with specified size and price
func (s *BaseStrategy) SellEvent(riskAmount float64, price float64) Event {
	event := Event{
		Symbol: s.symbol,
		Name:   s.name,
		Type:   MARKET_SELL,
		Time:   s.time,
		Price:  price,
		Size:   riskAmount,
		Text:   "Sell",
	}

	// Update portfolio
	if s.portfolio != nil {
		s.portfolio.Lock()
		defer s.portfolio.Unlock()

		// Calculate position size based on available balance
		positionSize := riskAmount
		units := positionSize / price // Calculate actual units to sell

		// Open short position if we have enough balance
		if s.portfolio.Balance >= positionSize {
			s.portfolio.Balance -= positionSize // Deduct the position size from balance
			pos := &Position{
				Symbol:     s.symbol,
				Side:       false, // short
				EntryTime:  s.time,
				EntryPrice: price,
				Size:       positionSize,
				Units:      units,
				Current:    price,
				OpenEvent:  event,
			}
			s.portfolio.Positions[s.symbol] = append(s.portfolio.Positions[s.symbol], pos)
		}
	}

	return event
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

	// Calculate P&L and close position (ClosePosition updates Balance internally)
	pnl := 0.0
	if s.portfolio != nil {
		for _, pos := range s.portfolio.Positions[openEvent.Symbol] {
			if pos.OpenEvent.Time.Equal(openEvent.Time) && pos.OpenEvent.Price == openEvent.Price {
				pnl = s.portfolio.ClosePosition(pos, closePrice, s.time)
				break
			}
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

// Buy creates a buy event with default size of 1000 and current price
func (s *BaseStrategy) Buy() Event {
	return s.BuyEvent(1000, s.price)
}

// Sell creates a sell event with default size of 1000 and current price
func (s *BaseStrategy) Sell() Event {
	return s.SellEvent(1000, s.price)
}

// Close is a helper that finds the oldest position for the symbol and closes it at current price
func (s *BaseStrategy) Close() Event {
	if s.portfolio != nil {
		var openEvent Event
		s.portfolio.Lock()
		positions := s.portfolio.Positions[s.symbol]
		if len(positions) > 0 {
			openEvent = positions[0].OpenEvent
		}
		s.portfolio.Unlock()
		if len(positions) > 0 {
			return s.CloseEvent(openEvent, s.price)
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
	} //
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
