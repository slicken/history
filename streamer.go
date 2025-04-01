package history

import (
	"time"
)

// Streamer interface for bars
type Streamer interface {
	<-chan Bars
}

// Streamer bars
func (bars Bars) Stream() <-chan Bars {
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

// StreamerDuration bars
func (bars Bars) StreamDuration(duration time.Duration) <-chan Bars {
	c := make(chan Bars, 1)

	if len(bars) == 0 {
		defer close(c)
		return c
	}

	go func() {
		for i := len(bars) - 1; i >= 0; i-- {
			time.Sleep(duration)
			c <- bars[i : len(bars)-1]
		}
		close(c)
	}()

	return c
}

// Stream bars
func (bars Bars) StreamInterval(start, end time.Time, interval time.Duration) <-chan Bars {
	c := make(chan Bars, 1)

	// check if we have bars
	if len(bars) == 0 {
		defer close(c)
		return c
	}

	// check if our time is within bars first last times, adjust if needed
	if ft := bars.FirstBar().T(); !ft.IsZero() {
		if start.IsZero() || start.Before(ft) {
			start = ft
		}
	}
	if lt := bars.LastBar().T(); !lt.IsZero() {
		if end.IsZero() || end.After(lt) {
			end = lt
		}
	}

	// adjust interval if needed
	if interval < mindur {
		interval = mindur
	}
	if interval > maxdur {
		interval = maxdur
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

// Stream bar
func (bars Bars) StreamBar() <-chan Bar {
	c := make(chan Bar, 1)

	if len(bars) == 0 {
		defer close(c)
		return c
	}

	go func() {
		for i := len(bars) - 1; i >= 0; i-- {
			c <- bars[i]
		}
		close(c)
	}()

	return c
}
