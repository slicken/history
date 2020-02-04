package history

import (
	"sync"
	"time"
)

// --- add streamer delay ---------------------- not done -------------------------

// Streamer streams data, historys or bars
type Streamer interface {
	Stream(time.Time, time.Time, time.Duration) <-chan interface{}
}

// Streamer bars
func (bars Bars) Streamer() <-chan Bars {
	c := make(chan Bars, 1)

	if len(bars) == 0 {
		defer close(c)
		return c
	}

	go func() {
		for i := len(bars) - 1; i >= 0; i-- {
			c <- bars[i : len(bars)-1]
		}
		close(c)
	}()

	return c
}

// Stream bars
func (bars Bars) Stream(start, end time.Time, interval time.Duration) <-chan Bars {
	c := make(chan Bars, 1)

	// check if we have bars
	if len(bars) == 0 {
		defer close(c)
		return c
	}
	// check if our time is within bars first last times, adjust if needed
	if first := bars[len(bars)-1].Time; !first.IsZero() {
		if start.IsZero() || start.Before(first) {
			start = first
		}
	}
	if last := bars[0].Time; !last.IsZero() {
		if end.IsZero() || end.After(last) {
			end = last
		}
	}
	// adjust interval if needed
	if interval < minDur {
		interval = minDur
	}
	if interval > maxDur {
		interval = maxDur
	}

	go func() {
		// time value witch we will increase on loop
		dt := start
		for dt.Before(end) {
			// add looping interval to time
			dt = dt.Add(interval)
			// get bars from timespan
			stream := bars.TimeSpan(start, dt)

			c <- stream
		}
		close(c)

	}()

	return c
}

// Stream all Data
func (data *Data) Stream(start, end time.Time, interval time.Duration) <-chan *Data {
	c := make(chan *Data, 1)

	// check if our time is within bars first last times, adjust if needed
	if first := data.FirstTime(); !first.IsZero() {
		if start.IsZero() || start.Before(first) {
			start = first
		}
	}
	if last := data.LastTime(); !last.IsZero() {
		if end.IsZero() || end.After(last) {
			end = last
		}
	}
	// adjust interval if needed
	if interval < minDur {
		interval = minDur
	}
	if interval > maxDur {
		interval = maxDur
	}

	go func() {
		// time value witch we will increase on loop
		dt := start
		var wg sync.WaitGroup
		for dt.Before(end) {
			// create new data array
			stream := new(Data)
			// add looping interval to time
			dt = dt.Add(interval)
			for _, h := range data.History {

				wg.Add(1)
				go func(h *History, wg *sync.WaitGroup) {
					defer wg.Done()

					// get history from timespan
					// create new prevents from copying lock
					hist := new(History)
					hist.Symbol = h.Symbol
					hist.Timeframe = h.Timeframe
					hist.Bars = h.Bars.TimeSpan(start, end)

					stream.Lock()
					stream.History = append(stream.History, hist)
					stream.Unlock()
				}(h, &wg)
			}

			wg.Wait()
			c <- stream
		}
		close(c)
	}()

	return c
}
