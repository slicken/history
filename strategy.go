package history

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
}
