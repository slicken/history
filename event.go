package history

import (
	"sort"
	"time"
)

// Event data for specific time and price
type Event struct {
	Symbol    string
	Timeframe string
	Type      string // Buy / Sell
	Text      string
	Value     float64
	Time      time.Time
	Price     float64
}

// Add ..
func (e *Event) Add(typ, text string, time time.Time, price float64) {
	e.Type = typ
	e.Text = text
	e.Time = time
	e.Price = price
}

// // Exists ..
// func (e *Event) Exists(events Events) bool {
// 	for _, old := range events {
// 		if e.Time == old.Time && e.Price == old.Price {
// 			return true
// 		}
// 	}

// 	return false
// }

// // WithinBars ..
// func (e *Event) WithinBars(events Events, limit int) bool {

// 	for _, o := range events {
// 		if e.Symbol == o.Symbol && e.Timeframe == o.Timeframe {
// 			tf := time.Duration(Tf(o.Timeframe)) * time.Minute
// 			diff := o.Time.Sub(e.Time)
// 			bbw := -int(diff / tf)
// 			if limit > bbw {
// 				return true
// 			}
// 		}
// 	}

// 	return false
// }

// Events type
type Events []Event

// Sort Events
func (ev Events) Sort() {
	sort.SliceStable(ev, func(i, j int) bool {
		return ev[i].Value > ev[j].Value
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
