package history

import (
	"errors"
	"log"
	"time"
)

const dtFormat = "2006/01/02 15:04"

// PortfolioStrategy interface for strategies that use portfolio management
type PortfolioStrategy interface {
	Strategy
	GetPortfolioManager() *PortfolioManager
}

// TestResult contains the test results including events and portfolio stats
type TestResult struct {
	Events         *Events
	PortfolioStats *PortfolioStats
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
func (t *Tester) Test(start, end time.Time) (*TestResult, error) {
	if len(t.hist.bars) == 0 {
		return nil, errors.New("no history")
	}

	log.Printf("[TEST] %s [%v ==> %v]\n", t.strategy.Name(), start.Format(dtFormat), end.Format(dtFormat))

	// Check if strategy implements PortfolioStrategy interface
	portfolioStrat, hasPortfolio := t.strategy.(PortfolioStrategy)
	var portfolio *PortfolioManager
	if hasPortfolio {
		portfolio = portfolioStrat.GetPortfolioManager()
		// Double check that we actually got a portfolio manager
		hasPortfolio = portfolio != nil
		if hasPortfolio {
			stats := portfolio.GetStats()
			log.Printf("[TEST] Initial Balance: %.2f\n", stats.InitialBalance)
		}
	}

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

		var currentBars Bars

		// Create a channel to receive bars using StreamInterval
		for bar := range bars.StreamInterval(start, end, bars.Period()) {
			// Skip empty bars
			if bar.Time.IsZero() {
				continue
			}

			// Prepend the new bar to our current bars
			currentBars = append(Bars{bar}, currentBars...)

			// Update portfolio positions with current price if portfolio exists
			if hasPortfolio && portfolio != nil {
				portfolio.UpdatePosition(symbol, bar.Close)
			}

			// Set context for the current bar if strategy supports it
			if baseStrat, ok := t.strategy.(interface{ SetContext(string, Bar) }); ok {
				baseStrat.SetContext(symbol, bar)
			}

			// Process strategy with all bars up to this point
			if event, ok := t.strategy.OnBar(symbol, currentBars); ok {
				// Add event to events list
				if !t.events.Add(event) {
					log.Printf("[TEST] could not add event: %+v\n", event)
				}
			}
			// Process strategy with all bars up to this point (loop for multiple closes per bar, e.g. close_all)
			// for {
			// 	if event, ok := t.strategy.OnBar(symbol, currentBars); ok {
			// 		if !t.events.Add(event) {
			// 			log.Printf("[TEST] could not add event: %+v\n", event)
			// 		}
			// 		if hasPortfolio && portfolio != nil && event.Type == CLOSE {
			// 			continue
			// 		}
			// 		break
			// 	} else {
			// 		break
			// 	}
			// }
		}
	}

	result := &TestResult{
		Events: t.events,
	}

	if hasPortfolio && portfolio != nil {
		log.Printf("[TEST] %s completed with %d Events (%d Trades)\n", t.strategy.Name(), len(*t.events), portfolio.Stats.TotalTrades)
	} else {
		log.Printf("[TEST] %s completed with %d Events\n", t.strategy.Name(), len(*t.events))
	}

	// Add portfolio stats if available
	if hasPortfolio && portfolio != nil {
		stats := portfolio.GetStats()
		result.PortfolioStats = &stats

		// Final Balance = closed balance (cash only, excludes open positions)
		closedBal := stats.Balance
		pctChange := (closedBal - stats.InitialBalance) / stats.InitialBalance * 100
		log.Printf("[TEST] Final Balance: %.2f (%+.2f%%)\n", closedBal, pctChange)

		if stats.OpenPositionsCnt > 0 {
			log.Printf("[TEST] %d open position(s), Unrealized P&L: %+.2f, Equity: %.2f\n",
				stats.OpenPositionsCnt, stats.UnrealizedPnL, stats.Equity)
		}

		log.Printf("[TEST] Win Rate: %.2f%% [W:%d|L:%d]\n", result.PortfolioStats.WinRate*100, result.PortfolioStats.WinningTrades, result.PortfolioStats.LosingTrades)
		log.Printf("[TEST] Max Drawdown: %.2f%%\n", result.PortfolioStats.MaxDrawdown*100)
	}

	return result, nil
}

// ClearEvents removes all events
func (t *Tester) ClearEvents() {
	t.events = new(Events)
}
