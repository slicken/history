/*
	-------------------------------------------------------------------------------------------
	history.go is the main package of history repo
	-------------------------------------------------------------------------------------------
	History 						main struct holds all history data (bars) with settings
	EventListener 					is loaded with strategies and is looking at History (for real time strategies)
	Portfolio 						for tracking gains when backtesting

	History.bars[symbol]Bars		Bars = []Bar
	History.Tick[symbol]float64		interface for Tickdata from ws::
	History.Update(true)			if true, it will update if new bars
	History.Downloader 				Interface where you connect you downloader and stores Bars data
	History.C						notify when symbol (pair+timeframe) get new data (bars)

	Important things to be aware of
	Tf		 	= Timeframe
	Pair		= quote asset + base asset
	Symbol		= Pair + Tf
	---
	Check '/examples/main.go' how to use package and examples

	'''hist := new(history.History)'''
	If you want to use your own download function create a function named
	'''type Binance struct{}
       func (e Binance) DLBars(sym, tf string, limit int) (history.Bars, error)'''
	connect the download function
	'''hist.Downloader = &Downloader{}'''
	Update if new bars automaticly
	'''hist.Update(true)'''
	Get some symbols bars data in reverse
	'''hist.Get(symbol).Reverse()'''

*/

package history

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

// History maintaner
type History struct {
	bars   map[string]Bars
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

// Bars returns bars saftly
func (h *History) Bars(symbol string) Bars {
	h.RLock()
	defer h.RUnlock()

	bars, ok := h.bars[symbol]
	if !ok {
		return Bars{}
	}

	return bars
}

// Returns a map of all bars
func (h *History) Map() map[string]Bars {
	return h.bars
}

// MinPeriod returns minimum period of historys
func (h *History) MinPeriod() time.Duration {
	h.RLock()
	defer h.RUnlock()

	var min = mindur
	for _, bars := range h.bars {
		period := bars.Period()
		if min == mindur || period < min {
			min = period
		}
	}

	return min
}

// FirstTime returns first time in all historys
func (h *History) FirstTime() time.Time {
	h.RLock()
	defer h.RUnlock()

	var v = time.Time{}
	for _, bars := range h.bars {

		t := bars.FirstBar().T()
		if v.IsZero() || t.Before(v) {
			v = t
		}
	}

	return v
}

// LastTime returns latest time in all historys
func (h *History) LastTime() time.Time {
	h.RLock()
	defer h.RUnlock()

	var v time.Time
	for _, bars := range h.bars {

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

	for symbol := range h.bars {
		if length >= len(h.bars[symbol]) {
			continue
		}
		h.bars[symbol] = h.bars[symbol][:length]
	}

	return h
}

// LimiTimeSpan the data for specified data time intervalls
func (h *History) LimitTimeSpan(start, end time.Time) *History {
	// h.Lock()
	// defer h.Unlock()
	var wg sync.WaitGroup

	for symbol := range h.bars {

		wg.Add(1)
		go func(sym string, wg *sync.WaitGroup) {
			defer wg.Done()

			h.Lock()
			h.bars[sym] = h.bars[sym].TimeSpan(start, end)
			h.Unlock()
		}(symbol, &wg)
	}

	wg.Wait()
	return h
}

// need fixing
// Unload removes symbol bars from data struct gracefully
func (h *History) Unload(symbol string) error {
	h.Lock()
	defer h.Unlock()

	pair, tf := SplitSymbol(symbol)
	if tf == "" {
		// delete all timeframes of pair if symbol if missing timeframe
		for _symbol := range h.bars {
			_pair, _ := SplitSymbol(_symbol)
			if _pair == pair {
				delete(h.bars, _symbol)
			}
		}
	} else {
		delete(h.bars, symbol)
	}

	log.Println(symbol, "unloaded")
	return nil
}

// Load loads symbols from slice defined with symboltf strings
func (h *History) Load(symbols ...string) error {
	var wg sync.WaitGroup

	for _, symbol := range symbols {

		wg.Add(1)
		go func(symbol string, wg *sync.WaitGroup) {
			defer wg.Done()

			// we add ether way
			bars, _ := ReadBars(symbol)
			h.Add(symbol, bars)
		}(symbol, &wg)
	}

	wg.Wait()
	return nil
}

// Add new history safely to datastruct
func (h *History) Add(symbol string, bars Bars) error {
	h.Lock()
	defer h.Unlock()

	var msg string

	if len(h.bars) == 0 {
		h.bars = make(map[string]Bars, 0)
	}

	b, ok := h.bars[symbol]
	if !ok {
		msg = "loading"

		h.bars[symbol] = bars
		// increase cap by +1
		c := make(chan string, len(h.bars))
		// copy values to new channel
		for len(h.C) > 0 {
			select {
			case v := <-h.C:
				c <- v
			default:
			}
		}
		h.C = c
	} else if len(b) == len(bars) && b.LastBar() == bars.LastBar() {
		// nothing new
		return errors.New("no new bars")
	} else {
		// save bars
		msg = fmt.Sprintf("added %d bars", len(bars))
		if err := WriteBars(symbol, bars); err != nil {
			log.Printf("could not save %s bars: %v\n", symbol, err)
		}
	}
	if len(bars) == 0 {
		return nil
	}

	// update history
	h.bars[symbol] = merge(b, bars)

	// delete if total bars is less then two
	if 2 > len(h.bars[symbol]) {
		delete(h.bars, symbol)
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

// Update enables or disables new bars data
// this will also remove outdated historys from struct but not from file
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
			for symbol := range h.bars {
				limit := maxlimit

				// calc how many new bars we can download from our last bar
				if len(h.bars[symbol]) > 0 {
					limit = calcLimit(h.bars[symbol].LastBar().T(), h.bars[symbol].Period())
					if limit > maxlimit {
						limit = maxlimit
					}
				}

				if limit > 1 {
					wg.Add(1)
					go h.download(symbol, limit, &wg)
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

// download and check validity before adding to history
func (h *History) download(symbol string, limit int, wg *sync.WaitGroup) error {
	defer wg.Done()

	pair, tf := SplitSymbol(symbol)

	var err error
	var bars Bars
	bars, err = h.GetKlines(pair, tf, limit)
	if err != nil {
		log.Printf("failed to download %d bars for %s: %v\n", limit, symbol, err)
		time.Sleep(2 * time.Minute)
		return err
	}
	// since we always get the current bar witch is not finish, we dont want to save that
	if 2 > len(bars) {
		return nil
	}
	// check if lastbar time is fresh, if not then delete symbol from history (not file)
	if time.Now().Add(2 * -bars.Period()).After(bars.LastBar().T()) {
		h.Lock()
		delete(h.bars, symbol)
		h.Unlock()
		log.Println(symbol, "outdated")
		return nil
	}
	// add to history
	h.Add(symbol, bars[1:])
	return nil
}
