package history

import (
	"sort"
	"time"
)

// EventType ..
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

var EventTypes = map[EventType]string{
	0: "MARKET_BUY",
	1: "MARKET_SELL",
	2: "LIMIT_BUY",
	3: "LIMIT_SELL",
	4: "CLOSE_BUY",
	5: "CLOSE_SELL",
	6: "MODIFY",
	7: "NEWS",
	8: "OTHER",
}

// Event data for specific time and price
type Event struct {
	Symbol    string
	Timeframe string
	Title     string
	Text      string
	Type      EventType
	Time      time.Time
	Price     float64
	Size      float64
}

// Add ..
func (e *Event) Add(typ EventType, title string, time time.Time, price float64) {
	e.Type = typ
	e.Title = title
	e.Time = time
	e.Price = price
}

// Events type
type Events []Event

// Sort Events
func (ev Events) Sort() {
	sort.SliceStable(ev, func(i, j int) bool {
		return ev[i].Price > ev[j].Price
	})
	return
}

// Exists ..
func (ev Events) Exists(e Event) bool {
	for _, old := range ev {
		if e.Time == old.Time && e.Price == old.Price {
			return true
		}
	}

	return false
}

// Map events
func (ev Events) Map() map[string]Events {
	m := make(map[string]Events)

	for _, e := range ev {
		// describe key
		key := e.Symbol
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

// MapEvents ..
func MapEvents(ev ...Event) map[string]Events {
	m := make(map[string]Events)

	for _, e := range ev {
		// describe key
		key := e.Symbol
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

// ListEvents ..
func ListEvents(ev ...Event) Events {
	var v Events

	for _, e := range ev {
		v = append(v, e)
	}

	return v
}
