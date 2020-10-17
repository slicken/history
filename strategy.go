package history

import (
	"fmt"
	"log"
	"sync"
	"time"
)

const TFORMAT = "2006/01/02 15:04"

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
	log.Printf("TEST %s\t %v --> %v\n", fmt.Sprintf("%T", strat)[6:], start.Format(TFORMAT), end.Format(TFORMAT))

	// MultiStrategy
	if strat, ok := strat.(MultiStrategy); ok {
		// stream data
		for d := range data.Stream(start, end, data.MinPeriod()) {

			if event, ok := strat.Event(d); ok {

				// add if new
				if !event.Exists(events) {
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
						if !event.Exists(events) {
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

// Event will hold event data for specific time and price
type Event struct {
	Symbol, Timeframe string
	Type, Text        string
	Time              time.Time
	Price             float64
}

// Add ..
func (e *Event) Add(typ, text string, time time.Time, price float64) {
	e.Type = typ
	e.Text = text
	e.Time = time
	e.Price = price
	//	e.Name = fmt.Sprintf("buy %.8f", price)
}

// Exists ..
func (e *Event) Exists(events Events) bool {
	for _, old := range events {
		if e.Time == old.Time && e.Price == old.Price {
			return true
		}
	}

	return false
}

// // WithinBars ..
// func (e *Event) WithinBars(events Events, limit int) bool {

// 	for _, o := range events {
// 		if e.Symbol == o.Symbol && e.Timeframe == o.Timeframe {
// 			tf := time.Duration(Tf(o.Timeframe)) * time.Minute
// 			diff := o.Time.Sub(e.Time)
// 			bbw := -int(diff / tf)
// 			if limit > bbw {
// 				return true
// 			}
// 		}
// 	}

// 	return false
// }

// Events type
type Events []Event

// Map events
func (ev Events) Map() map[string]Events {
	m := make(map[string]Events, 0)

	for _, e := range ev {
		// describe key
		key := e.Symbol + e.Timeframe
		if key == "" {
			key = "unknown"
		}
		if _, ok := m[key]; !ok {
			m[key] = Events{e}
			continue
		}
		m[key] = append(m[key], e)
	}
	return m
}
