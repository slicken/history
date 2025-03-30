package history

import (
	"sort"
	"time"
)

// Strategy interface using Bars to output Events
type Strategy interface {
	Run(string, Bars) (Event, bool)
}

// Event data for specific time and price
type Event struct {
	Symbol    string
	Pair      string
	Timeframe string
	Name      string
	Text      string
	Type      EventType
	Time      time.Time
	Price     float64
	Size      float64
}

// EventType
type EventType int

const (
	MARKET_BUY EventType = iota
	MARKET_SELL
	LIMIT_BUY
	LIMIT_SELL

	CLOSE_BUY
	CLOSE_SELL

	MODIFY
	NEWS
	OTHER
)

// EventTypes
var EventTypes = map[EventType]string{
	MARKET_BUY:  "MARKET_BUY",
	MARKET_SELL: "MARKET_SELL",
	LIMIT_BUY:   "LIMIT_BUY",
	LIMIT_SELL:  "LIMIT_SELL",
	CLOSE_BUY:   "CLOSE_BUY",
	CLOSE_SELL:  "CLOSE_SELL",
	MODIFY:      "MODIFY",
	NEWS:        "NEWS",
	OTHER:       "OTHER",
}

// NewEvent
func NewEvent(symbol string) Event {
	pair, tf := SplitSymbol(symbol)
	return Event{Symbol: symbol, Pair: pair, Timeframe: tf}
}

// Returns true if event is of any type buy
func (event *Event) IsBuy() bool {
	if event.Type == MARKET_BUY || event.Type == LIMIT_BUY || event.Type == CLOSE_BUY {
		return true
	}
	return false
}

// Returns true if event is of any type buy
func (event *Event) StringType() string {
	return EventTypes[event.Type]
}

/*

	------ EVENTS ------

*/

// Events type
type Events []Event

// Sort Events
func (events Events) Sort() Events {
	sort.SliceStable(events, func(i, j int) bool {
		return events[i].Price > events[j].Price
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
		if event.Time == (*events)[i].Time && event.Price == (*events)[i].Price {
			return false
		}
	}

	*events = append(*events, event)
	return true
}

// Delete event from events list
func (events *Events) Del(event Event) bool {
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
