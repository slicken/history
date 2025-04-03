package history

import (
	"errors"
	"log"
	"time"
)

const dt_stamp = "2006/01/02 15:04"

// PortfolioStrategy interface for strategies that use portfolio management
type PortfolioStrategy interface {
	Strategy
	GetPortfolioManager() *PortfolioManager
}

// Tester handles backtesting of strategies
type Tester struct {
	hist     *History
	strategy Strategy
	events   *Events
}

// NewTester creates a new backtester instance
func NewTester(hist *History, strategy Strategy) *Tester {
	return &Tester{
		hist:     hist,
		strategy: strategy,
		events:   new(Events),
	}
}

// Test runs the strategy on historical data between start and end time
func (t *Tester) Test(start, end time.Time) (*Events, error) {
	if len(t.hist.bars) == 0 {
		return nil, errors.New("no history")
	}

	log.Printf("[TEST] %s [%v ==> %v]\n", t.strategy.Name(), start.Format(dt_stamp), end.Format(dt_stamp))

	// Get all symbols from history
	symbols := make([]string, 0)
	for symbol := range t.hist.Map() {
		symbols = append(symbols, symbol)
	}

	// Test each symbol
	for _, symbol := range symbols {
		bars := t.hist.GetBars(symbol)
		if len(bars) == 0 {
			continue
		}

		// Filter bars within time range
		bars = bars.TimeSpan(start, end)
		if len(bars) == 0 {
			continue
		}

		// Get the bar period for streaming interval
		period := bars.Period()

		// Create a channel to receive bars using StreamInterval
		barChan := bars.StreamInterval(start, end, period)

		// Process bars as they arrive
		var currentBars Bars
		for bar := range barChan {
			// Skip empty bars
			if bar.Time.IsZero() {
				continue
			}

			// Add the new bar to our current bars
			currentBars = append(Bars{bar}, currentBars...)

			// Process strategy with all bars up to this point
			if event, ok := t.strategy.OnBar(symbol, currentBars); ok {
				// Add event to events list
				t.events.Add(event)
			}
		}
	}

	log.Printf("[TEST] completed with %d Events\n", len(*t.events))
	return t.events, nil
}

// ClearEvents removes all events
func (t *Tester) ClearEvents() {
	t.events = new(Events)
}
