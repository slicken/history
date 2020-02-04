package history

import (
	"sort"
	"time"
)

// History wraps Bars with aditional info Symbol,Timeframe,lastUpdated info.
// It has no functions as all ata is controlled from Data and Bars type
type History struct {
	Symbol    string
	Timeframe string
	Bars      Bars

	lastUpdate time.Time
}

// Bars type holds all functions for bars
type Bars []Bar

// Sort bars
func (bars Bars) Sort() Bars {
	sort.SliceStable(bars, func(i, j int) bool {
		return bars[i].Time.Unix() > bars[j].Time.Unix()
	})

	return bars
}

// minDur is minumum duration for periods
const minDur = time.Duration(1 * time.Minute)

// maxDur is maximum duration for periods
const maxDur = time.Duration(43829 * time.Minute)

// Period returns the calculated timeframe interval,
// need at least 2 bars or it will return 1 minute as default
func (bars Bars) Period() time.Duration {
	if len(bars) < 2 {
		return minDur
	}

	return bars[0].Time.Sub(bars[1].Time)
}

// TimeSpan returns slice for given times
func (bars Bars) TimeSpan(start, end time.Time) Bars {
	span := make(Bars, 0)

	for _, v := range bars {
		if v.Time.After(start) && v.Time.Before(end) {
			span = append(span, v)
		}
	}

	span.Sort()
	return span
}

// FirstBar ..
func (bars Bars) FirstBar() Bar {
	if len(bars) == 0 {
		return Bar{}
	}

	return bars[len(bars)-1]
}

// LastBar ..
func (bars Bars) LastBar() Bar {
	if len(bars) == 0 {
		return Bar{}
	}

	return bars[0]
}
