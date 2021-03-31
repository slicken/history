package history

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

var (
	errNoHist = errors.New("no history")
	errNoBars = errors.New("no bars")
)

// TO DO--
// rebuild the structure to
// Data rename to History
// History to Bars ... and should hold map[symbolTF]Bars ([]bar)

// Data is ExchangeData
type Data struct {
	History []*History
	update  bool
	// C notify channel when we got now bars for a history (symbol timeframe)
	C chan string
	sync.RWMutex

	dataPath string
	// Plug diffrent downloaders
	Downloader
}

// Downloader interface plugs functions that download bars
type Downloader interface {
	GetKlines(symbol, timeframe string, limit int) (Bars, error)
}

// List returns string slice (BTCUSDT4h) of loaded historys
func (d *Data) List() []string {
	d.RLock()
	defer d.RUnlock()

	var list []string
	for _, hist := range d.History {
		list = append(list, string(hist.Symbol+hist.Timeframe))
	}
	return list
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

	return hist.Bars
}

// MinPeriod returns minimum period of historys
func (d *Data) MinPeriod() time.Duration {
	d.RLock()
	defer d.RUnlock()

	var min = MINDUR
	for _, hist := range d.History {
		period := hist.Bars.Period()
		if min == MINDUR || period < min {
			min = period
		}
	}

	return min
}

// FirstTime returns minimum period of historys
func (d *Data) FirstTime() time.Time {
	d.RLock()
	defer d.RUnlock()

	var v = time.Time{}
	for _, hist := range d.History {

		func(h *History) {
			if len(h.Bars) == 0 {
				return
			}

			t := h.Bars[len(h.Bars)-1].Time
			if v.IsZero() || t.Before(v) {
				v = t
			}

		}(hist)
	}

	return v
}

