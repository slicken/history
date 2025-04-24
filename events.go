package history

import (
	"sort"
	"time"
)

// Event data for specific time and price
type Event struct {
	Symbol string    // Trading symbol (e.g. "BTC/USDT1h")
	Name   string    // Event name (e.g. strategy name)
	Text   string    // Additional event information
	Type   EventType // Type of event
	Time   time.Time // When event occurred
	Price  float64   // Price at event
	Size   float64   // Position size
}

// EventType
type EventType int

const (
	MARKET_BUY  EventType = iota // Market buy order
	MARKET_SELL                  // Market sell order
	LIMIT_BUY                    // Limit buy order
	LIMIT_SELL                   // Limit sell order
	STOP_BUY                     // Stop buy order (can be used for breakout or stop loss buy)
	STOP_SELL                    // Stop sell order (can be used for stop loss or take profit)
	CLOSE                        // Close position event
	NEWS                         // News event
	OTHER                        // Other custom events
	FORECAST                     // Forecast event
)

// EventTypes
var EventTypes = map[EventType]string{
	MARKET_BUY:  "MARKET_BUY",
	MARKET_SELL: "MARKET_SELL",
	LIMIT_BUY:   "LIMIT_BUY",
	LIMIT_SELL:  "LIMIT_SELL",
	STOP_BUY:    "STOP_BUY",
	STOP_SELL:   "STOP_SELL",
	CLOSE:       "CLOSE",
	NEWS:        "NEWS",
	OTHER:       "OTHER",
	FORECAST:    "FORECAST",
}

// NewEvent creates a new event for a symbol
func NewEvent(symbol string) Event {
	return Event{Symbol: symbol}
}

/*

	------ EVENTS ------

*/

// Events type
type Events []Event

// Sort Events
func (events Events) Sort() Events {
	sort.SliceStable(events, func(i, j int) bool {
		return events[i].Time.Before(events[j].Time)
	})
	return events
}

// Return events for given symbol
func (events Events) Symbol(symbol string) Events {
	var ev Events
	for _, event := range events {
		if symbol == event.Symbol {
			ev = append(ev, event)
		}
	}
	return ev
}

// Exists
func (events Events) Exists(event Event) bool {
	for _, old := range events {
		if event.Time == old.Time && event.Price == old.Price {
			return true
		}
	}

	return false
}

// FirstEvent in dataset
func (events Events) FirstEvent() Event {
	if 1 > len(events) {
		return Event{}
	}

	return events[len(events)-1]
}

// LastEvent in dataset
func (events Events) LastEvent() Event {
	if 1 > len(events) {
		return Event{}
	}

	return events[0]
}

// Find Event for the given time
func (events Events) Find(dt time.Time) (n int, e Event) {
	if 1 > len(events) {
		return -1, Event{}
	}
	if events.FirstEvent().Time.After(dt) || events.LastEvent().Time.Before(dt) {
		return -1, Event{}
	}

	for i, event := range events {
		if event.Time == dt {
			return i, event
		}
	}

	return -1, Event{}
}

// Add event to events list
// Note: Important to have a price
func (events *Events) Add(event Event) bool {
	// check if event exist
	if event.Symbol == "" || event.Price == 0 {
		return false
	}
	for i := len(*events) - 1; i >= 0; i-- {
		// Consider an event duplicate only if it has the same symbol, time, price, and type
		if event.Symbol == (*events)[i].Symbol &&
			event.Time == (*events)[i].Time &&
			event.Price == (*events)[i].Price &&
			event.Type == (*events)[i].Type {
			return false
		}
	}

	*events = append(*events, event)
	return true
}

// Delete event from events list
func (events *Events) Delete(event Event) bool {
	for i, v := range *events {
		if v == event {
			*events = append((*events)[:i], (*events)[i+1:]...)
			return true
		}
	}

	return false
}

// Map events
func (events Events) Map() map[string]Events {
	m := make(map[string]Events)

	for _, event := range events {
		key := event.Symbol
		if key == "" {
			key = "unknown"
		}
		if _, ok := m[key]; !ok {
			m[key] = Events{event}
			continue
		}
		m[key] = append(m[key], event)
	}

	return m
}

// MapEvents
func MapEvents(events ...Event) map[string]Events {
	m := make(map[string]Events)

	events = Events(events)
	for _, event := range events {
		key := event.Symbol
		if key == "" {
			key = "unknown"
		}
		if _, ok := m[key]; !ok {
			m[key] = Events{event}
			continue
		}
		m[key] = append(m[key], event)
	}

	return m
}

// ListEvents
func ListEvents(ev ...Event) Events {
	var v Events

	for _, e := range ev {
		v = append(v, e)
	}

	return v
}

// Remove event index in slice
func (ev Events) RemoveIndex(index int) Events {
	return append(ev[:index], ev[index+1:]...)
}
