package history

import (
	"fmt"
	"log"
	"sync"
	"time"
)

const timeFormat = "2006/01/02 15:04"

// Strategy interface using Bars type as Event condition
type Strategy interface {
	Event(Bars) (Event, bool)
}

// MultiStrategy interface using Data type as Event condition
// Events can be built from multiple Symbol or Timeframe conditions
type MultiStrategy interface {
	Event(*Data) (Event, bool)
}

// // Events runs strategies on bars data
// func (bars Bars) Events(strategies ...Strategy) (Event, bool) {
// 	var event Event
// 	if len(bars) == 0 {
// 		return event, false
// 	}

// 	for _, strat := range strategies {
// 		// run strategy
// 		if new, ok := strat.Event(bars); ok {

// 			if event.Time.IsZero() {
// 				event = new
// 				continue
// 			}
// 			// event.Name += " " + new.Name
// 		}
// 	}
// 	return event, !event.Time.IsZero()
// }

// // Events strategys compatible with both Strategy (bars) and MultiStrategy (data)
// func (data *Data) Events(strategies ...interface{}) (Events, error) {
// 	events := make(Events, 0)
// 	if len(data.History) == 0 {
// 		return events, errNoHist
// 	}

// 	for _, strat := range strategies {
// 		// MultiStrategy
// 		if strat, ok := strat.(MultiStrategy); ok {

// 			// strategy condition event
// 			if event, ok := strat.Event(data); ok {
// 				// add if new
// 				if !event.Exists(events) {
// 					events = append(events, event)
// 				}
// 			}
// 		}
// 		// Strategy
// 		if strat, ok := strat.(Strategy); ok {

// 			var wg sync.WaitGroup
// 			for _, hist := range data.History {

// 				wg.Add(1)
// 				go func(hist *History, rw *sync.RWMutex) {
// 					defer wg.Done()

// 					bars := data.Bars(hist.Symbol, hist.Timeframe)

// 					if event, ok := strat.Event(bars); ok {

// 						event.Symbol = hist.Symbol
// 						event.Timeframe = hist.Timeframe
// 						// add if new
// 						if !event.Exists(events) {
// 							rw.Lock()
// 							events = append(events, event)
// 							rw.Unlock()
// 						}
// 					}
// 				}(hist, &data.RWMutex)
// 			}
// 			wg.Wait()
// 		}
// 	}

// 	log.Printf("scan completed with %d Events\n", len(events))
// 	return events, nil
// }

// Tester is strategy backtester interface
type Tester interface {
	Test(Strategy, time.Time, time.Time) (Events, error)
}

// Test strategys compatible with both Strategy (bars) and MultiStrategy
func (data *Data) Test(strat interface{}, start, end time.Time) (Events, error) {
	events := make(Events, 0)
	if len(data.History) == 0 {
		return events, errNoHist
	}
	log.Printf("TEST %s\t %v --> %v\n", fmt.Sprintf("%T", strat)[6:], start.Format(timeFormat), end.Format(timeFormat))

	// MultiStrategy
	if strat, ok := strat.(MultiStrategy); ok {
		// stream data
		for d := range data.Stream(start, end, data.MinPeriod()) {

			if event, ok := strat.Event(d); ok {

				// add if new
				if !events.Exists(event) {
					events = append(events, event)
				}
			}
		}
	}
	// BarStrategy
	if strat, ok := strat.(Strategy); ok {

		var wg sync.WaitGroup
		for _, hist := range data.History {

			wg.Add(1)
			go func(hist *History, rw *sync.RWMutex) {
				defer wg.Done()

				bars := data.Bars(hist.Symbol, hist.Timeframe)
				for b := range bars.Stream(start, end, bars.Period()) {

					if event, ok := strat.Event(b); ok {
						event.Symbol = hist.Symbol
						event.Timeframe = hist.Timeframe

						// add if new
						if !events.Exists(event) {
							rw.Lock()
							events = append(events, event)
							rw.Unlock()
						}
					}
				}
			}(hist, &data.RWMutex)
		}
		wg.Wait()
	}

	log.Printf("TEST completed with %d Events\n", len(events))
	return events, nil
}
