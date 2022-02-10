/*
	------------------------------------------------------------------------------
	history.go is the main package of history repo
	------------------------------------------------------------------------------
	History 				main struct holds all history data with settings
	EventListener 			is loaded with strategies and is looking at History (for real time strategies)
	Portfolio 				for tracking gains when backtesting

	History.Bars[symbol]	contains tohlcv (time,open,high,low,close,volume)
	History.Tick[symbol]	tickdata from ws::    								!! NOT IMPLANMENTED !!
	History.Update(true)	if you want to keep it running looking for new bars
	History.Downloader 		Interface where you connect you exchange downloads spitting out Bars
	History.C				notify when symbol (pair+timeframe) get new data (bars)
												 .slk.prod.21
*/

package history

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

var (
	errNoHist   = errors.New("no history")
	errNoBars   = errors.New("no bars")
	errNotFound = errors.New("not found")
)

// History maintaner
type History struct {
	Bars   map[string]Bars
	Tick   map[string]chan float64
	update bool
	// C notify channel when we got now bars for a history (symbol)
	C chan string
	// Plug diffrent downloaders
	Downloader

	sync.RWMutex
}

// Downloader interface plugs functions that download bars
type Downloader interface {
	GetKlines(pair, timeframe string, limit int) (Bars, error)
}

// GetBars returns bars saftly
func (h *History) GetBars(pair, timeframe string) Bars {
	h.RLock()
	defer h.RUnlock()

	if bars, ok := h.Bars[pair+timeframe]; ok {
		return bars
	}

	return Bars{}
}

// GetTick returns tick channel
func (h *History) GetTick(pair, timeframe string) (chan float64, error) {
	h.RLock()
	defer h.RUnlock()

	if tc, ok := h.Tick[pair+timeframe]; ok {
		return tc, nil
	}

	return nil, errors.New("not found")
}

// MinPeriod returns minimum period of historys
func (h *History) MinPeriod() time.Duration {
	h.RLock()
	defer h.RUnlock()

	var min = mindur
	for _, bars := range h.Bars {
		period := bars.Period()
		if min == mindur || period < min {
			min = period
		}
	}

	return min
}

// FirstTime returns minimum period of historys
func (h *History) FirstTime() time.Time {
	h.RLock()
	defer h.RUnlock()

	var v = time.Time{}
	for _, bars := range h.Bars {

		t := bars.FirstBar().T()
		if v.IsZero() || t.Before(v) {
			v = t
		}
	}

	return v
}

// LastTime returns minimum period of historys
func (h *History) LastTime() time.Time {
	h.RLock()
	defer h.RUnlock()

	var v time.Time
	for _, bars := range h.Bars {

		t := bars.LastBar().T()
		if v.IsZero() || t.After(v) {
			v = t
		}
	}

	return v
}

// Limit the data for all history
func (h *History) Limit(length int) *History {
	h.Lock()
	defer h.Unlock()

	for symbol := range h.Bars {
		if length >= len(h.Bars[symbol]) {
			continue
		}
		h.Bars[symbol] = h.Bars[symbol][:length]
	}

	return h
}

// TimeSpan returns all historys for given start to end time
func (h *History) TimeSpan(start, end time.Time) *History {
	var wg sync.WaitGroup

	for symbol := range h.Bars {

		wg.Add(1)
		go func(sym string, wg *sync.WaitGroup) {
			defer wg.Done()

			h.Lock()
			h.Bars[sym] = h.Bars[sym].TimeSpan(start, end)
			h.Unlock()
		}(symbol, &wg)
	}

	wg.Wait()
	return h
}

// Unload removes a history from data struct gracefully
func (h *History) Unload(symbol string) error {
	h.Lock()
	defer h.Unlock()

	sym, tf := Split(symbol)
	if tf == "" {
		// delete all timeframes of pair if symbol if missing timeframe
		for v := range h.Bars {
			s, _ := Split(v)
			if s == sym {
				delete(h.Bars, v)
			}
		}
	} else {
		delete(h.Bars, symbol)
	}

	log.Println(symbol, "unloaded")
	return nil
}

