package history

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

const tformat = "2006/01/02 15:04"

// Strategy interface using Bars type as Event condition
type Strategy interface {
	Event(Bars) (Event, bool)
}

// MultiStrategy interface using Data type as Event condition
// Events can be built from multiple Symbol or Timeframe conditions
type MultiStrategy interface {
	Event(Data) (Event, bool)
}

// Events strategy Event on bars data
func (bars Bars) Events(strategies ...Strategy) (Event, bool) {
	var event Event
	if len(bars) == 0 {
		return event, false
	}

	for _, strat := range strategies {
		// strategy condition event
		if new, ok := strat.Event(bars); ok {

			if event.Time.IsZero() {
				event = new
				continue
			}
			event.Name += " " + new.Name
		}
	}

	return event, !event.Time.IsZero()
}

// Events strategys compatible with both Strategy (bars) and MultiStrategy
func (data Data) Events(strategies ...interface{}) (Events, error) {
	events := make(Events, 0)
	if len(data.History) == 0 {
		return events, errors.New("no histories")
	}
	log.Printf("Scanning Events %T\n", strategies)

	for _, strat := range strategies {
		// MultiStrategy
		if strat, ok := strat.(MultiStrategy); ok {

			// strategy condition event
			if event, ok := strat.Event(data); ok {
				// add if new
				if !events.Exist(event) {
					events = append(events, event)
				}
			}
		}
		// Strategy
		if strat, ok := strat.(Strategy); ok {

			var wg sync.WaitGroup
			for _, hist := range data.History {

				wg.Add(1)
				go func(hist *History, rw *sync.RWMutex) {
					defer wg.Done()

					bars := data.Bars(hist.Symbol, hist.Timeframe)

					if event, ok := strat.Event(bars); ok {

						event.Symbol = hist.Symbol
						event.Timeframe = hist.Timeframe
						// add if new
						if !events.Exist(event) {
							rw.Lock()
							events = append(events, event)
							rw.Unlock()
						}
					}
				}(hist, &data.RWMutex)
			}
			wg.Wait()
		}
	}

	log.Printf("Scan complete! result %d Events\n", len(events))
	return events, nil
}

// Tester is strategy backtester interface
type Tester interface {
	Test(Strategy, time.Time, time.Time) (Events, error)
}

// Test strategies on bars data
func (bars Bars) Test(strat Strategy, start, end time.Time) (Events, error) {
	events := make(Events, 0)
	if len(bars) == 0 {
		return events, errors.New("no bars")
	}
	log.Printf("Test %T...\t %v --> %v\n", strat, start.Format(tformat), end.Format(tformat))

	for b := range bars.Stream(start, end, bars.Period()) {

		if event, ok := strat.Event(b); ok {
			// if new equals last, skip
			if len(events) > 0 {
				if events[len(events)-1].Time == event.Time {
					continue
				}
			}
			events = append(events, event)
		}
	}

	log.Printf("Test complete! result %d Events\n", len(events))
	return events, nil
}

// Test strategys compatible with both Strategy (bars) and MultiStrategy
func (data Data) Test(strat interface{}, start, end time.Time) (Events, error) {
	events := make(Events, 0)
	if len(data.History) == 0 {
		return events, errors.New("no histories")
	}
	log.Printf("Test %T...\t %v --> %v\n", strat, start.Format(tformat), end.Format(tformat))

	// MultiStrategy
	if strat, ok := strat.(MultiStrategy); ok {
		// stream data
		for d := range data.Stream(start, end, data.Period()) {

			if event, ok := strat.Event(d); ok {

				// add if new
				if !events.Exist(event) {
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
						if !events.Exist(event) {
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

	log.Printf("Test complete! result %d Events\n", len(events))
	return events, nil
}

// Event will hold event data for specific time and price
type Event struct {
	Time              time.Time
	Price             float64
	Symbol, Timeframe string
	Type, Name        string
}

// Events type
type Events []Event

// Exist checks if event exists
func (ev Events) Exist(new Event) bool {
	for _, old := range ev {
		if new.Symbol == old.Symbol && new.Timeframe == old.Timeframe && new.Time == old.Time {
			return true
		}
	}
	return false
}

// Buy event
func (e *Event) Buy(time time.Time, price float64) {
	e.Time = time
	e.Price = price
	e.Type = "B"
	e.Name = fmt.Sprintf("buy %.8f", price)
}

// Sell event
func (e *Event) Sell(time time.Time, price float64) {
	e.Time = time
	e.Price = price
	e.Type = "S"
	e.Name = fmt.Sprintf("sell %f", price)
}

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

// Subscribe Strategies that runs everytime
// updater channel (data.C) gets updated.
func (data *Data) Subscribe(events *Events, strategies ...Strategy) {

	var n string
	for _, strat := range strategies {
		n += fmt.Sprintf("%T ", strat)[6:]
	}

	log.Printf("Subscribed %q\n.", n)
	go func() {
		for {

			select {
			case s, ok := <-data.C:
				if !ok {
					log.Println("Subscriber stopped.")
				}
				symbol, timeframe := Split(s)
				if symbol == "" || timeframe == "" {
					continue
				}
				bars := data.Bars(symbol, timeframe)
				if event, ok := bars.Events(strategies...); ok && !events.Exist(event) {

					event.Symbol = symbol
					event.Timeframe = timeframe
					*events = append(*events, event)
					fmt.Printf("%s%s [%s] %s\n", symbol, timeframe, event.Type, event.Name)
				}
			default:
				time.Sleep(time.Second)
			}
		}
	}()
}
