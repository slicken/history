package history

import (
	"sort"
	"time"
)

// Strategy interface using Bars type as Event condition
type Strategy interface {
	Event(Bars) (Event, bool)
}

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
	Pair      string
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

// IsBuy
func (e *Event) IsBuy() bool {
	if e.Type == 0 || e.Type == 2 || e.Type == 4 {
		return true
	}
	return false
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

// GetLast ..
func (ev Events) GetLast(symbol string) Event {
	var e Events

	for _, old := range ev {
		if symbol == old.Pair+old.Timeframe {
			e = append(e, old)
		}
	}

	if len(e) == 0 {
		return Event{}
	}

	return e[len(e)-1]
}

// Add event to events list
func (ev *Events) Add(e Event) bool {
	*ev = append(*ev, e)
	return true
}

// Delete the event from events list
func (ev *Events) Del(e Event) bool {
	for i, v := range *ev {
		if v == e {
			*ev = append((*ev)[:i], (*ev)[i+1:]...)
			return true
		}
	}

	return false
}

// Remove event index in slice
func (ev Events) RemoveIndex(index int) Events {
	return append(ev[:index], ev[index+1:]...)
}

// Map events
func (ev Events) Map() map[string]Events {
	m := make(map[string]Events)

	for _, e := range ev {
		// describe key
		key := e.Pair
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
		key := e.Pair
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

// // ListEvents ..
// func ListEvents(ev ...Event) Events {
// 	var v Events

// 	for _, e := range ev {
// 		v = append(v, e)
// 	}

// 	return v
// }
