package history

import (
	"errors"
	"fmt"
	"log"
	"time"
)

// EventListener is where you subscribe strategies too
type EventListener struct {
	running    bool
	strategies []Strategy
}

// Start event listener
func (e *EventListener) Start(hist *History, events *Events) {
	e.running = true

	log.Println("EVENTLISTENER started")

	for symbol := range hist.Bars {
		hist.C <- symbol
	}

	go func() {
		for {

			select {
			case symbol, ok := <-hist.C:
				if !ok || len(e.strategies) == 0 {
					continue
				}
				pair, timeframe := Split(symbol)
				if pair+timeframe == "" {
					continue
				}

				// get bars and run strategies on them
				bars := hist.Bars[symbol]
				for _, strat := range e.strategies {
					if new, ok := strat.Event(bars); ok && !events.Exists(new) {

						new.Pair = pair
						new.Timeframe = timeframe
						*events = append(*events, new)
						log.Printf("%s%s %s %s %.8f\n", new.Pair, new.Timeframe, EventTypes[new.Type], new.Text, new.Price)
					}
				}

			default:
				if e.running == false {
					log.Println("EVENTLISTENER stopped")
					return
				}
				time.Sleep(time.Second) // remove for faster action to event channel
			}
		}
	}()
}

// List ..
func (e *EventListener) List() {
	for _, strat := range e.strategies {
		fmt.Println(fmt.Sprintf("%T", strat)[6:])
	}
}

// Stop event listener
func (e *EventListener) Stop() {
	e.running = false
}

// Add strategy
func (e *EventListener) Add(s Strategy) {
	e.strategies = append(e.strategies, s)

	log.Println("EVENTLISTENER added", fmt.Sprintf("%T", s)[6:])
}

// Remove strategy
func (e *EventListener) Remove(s Strategy) error {
	for i, strat := range e.strategies {
		if strat == s {
			l := len(e.strategies) - 1
			e.strategies[i] = e.strategies[l]
			e.strategies = e.strategies[:l]

			log.Println("EVENTLISTENER removed", fmt.Sprintf("%T", s)[6:])
			return nil
		}
	}

	return errors.New("not found")
}
