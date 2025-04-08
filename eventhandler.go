package history

import (
	"errors"
	"fmt"
	"log"
	"sync"
)

// EventHandler manages strategies and event handling
type EventHandler struct {
	handlers   map[EventType][]EventCallback
	strategies []Strategy
	running    bool
	*sync.RWMutex
}

// EventCallback is a function type that handles events
type EventCallback func(Event) error

// NewEventHandler creates a new event handler
func NewEventHandler() *EventHandler {
	return &EventHandler{
		handlers:   make(map[EventType][]EventCallback),
		strategies: make([]Strategy, 0),
		running:    false,
	}
}

// Subscribe adds a callback function for a specific event type
func (eh *EventHandler) Subscribe(eventType EventType, callback EventCallback) {
	eh.Lock()
	defer eh.Unlock()

	if _, exists := eh.handlers[eventType]; !exists {
		eh.handlers[eventType] = make([]EventCallback, 0)
	}
	eh.handlers[eventType] = append(eh.handlers[eventType], callback)
}

// Unsubscribe removes a callback function for a specific event type
func (eh *EventHandler) Unsubscribe(eventType EventType, callback EventCallback) {
	eh.Lock()
	defer eh.Unlock()

	if callbacks, exists := eh.handlers[eventType]; exists {
		for i, cb := range callbacks {
			if fmt.Sprintf("%p", cb) == fmt.Sprintf("%p", callback) {
				eh.handlers[eventType] = append(callbacks[:i], callbacks[i+1:]...)
				break
			}
		}
	}
}

// Handle processes an event by calling all registered callbacks for its type
func (eh *EventHandler) Handle(event Event) error {
	eh.RLock()
	callbacks, exists := eh.handlers[event.Type]
	eh.RUnlock()

	if !exists {
		return nil // No handlers for this event type
	}

	for _, callback := range callbacks {
		if err := callback(event); err != nil {
			return fmt.Errorf("event handler error: %v", err)
		}
	}

	return nil
}

// HandleEvents processes multiple events
func (eh *EventHandler) HandleEvents(events Events) error {
	for _, event := range events {
		if err := eh.Handle(event); err != nil {
			return err
		}
	}
	return nil
}

// Clear removes all event handlers
func (eh *EventHandler) Clear() {
	eh.Lock()
	eh.handlers = make(map[EventType][]EventCallback)
	eh.Unlock()
}

// Start event handler
func (eh *EventHandler) Start(hist *History, events *Events) error {
	eh.Lock()
	if eh.running {
		eh.Unlock()
		return errors.New("already running")
	}
	eh.running = true
	eh.Unlock()

	log.Println("[EVENTHANDLER] started")

	// Drain any existing signals in the channel
	for len(hist.C) > 0 {
		<-hist.C
	}

	go func() {
		for {
			select {
			case symbol := <-hist.C:
				if len(eh.strategies) == 0 {
					continue
				}
				// run all strategies on bars
				bars := hist.GetBars(symbol)
				for _, strategy := range eh.strategies {
					if event, ok := strategy.OnBar(symbol, bars); ok {
						ok := events.Add(event)
						if !ok {
							continue
						}

						// Handle the event
						if err := eh.Handle(event); err != nil {
							log.Printf("Error handling event: %v", err)
						}
					}
				}

			default:
				eh.RLock()
				if !eh.running {
					eh.RUnlock()
					log.Println("[EVENTHANDLER] stopped")
					return
				}
				eh.RUnlock()
			}
		}
	}()
	return nil
}

// Stop event handler
func (eh *EventHandler) Stop() error {
	eh.Lock()
	defer eh.Unlock()

	if !eh.running {
		return errors.New("not running")
	}
	eh.running = false
	return nil
}

// AddStrategy adds a strategy to the handler
func (eh *EventHandler) AddStrategy(strategy Strategy) error {
	eh.Lock()
	defer eh.Unlock()

	name := fmt.Sprintf("%T", strategy)[6:]

	for _, _strategy := range eh.strategies {
		if name == fmt.Sprintf("%T", _strategy)[6:] {
			return errors.New("strategy already exists")
		}
	}
	eh.strategies = append(eh.strategies, strategy)
	log.Println("[EVENTHANDLER] added strategy:", name)
	return nil
}

// RemoveStrategy removes a strategy from the handler
func (eh *EventHandler) RemoveStrategy(strategy Strategy) error {
	eh.Lock()
	defer eh.Unlock()

	name := fmt.Sprintf("%T", strategy)[6:]

	for i, _strategy := range eh.strategies {
		if _strategy == strategy {
			l := len(eh.strategies) - 1
			eh.strategies[i] = eh.strategies[l]
			eh.strategies = eh.strategies[:l]

			log.Println("[EVENTHANDLER] removed strategy:", name)
			return nil
		}
	}

	return errors.New("strategy not found")
}

// ListStrategies lists all added strategies
func (eh *EventHandler) ListStrategies() {
	eh.RLock()
	defer eh.RUnlock()

	for _, strategy := range eh.strategies {
		_name := fmt.Sprintf("%T", strategy)[6:]
		log.Println("[EVENTHANDLER] strategy:", _name)
	}
}
