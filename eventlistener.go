package history

import (
	"errors"
	"fmt"
	"log"
)

// EventListener is where you subscribe strategies too
type EventListener struct {
	strategies []Strategy
	running    bool
}

// Start event listener
func (e *EventListener) Start(hist *History, events *Events) error {
	if e.running {
		return errors.New("alredy running")
	}
	e.running = true
	log.Println("[EVENTLISTENER] started")

	go func() {
		for {

			select {
			case symbol := <-hist.C:
				if len(e.strategies) == 0 {
					continue
				}
				// run all strategies on bars
				bars := hist.GetBars(symbol)
				for _, strategy := range e.strategies {
					if event, ok := strategy.Run(symbol, bars); ok {

						ok := events.Add(event)
						if !ok {
							continue
						}
						// preform action
						log.Printf("%s %s %s %s %.8f\n", event.Symbol, EventTypes[event.Type], event.Name, event.Text, event.Price)
					}
				}

			default:
				if !e.running {
					log.Println("[EVENTLISTENER] stopped")
					return
				}
			}

		}
	}()
	return nil
}

// List added strategies
func (e *EventListener) List() {
	for _, strategy := range e.strategies {
		_name := fmt.Sprintf("%T", strategy)[6:]
		log.Println("[EVENTLISTENER]", _name)
	}
}

// Stop event listener
func (e *EventListener) Stop() error {
	if !e.running {
		return errors.New("not running")
	}
	e.running = false
	return nil
}

// Add strategy
func (e *EventListener) Add(strategy Strategy) error {
	name := fmt.Sprintf("%T", strategy)[6:]

	for _, _strategy := range e.strategies {
		_name := fmt.Sprintf("%T", _strategy)[6:]
		if name == _name {
			return errors.New("alredy exist")
		}
	}
	e.strategies = append(e.strategies, strategy)
	log.Println("[EVENTLISTENER] added", name)
	return nil
}

// Remove strategy
func (e *EventListener) Remove(strategy Strategy) error {
	name := fmt.Sprintf("%T", strategy)[6:]

	for i, _strategy := range e.strategies {
		if _strategy == strategy {
			l := len(e.strategies) - 1
			e.strategies[i] = e.strategies[l]
			e.strategies = e.strategies[:l]

			log.Println("[EVENTLISTENER] removed", name)
			return nil
		}
	}

	return errors.New("not found")
}
