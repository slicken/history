package charts

import "github.com/slicken/history"

// ChartBuilder builds chart HTML from bars and events.
// Both HighChart and TradingView implement this interface.
type ChartBuilder interface {
	BuildCharts(m map[string]history.Bars, events map[string]history.Events) ([]byte, error)
}