// LastTime returns minimum period of historys
func (d *Data) LastTime() time.Time {
	d.RLock()
	defer d.RUnlock()

	var last = time.Time{}
	for _, hist := range d.History {

		func(h *History) {
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

// Limit the data for all history
func (d *Data) Limit(length int) *Data {
	var wg sync.WaitGroup

	for _, h := range d.History {

		wg.Add(1)
		go func(h *History, wg *sync.WaitGroup, lenght int) {
			defer wg.Done()

			if lenght > len(h.Bars) {
				return
			}
			h.Bars = h.Bars[:lenght]

		}(h, &wg, length)
	}

	wg.Wait()
	return d
}

// TimeSpan returns all historys for given start to end time
func (d *Data) TimeSpan(start, end time.Time) *Data {
	var wg sync.WaitGroup

	for _, h := range d.History {

		wg.Add(1)
		go func(h *History, wg *sync.WaitGroup) {
			defer wg.Done()

			h.Bars = h.Bars.TimeSpan(start, end)
		}(h, &wg)
	}

	wg.Wait()
	return d
}

// Delete removes a history from data struct gracefully
func (d *Data) Delete(symbol, timeframe string) error {
	d.Lock()
	defer d.Unlock()

	for i, hist := range d.History {
		if hist.Symbol == symbol && hist.Timeframe == timeframe {
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
func (d *Data) Load(symbols ...string) error {
	var wg sync.WaitGroup

	for _, s := range symbols {
		symbol, timeframe := Split(s)
		if symbol == "" || timeframe == "" {
			log.Printf("could not load %s. invalid input\n", symbol+timeframe)
			continue
		}
		wg.Add(1)
		go func(symbol, timeframe string, wg *sync.WaitGroup) { // is this working properly ?
			defer wg.Done()

			// we dont check for errors, cause we use add ether way
			bars, _ := ReadBars(symbol, timeframe)

			d.Add(symbol, timeframe, bars, 0)
		}(symbol, timeframe, &wg)
	}

	wg.Wait()
	return nil
}

// // Add new history safely to datastruct
// func (d *Data) Add(symbol, timeframe string, bars Bars, n int) error {

// 	// create or get history
// 	hist, err := d.history(symbol, timeframe)
// 	if err != nil {
// 		hist = new(History)
// 		hist.Symbol = symbol
// 		hist.Timeframe = timeframe

// 		d.Lock()
// 		d.History = append(d.History, hist)
// 		// increase cap by +1
// 		c := make(chan string, len(d.History))
// 		// copy values to new channel
// 		for len(d.C) > 0 {
// 			select {
// 			case v, _ := <-d.C:
// 				c <- v
// 			default:
// 			}
// 		}
// 		d.C = c
// 		d.Unlock()
// 	}

// 	// add or merge bars
// 	if len(bars) != 0 {
// 		update := fmt.Sprintf("added %d bars", n)
// 		if err != nil {
// 			update = "loaded"
// 		}

// 		d.Lock()

// 		// save bars
// 		if update != "loaded" {
// 			if err = SaveBars(symbol, timeframe, bars); err != nil {
// 				log.Printf("could not save %s%s bars: %v\n", hist.Symbol, hist.Timeframe, err)
// 			}
// 		}

// 		// update history
// 		hist.lastUpdate = time.Now()
// 		hist.Bars = merge(hist.Bars, bars)

// 		d.Unlock()

// 		log.Printf("%s%s %s\n", hist.Symbol, hist.Timeframe, update)

// 		// notify data.C that we have updated symbol (not when loading)
// 		if n > 0 {
// 			select {
// 			case d.C <- (symbol + timeframe):
// 			default:
// 			}
// 		}
// 	}

// 	return nil
// }

// Add new history safely to datastruct
func (d *Data) Add(symbol, timeframe string, bars Bars, n int) error {
	// d.RLock()
	// defer d.RUnlock()
	var msg string

	// create or get history
	hist, err := d.history(symbol, timeframe)
	if err != nil {
		msg = "loaded"

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

	} else {
		// save bars
		msg = fmt.Sprintf("added %d bars", n)

		if err = SaveBars(symbol, timeframe, bars); err != nil {
			log.Printf("could not save %s%s bars: %v\n", hist.Symbol, hist.Timeframe, err)
		}
	}
	if len(bars) == 0 {
		return nil
	}

	// update history
	d.Lock()
	hist.lastUpdate = time.Now()
	hist.Bars = merge(hist.Bars, bars)
	d.Unlock()

	log.Printf("%s%s %s\n", hist.Symbol, hist.Timeframe, msg)

	// notify data.C that we have updated symbol (not when loading)
	if n > 0 {
		select {
		case d.C <- (symbol + timeframe):
		default:
		}
	}

	return nil
}

// Update enables/disables new data updates
func (d *Data) Update(enabled bool) {
	d.Lock()
	d.update = enabled
	d.Unlock()

	go func() {
		if enabled {
			log.Printf("UPDATES enabled")
		}
		for {
			d.RLock()
			enabled = d.update
			d.RUnlock()
			if !enabled {
				log.Printf("UPDATES disabled")
				return
			}

			var wg sync.WaitGroup

			d.RLock()
			for _, hist := range d.History {

				wg.Add(1)
				go func(hist *History, wg *sync.WaitGroup) {
					defer wg.Done()

					limit := MAXLIMIT
					for limit == MAXLIMIT {
						if len(hist.Bars) != 0 {
							// check how many new bars there is from last bars time
							limit = calcLimit(hist.Bars.LastBar().T(), hist.Timeframe)
							if limit > MAXLIMIT {
								limit = MAXLIMIT
							}
						}

						if limit != 0 {
							d.updateHistory(hist, limit)
						}
					}

				}(hist, &wg)
			}
			d.RUnlock()

			wg.Wait()
			time.Sleep(10 * time.Second)
		}
	}()
}

// updateHistory downloads and updates history
func (d *Data) updateHistory(h *History, limit int) {
	if !time.Now().After(h.lastUpdate) {
		return
	}

	var err error
	tries := 0

	for tries < MAXTRIES {
		bars, err := d.GetKlines(h.Symbol, h.Timeframe, limit)
		if err != nil {
			// try again
			tries++
			time.Sleep(2 * time.Second)
			continue
		}
		// since we always "gets" the current bar witch is not finish, we dont want to save that
		if len(bars) == 1 {
			return
		}
		// success. add to history
		d.Add(h.Symbol, h.Timeframe, bars[1:], len(bars)-1)
		return
	}

	// failed. penatly time added
	log.Printf("failed to download %d bars for %s%s: %v\n", limit, h.Symbol, h.Timeframe, err)
	h.lastUpdate = time.Now().Add(10 * time.Minute)
}

// CUpdate
func (d *Data) CUpdate() {
	d.RLock()
	defer d.RUnlock()

	for _, h := range d.History {
		select {
		case d.C <- (h.Symbol + h.Timeframe):
		default:
		}
	}
}
