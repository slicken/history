package history

import (
	"sort"
	"time"
)

const (
	mindur = time.Duration(1 * time.Minute)
	// maxdur = time.Duration(43829 * time.Minute)
	maxdur = time.Duration(70560 * time.Minute)
)

// Bars type holds all functions for bars
type Bars []Bar

// Sort bars by time
func (bars Bars) Sort() Bars {
	sort.SliceStable(bars, func(i, j int) bool {
		return bars[i].Time.Unix() > bars[j].Time.Unix()
	})

	return bars
}

// Returns bars in reverse list
func (bars Bars) Reverse() Bars {
	length := len(bars)
	rev := make(Bars, length)
	for i, j := 0, length-1; i < length; i, j = i+1, j-1 {
		rev[i] = bars[j]
	}
	return rev
}

// Period returns the calculated timeframe interval,
// need at least 2 bars or it will return 1 minute as default
func (bars Bars) Period() time.Duration {
	if 2 > len(bars) {
		return mindur
	}

	return bars[0].Time.Sub(bars[1].Time)
}

// FirstBar in dataset
func (bars Bars) FirstBar() Bar {
	if 1 > len(bars) {
		return Bar{}
	}

	return bars[len(bars)-1]
}

// LastBar in dataset
func (bars Bars) LastBar() Bar {
	if 1 > len(bars) {
		return Bar{}
	}

	return bars[0]
}

// Find Bar for given time
func (bars Bars) Find(dt time.Time) (n int, bar Bar) {
	if 1 > len(bars) {
		return -1, Bar{}
	}
	if bars.FirstBar().T().After(dt) || bars.LastBar().T().Before(dt) {
		return -1, Bar{}
	}

	for i, b := range bars {
		if b.T() == dt {
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

// merges bars
func merge(old, new Bars) Bars {
	if len(old) == 0 {
		return new
	}

	first := old.FirstBar().T()
	last := old.LastBar().T()

	merged := old
	for _, b := range new {
		if b.T().After(last) || b.T().Before(first) {
			merged = append(merged, b)
		}
	}

	// sort it
	merged = merged.Sort()
	return merged
}
