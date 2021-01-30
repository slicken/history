package history

import (
	"errors"
	"fmt"
	"log"
	"time"
)

var errNotFound = errors.New("not found")

// EventListener is where you subscribe strategies too
type EventListener struct {
	running    bool
	strategies []Strategy
}

// Start event listener
func (e *EventListener) Start(data *Data, events *Events) {
	e.running = true

	log.Println("EVENTLISTENER started")
	go func() {
		for {

			select {
			case s, ok := <-data.C:
				if !ok || len(e.strategies) == 0 {
					continue
				}
				symbol, timeframe := Split(s)
				if symbol == "" || timeframe == "" {
					continue
				}

				// get bars and run strategies on them
				bars := data.Bars(symbol, timeframe)
				for _, strat := range e.strategies {
					if new, ok := strat.Event(bars); ok && !new.Exists(*events) {

						new.Symbol = symbol
						*events = append(*events, new)
						log.Printf("%s%s %s %s %.8f\n", symbol, timeframe, new.Type, new.Text, new.Price)
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

	return errNotFound
}
