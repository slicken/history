package history

import (
	"errors"
	"fmt"
	"log"
	"time"
)

const dt_stamp = "2006/01/02 15:04"

// Tester is strategy backtester interface
type Tester interface {
	Test() (Events, error)
}

// Test strategys compatible with both Strategy (bars) and MultiStrategy (whole history struct)
func (hist *History) Test(strategy Strategy, start, end time.Time) (Events, error) {
	if len(hist.bars) == 0 {
		return nil, errors.New("no history")
	}

	var events Events
	log.Printf("[TEST] %s [%v ==> %v]\n", fmt.Sprintf("%T", strategy)[6:], start.Format(dt_stamp), end.Format(dt_stamp))

	for symbol, bars := range hist.bars {
		for streamedBars := range bars.StreamInterval(start, end, bars.Period()) {
			if event, ok := strategy.Run(symbol, streamedBars); ok {
				events.Add(event)
			}
		}
	}

	log.Printf("[TEST] completed with %d Events\n", len(events))
	return events, nil
}
