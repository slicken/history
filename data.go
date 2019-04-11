package history

import (
	"errors"
	"log"
	"math"
	"sync"
	"time"
)

var (
	errNoHist = errors.New("no history")
	errNoBars = errors.New("no bars")
)

// Data is ExchangeData
type Data struct {
	History []*History
	sync.RWMutex

	// Plug diffrent downloaders
	Downloader
	update bool
	// C notify channel when a history has been updated
	C chan string
}

// Downloader interface to plug diffrent data providers
type Downloader interface {
	// Download downloads and returns bars
	Download(symbol, timeframe string, limit int) (Bars, error)
}

// history returns bars wrapped with settings
func (d *Data) history(symbol, timeframe string) (*History, error) {
	d.RLock()
	defer d.RUnlock()

	for _, hist := range d.History {
		if hist.Symbol == symbol && hist.Timeframe == timeframe {
			return hist, nil
		}
	}

	return nil, errNoHist
}

// Bars is returns bars slice of given symbol
func (d *Data) Bars(symbol, timeframe string) Bars {
	hist, err := d.history(symbol, timeframe)
	if err != nil {
		return nil
	}

	hist.RLock()
	defer hist.RUnlock()
	return hist.Bars
}

// TimeSpan returns all histories for given times
func (d *Data) TimeSpan(start, end time.Time) *Data {
	var wg sync.WaitGroup

	for _, h := range d.History {

		wg.Add(1)
		go func(h *History) {
			defer wg.Done()

			h.Bars = h.Bars.TimeSpan(start, end)
		}(h)
	}

	wg.Wait()
	return d
}

// Period returns minimum period of historys
func (d *Data) Period() time.Duration {
	var min = minDur

	d.RLock()
	defer d.RUnlock()
	for _, hist := range d.History {
		period := hist.Bars.Period()
		if min == minDur || period < min {
			min = period
		}
	}

	return min
}

// FirstTime returns minimum period of historys
func (d *Data) FirstTime() time.Time {
	var first = time.Time{}

	d.RLock()
	defer d.RUnlock()
	for _, hist := range d.History {

		func(h *History) {

			h.RLock()
			defer h.RUnlock()
			if len(h.Bars) == 0 {
				return
			}

			t := h.Bars[len(h.Bars)-1].Time
			if first.IsZero() || t.Before(first) {
				first = t
			}

		}(hist)
	}

	return first
}

// LastTime returns minimum period of historys
func (d *Data) LastTime() time.Time {
	var last = time.Time{}

	d.RLock()
	defer d.RUnlock()
	for _, hist := range d.History {

		func(h *History) {

			h.RLock()
			defer h.RUnlock()
			if len(h.Bars) == 0 {
				return
			}

			t := h.Bars[0].Time
			if last.IsZero() || t.After(last) {
				last = t
			}

		}(hist)
	}

	return last
}

// Delete adds new history saftly
func (d *Data) Delete(symbol, timeframe string) error {

	d.Lock()
	defer d.Unlock()
	for i, hist := range d.History {
		if hist.Symbol == symbol && hist.Timeframe == timeframe {

			select {
			case _, open := <-hist.update:
				if open {
					close(hist.update)
				}
			default:
			}
			// d.History = append(d.History[:i], d.History[:i+1]...)
			l := len(d.History) - 1
			d.History[i] = d.History[l]
			d.History = d.History[:l]
			return nil
		}
	}

	return errNoHist
}

// Load loads symbols from slice defined with symboltf strings
func (d *Data) Load(symbols []string) error {

	// var wg sync.WaitGroup
	for _, s := range symbols {
		if s == "" { //						---------------- fix so we can remove this --------------
			continue
		}
		symbol, timeframe := Split(s)
		if symbol == "" || timeframe == "" {
			log.Printf("could not load %s. invalid input\n", symbol+timeframe)
			continue
		}
		// wg.Add(1)
		// go func(symbol, timeframe string) {
		// 	defer wg.Done()

		bars, _ := ReadBars(symbol, timeframe)
		d.Add(symbol, timeframe, &bars)
		// }(symbol, timeframe)
	}

	// wg.Wait()
	return nil
}

// Add adds new history saftly
func (d *Data) Add(symbol, timeframe string, bars *Bars) error {

	// create or get history
	hist, err := d.history(symbol, timeframe)
	if err != nil {
		hist = new(History)
		hist.Symbol = symbol
		hist.Timeframe = timeframe

		d.Lock()
		d.History = append(d.History, hist)
		// increase cap by +1
		c := make(chan string, len(d.History))
		// copy values to new channel
		for len(d.C) > 0 {
			select {
			case v, _ := <-d.C:
				c <- v
			default:
			}
		}
		d.C = c
		d.Unlock()
	}

	// add or merge bars
	if len(*bars) != 0 {
		update := "updated"
		if err != nil {
			update = "loaded"
		}
		if len(hist.Bars) != 0 {
			bars = merge(&hist.Bars, bars)
		}

		hist.Lock()
		hist.Bars = *bars
		// save to file
		if err = SaveBars(symbol, timeframe, &hist.Bars); err != nil {
			log.Printf("could not save %s%s bars: %v\n", hist.Symbol, hist.Timeframe, err)
		}
		hist.Unlock()

		log.Printf("%s%s %s\n", hist.Symbol, hist.Timeframe, update)

		// notify data updates channel
		select {
		case d.C <- (symbol + timeframe):
		default:
		}
	}

	// kick off updateHandler
	if d.update {
		hist.updateHandler(d)
	}

	return nil
}

// Update enabled history updates on each
// if alredy enabled it will force download all histories
func (d *Data) Update(enabled bool) {
	d.Lock()
	defer d.Unlock()
	var limit int
	var old bool
	old = d.update
	d.update = enabled

	for _, hist := range d.History {

		switch enabled {
		case true:
			hist.updateHandler(d)

			if old == false {
				continue
			}
			limit = maxlimit
			if len(hist.Bars) > 0 {
				limit = int(math.Max(float64(calcLimit(hist.Bars[0].Time, hist.Timeframe)), 1))
			}
			hist.update <- limit

		case false:
			// close updates channel for history
			// this will return from updates loop
			select {
			case _, open := <-hist.update:
				if open {
					close(hist.update)
				}
			default:
			}

		}
	}
}
