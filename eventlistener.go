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

	log.Println("[EVENTLISTENER] started")

	go func() {
		for {

			select {
			case symbol, ok := <-hist.C:
				if !ok || len(e.strategies) == 0 {
					continue
				}
				// get bars and run all strategies on them
				bars := hist.Bars(symbol)
				for _, strategy := range e.strategies {
					if event, ok := strategy.Run(symbol, bars); ok {

						ok := events.Add(event)
						if !ok {
							log.Println("[EVENTLISTENER] coult not add event")
							continue
						}
						log.Printf("%s %s %s %.8f\n", event.Symbol, EventTypes[event.Type], event.Text, event.Price)
					}
				}

			default:
				if !e.running {
					return
				}
				time.Sleep(time.Second)
			}
		}
	}()
}

// List ..
func (e *EventListener) List() {
	for _, strategy := range e.strategies {
		fmt.Println(fmt.Sprintf("%T", strategy)[6:])
	}
}

// Stop event listener
func (e *EventListener) Stop() {
	e.running = false
	log.Println("[EVENTLISTENER] stopped!")
}

// Add strategy
func (e *EventListener) Add(s Strategy) {
	e.strategies = append(e.strategies, s)
	log.Println("[EVENTLISTENER] added", fmt.Sprintf("%T", s)[6:])
}

// Remove strategy
func (e *EventListener) Remove(s Strategy) error {
	for i, strategy := range e.strategies {
		if strategy == s {
			l := len(e.strategies) - 1
			e.strategies[i] = e.strategies[l]
			e.strategies = e.strategies[:l]

			log.Println("[EVENTLISTENER] removed", fmt.Sprintf("%T", s)[6:])
			return nil
		}
	}

	return errors.New("not found")
}