// Load loads symbols from slice defined with symboltf strings
func (h *History) Load(symbols ...string) error {
	var wg sync.WaitGroup

	for _, s := range symbols {
		if sym, tf := Split(s); sym == "" || tf == "" {
			log.Printf("could not split %s. got symbol=%s timeframe=%s\n", s, sym, tf)
			continue
		}

		wg.Add(1)
		go func(symbol string, wg *sync.WaitGroup) {
			defer wg.Done()

			// we dont check for errors, cause we use add ether way
			bars, _ := ReadBars(symbol)
			h.Add(symbol, bars)
		}(s, &wg)
	}

	wg.Wait()
	return nil
}

// Add new history safely to datastruct
func (h *History) Add(symbol string, bars Bars) error {
	h.Lock()
	defer h.Unlock()

	var msg string

	if len(h.Bars) == 0 {
		h.Bars = make(map[string]Bars, 0)
	}

	b, ok := h.Bars[symbol]
	if !ok {
		msg = "loading"

		h.Bars[symbol] = bars
		// increase cap by +1
		c := make(chan string, len(h.Bars))
		// copy values to new channel
		for len(h.C) > 0 {
			select {
			case v, _ := <-h.C:
				c <- v
			default:
			}
		}
		h.C = c
	} else if len(b) == len(bars) && b.LastBar() == bars.LastBar() {
		// nothing new
		return errors.New("nothing new")
	} else {
		// save bars
		msg = fmt.Sprintf("added %d bars", len(bars))
		if err := SaveBars(symbol, bars); err != nil {
			log.Printf("could not save %s bars: %v\n", symbol, err)
		}
	}
	if len(bars) == 0 {
		return nil
	}

	// update history
	h.Bars[symbol] = merge(b, bars)

	// delete if total bars to small
	if 2 > len(h.Bars[symbol]) {
		delete(h.Bars, symbol)
		return errors.New("history to short")
	}

	log.Println(symbol, msg)

	// notify data.C that we have bars
	select {
	case h.C <- (symbol):
	default:
	}

	return nil
}

// Update enables/disables new data updates
func (h *History) Update(enabled bool) {
	h.Lock()
	h.update = enabled
	h.Unlock()

	var done bool
	go func(w *bool) {
		for {
			h.RLock()
			enabled = h.update
			h.RUnlock()
			if !enabled {
				done = true
				return
			}

			h.RLock()
			var wg sync.WaitGroup
			for symbol := range h.Bars {
				limit := maxlimit

				// calc how many new bars we can download from our last bar
				if len(h.Bars[symbol]) > 0 {
					limit = calcLimit(h.Bars[symbol].LastBar().T(), h.Bars[symbol].Period())
					if limit > maxlimit {
						limit = maxlimit
					}
				}

				if limit > 1 {
					wg.Add(1)
					go h.getSeries(symbol, limit, &wg)
				}
			}
			h.RUnlock()

			wg.Wait()
			done = true

			time.Sleep(time.Second)
		}
	}(&done)

	// wait for first update
	for !done {
		time.Sleep(100 * time.Microsecond)
	}
}

// getSeries downloads and updates history
func (h *History) getSeries(symbol string, limit int, wg *sync.WaitGroup) error {
	defer wg.Done()

	sym, tf := Split(symbol)

	var err error
	var bars Bars

	bars, err = h.GetKlines(sym, tf, limit)
	if err != nil {
		log.Printf("failed to download %d bars for %s: %v\n", limit, symbol, err)
		time.Sleep(2 * time.Minute)
		return err
	}
	// since we always get the current bar witch is not finish, we dont want to save that
	if 2 > len(bars) {
		return nil
	}
	// check if lastbar time is fresh, if not then delete symbol
	if time.Now().Add(2 * -bars.Period()).After(bars.LastBar().T()) {
		h.Lock()
		delete(h.Bars, symbol)
		h.Unlock()
		log.Println(symbol, "outdated")
		return nil
	}
	// add to history
	h.Add(symbol, bars[1:])
	return nil
}
