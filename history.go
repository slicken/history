/*
	----------------------------------------------
	history.go is the main package of history repo
	----------------------------------------------
	Bars					ohlcv pair+timeframe data (bars/candlestick)
	Tick					tickdata if enabled
	Downloader Interface
	C						pair+timeframe when we got new data
	Portfolio 				backtesting
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

// GetTick returns tick channel              -
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
		// delete all timeframes of symbol if missing
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
		msg = "loaded"

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

	// notify data.C that we have updated symbol (not when loading)
	if len(b) > 0 { // || msg == "loaded" {
		select {
		case h.C <- (symbol):
		default:
		}
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

				// check how many new bars there is from last bars time
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
	tries := 0
	// try download new bars
	for tries < maxtries {
		bars, err = h.GetKlines(sym, tf, limit)
		if err != nil {
			tries++
			time.Sleep(2 * time.Second)
			continue
		}
		// since we always "gets" the current bar witch is not finish, we dont want to save that
		if 2 > len(bars) {
			return nil
		}
		// check if lastbar time is valid. if not, unload
		if time.Now().Add(2 * -bars.Period()).After(bars[0].T()) {
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

	log.Printf("failed to download %d bars for %s: %v\n", limit, symbol, err)
	return err
}
