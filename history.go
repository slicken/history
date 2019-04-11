package history

import (
	"log"
	"sync"
	"time"
)

// History wraps Bars with its settings. It has no acessable functions as
// all is controlled from Data and Bars type
type History struct {
	Symbol    string
	Timeframe string
	Bars      Bars

	update chan int
	sync.RWMutex
}

// TimeSpan returns slice for given times ----------------------------- make pointer?
func (h History) TimeSpan(start, end time.Time) History {
	bars := make(Bars, 0)

	for _, v := range h.Bars {
		if v.Time.After(start) && v.Time.Before(end) {
			bars = append(bars, v)
		}
	}

	h.Bars = make(Bars, 0, 0)
	h.Bars = bars
	h.Bars.Sort()

	return h
}

// Updater downloads and updates bars
// updater channel is the controller. close channel to close update
func (h *History) updateHandler(d *Data) {
	// return if we alredy running
	select {
	case _, open := <-h.update:
		if open {
			return
		}
	default:
	}

	h.Lock()
	h.update = make(chan int, 1)
	h.Unlock()

	go func() {
		var limit, tries int
		symbol, timeframe := h.Symbol, h.Timeframe

		for {
			select {
			case v, ok := <-h.update:
				if !ok {
					return
				}

				tries = 0
				for {

					bars, err := d.Download(symbol, timeframe, v)
					if err != nil {
						log.Printf("failed to download %d bar(s) for %s%s: %v\n", v, symbol, timeframe, err)
						if tries >= maxtries {
							log.Printf("stopped updates for %s%s\n", symbol, timeframe)
							close(h.update)
							return
						}
						// failed. try again
						tries++
						time.Sleep(10 * time.Second)
						continue
					}
					// success. update history
					log.Printf("%s%s downloaded %d bar(s)\n", symbol, timeframe, len(bars))
					d.Add(symbol, timeframe, &bars)
					break
				}

			default:
				limit = maxlimit
				if len(h.Bars) != 0 {
					limit = calcLimit(h.Bars[0].Time, timeframe)
				}
				if limit > 0 {
					h.update <- limit
				}
				time.Sleep(15 * time.Second)
			}
		}
	}()
}
