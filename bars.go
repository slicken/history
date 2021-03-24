package history

import (
	"sort"
	"time"
)

const (
	// MINDUR is minumum duration for periods
	MINDUR = time.Duration(1 * time.Minute)
	// MAXDUR is maximum duration for periods
	MAXDUR = time.Duration(43829 * time.Minute)
)

// History wraps Bars with aditional info Symbol,Timeframe,lastUpdated info.
// It has no functions as all data is controlled from Data and Bars type
type History struct {
	Symbol    string
	Timeframe string
	//	Quote, Base string
	Bars Bars

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

// Period returns the calculated timeframe interval,
// need at least 2 bars or it will return 1 minute as default
func (bars Bars) Period() time.Duration {
	if 2 > len(bars) {
		return MINDUR
	}

	return bars[0].Time.Sub(bars[1].Time)
}

// FirstBar ..
func (bars Bars) FirstBar() Bar {
	if 1 > len(bars) {
		return Bar{}
	}

	return bars[len(bars)-1]
}

// LastBar ..
func (bars Bars) LastBar() Bar {
	if 1 > len(bars) {
		return Bar{}
	}

	return bars[0]
}

// Find returns bars for given time
func (bars Bars) Find(t time.Time) (n int, bar Bar) {
	if 1 > len(bars) {
		return -1, Bar{}
	}
	if bars.FirstBar().T().After(t) || bars.LastBar().T().Before(t) {
		return -1, Bar{}
	}

	for i, b := range bars {
		if b.T() == t {
			return i, b
		}
	}

	return -1, Bar{}
}

// TimeSpan returns bars for given start to end time
func (bars Bars) TimeSpan(start, end time.Time) Bars {
	span := make(Bars, 0)

	for _, b := range bars {
		if b.Time.After(start) && b.Time.Before(end) {
			span = append(span, b)
		}
	}

	span.Sort()
	return span
}

func merge(old, new Bars) Bars {
	if len(old) == 0 {
		return new
	}

	ft := old.FirstBar().T()
	lt := old.LastBar().T()

	merged := old
	for _, b := range new {
		if b.T().After(lt) || b.T().Before(ft) {
			merged = append(merged, b)
		}
	}

	// sort it
	merged = merged.Sort()
	return merged
}
