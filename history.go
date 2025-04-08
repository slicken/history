package history

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// History maintaner
type History struct {
	bars   map[string]Bars
	update bool
	// Notify channel when new bars are added for any loaded symbol
	C chan string
	// Plug diffrent downloaders
	Downloader
	// SQLite3 database connection
	db *sql.DB

	sync.RWMutex
}

// New creates and initializes a new History instance
func New() (*History, error) {
	h := &History{
		bars:   make(map[string]Bars),
		update: false,
		C:      make(chan string, 1),
	}

	// Initialize SQLite3 database
	db, err := sql.Open("sqlite3", "history.db")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}
	h.db = db

	// Test database connection
	if err := h.db.Ping(); err != nil {
		h.db.Close()
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	// Create bars table if it doesn't exist
	_, err = h.db.Exec(`
		CREATE TABLE IF NOT EXISTS bars (
			symbol TEXT NOT NULL,
			time INTEGER NOT NULL,
			open REAL NOT NULL,
			high REAL NOT NULL,
			low REAL NOT NULL,
			close REAL NOT NULL,
			volume REAL,
			PRIMARY KEY (symbol, time)
		)
	`)
	if err != nil {
		h.db.Close()
		return nil, fmt.Errorf("failed to create bars table: %v", err)
	}

	return h, nil
}

// Downloader interface plugs functions that download bars
type Downloader interface {
	GetKlines(pair, timeframe string, limit int) (Bars, error)
}

// GetBars returns bars saftly
func (h *History) GetBars(symbol string) Bars {
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

// LimiTime the data for specified data time intervalls
func (h *History) LimitTime(start, end time.Time) *History {
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
			bars, _ := h.ReadBars(symbol)
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
		msg = fmt.Sprintf("loaded %d bars", len(bars))

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
		msg = fmt.Sprintf("added %d new bars", len(bars))
		if err := h.WriteBars(symbol, bars); err != nil {
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

	// notify that we have new bars
	select {
	case h.C <- symbol:
	default:
	}

	return nil
}

// Update enables or disables new bars data
// this will also unload outdated historys (inaktive symbols) from history but not from database
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
				}

				if limit > 1 {

					var wg sync.WaitGroup
					for symbol := range h.bars {

						wg.Add(1)
						go func(symbol string, wg *sync.WaitGroup) {
							defer wg.Done()

							// download bars
							err := h.download(symbol, limit)
							if err != nil {
								log.Printf("faled to update %s: %v\n", symbol, err)
								time.Sleep(10 * time.Second)
							}
						}(symbol, &wg)
					}
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

// ReprocessHistory downloads and overwrites bars for all loaded symbols with specified limit
func (h *History) ReprocessHistory(limit int) error {
	log.Printf("reprocessing %d bars\n", limit)

	h.RLock()
	var wg sync.WaitGroup
	for symbol := range h.bars {

		wg.Add(1)
		go func(symbol string, wg *sync.WaitGroup) {
			defer wg.Done()

			// download bars
			err := h.download(symbol, limit)
			if err != nil {
				log.Printf("failed to reprocess bars for %s: %v\n", symbol, err)
			}
		}(symbol, &wg)
	}
	h.RUnlock()

	wg.Wait()
	return nil
}

// download and check validity before adding to history
func (h *History) download(symbol string, limit int) error {

	pair, tf := SplitSymbol(symbol)

	var err error
	var bars Bars
	bars, err = h.GetKlines(pair, tf, limit)
	if err != nil {
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
		log.Println(symbol, "outdated - unloading from history.")
		return nil
	}
	// add to history
	h.Add(symbol, bars[1:])
	return nil
}
