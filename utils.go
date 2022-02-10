package history

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	maxlimit = 1000
	datadir  = "data"
)

// Setmaxlimit limits new data request
func (h *History) SetMaxLimit(v int) {
	maxlimit = v
}

// Setdatadir to store files in
func (h *History) SetDataDir(v string) {
	datadir = v
}

// ReadBars loads ars from file
func ReadBars(symbol string) (Bars, error) {
	var bars Bars
	filename := nameJson(symbol)
	filepath := filepath.Join(datadir, filename)

	b, err := ioutil.ReadFile(filepath)
	if err != nil {
		return bars, err
	}

	if err = json.Unmarshal(b, &bars); err != nil {
		return bars, err
	}

	return bars, nil
}

// SaveBars saves bars to file
func SaveBars(symbol string, bars Bars) error {
	// merge if file alredy exist
	if old, err := ReadBars(symbol); err == nil {
		// skip if new last equeals old of
		if bars.LastBar() == old.LastBar() {
			return nil
		}
		bars = merge(old, bars)
	}

	b, err := json.MarshalIndent(&bars, "", "\t")
	if err != nil {
		return err
	}

	// create datadir if does not exist
	if _, err := os.Stat(datadir); err != nil {
		if err := os.MkdirAll(datadir, os.ModePerm); err != nil {
			log.Fatal(err)
		}
	}

	filename := nameJson(symbol)
	filepath := filepath.Join(datadir, filename)

	return ioutil.WriteFile(filepath, b, 0644)
}

// nameJson makes filename layout
func nameJson(f string) string {
	return strings.ToLower(f) + ".json"
}

// calculates how many bars between time.now and time.last
func calcLimit(last time.Time, period time.Duration) int {
	t := time.Until(last)
	return -int(t / period)
}

// SplitPair returns base quote
func Split(s string) (pair string, tf string) {
	// split pair and timeframe
	for i := len(s); i >= 0; i-- {
		pair = s[:len(s)-i]
		tf = s[len(s)-i:]
		if tf := TF2Interval(tf); tf != 0 {
			s = s[:len(s)-i]
			break
		}
	}

	return pair, tf
}

func toUnix(t time.Time) int64 {
	return t.UnixNano() / 1e6
}
