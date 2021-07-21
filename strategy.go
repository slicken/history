package history

import (
	"fmt"
	"log"
	"time"
)

const TFMT = "2006/01/02 15:04"

// Strategy interface using Bars type as Event condition
type Strategy interface {
	Event(Bars) (Event, bool)
}

// Tester is strategy backtester interface
type Tester interface {
	Test(Strategy, time.Time, time.Time) (Events, error)
}

// Test strategys compatible with both Strategy (bars) and MultiStrategy (whole history struct)
func (data *History) Test(strat interface{}, start, end time.Time) (Events, error) {
	if len(data.Bars) == 0 {
		return nil, errNoHist
	}

	events := make(Events, 0)
	log.Printf("TEST %s\t %v --> %v\n", fmt.Sprintf("%T", strat)[6:], start.Format(TFMT), end.Format(TFMT))

	// BarStrategy
	if strat, ok := strat.(Strategy); ok {

		for symbol, bars := range data.Bars {

			for b := range bars.Stream(start, end, bars.Period()) {

				if event, ok := strat.Event(b); ok {
					event.Symbol, event.Timeframe = Split(symbol)

					if !events.Exists(event) {
						events = append(events, event)
					}
				}
			}
		}
	}

	log.Printf("TEST completed with %d Events\n", len(events))
	return events, nil
}
