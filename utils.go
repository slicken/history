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

const (
	DATADIR       = "data"
	FILESEPARATOR = "_"
	MAXLIMIT      = 1000
	MAXTRIES      = 3
)

// ReadBars loads ars from file
func ReadBars(symbol, timeframe string) (Bars, error) {
	var bars Bars
	filename := fn(symbol, timeframe)
	filepath := filepath.Join(DATADIR, filename)

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
func SaveBars(symbol, timeframe string, bars *Bars) error {
	// merge if file alredy exist
	if old, err := ReadBars(symbol, timeframe); err == nil {
		new := *merge(&old, bars)
		// skip if new bars equals old
		if new[0].Time == old[0].Time && old[len(old)-1].Time == new[len(new)-1].Time {
			return nil
		}
		bars = &new
	}

	b, err := json.MarshalIndent(&bars, "", "\t")
	if err != nil {
		return err
	}

	// create DATADIR if not exist
	if _, err := os.Stat(DATADIR); err != nil {
		if err := os.MkdirAll(DATADIR, os.ModePerm); err != nil {
			log.Fatal(err)
		}
	}

	filename := fn(symbol, timeframe)
	filepath := filepath.Join(DATADIR, filename)

	return ioutil.WriteFile(filepath, b, 0644)
}

// fn makes filename layout
func fn(symbol, timeframe string) string {
	return strings.ToUpper(symbol) + FILESEPARATOR + strings.ToLower(timeframe) + ".json"
}

// calculates how many bars between time.now and time.last
func calcLimit(last time.Time, timeframe string) int {
	dur := time.Duration(Tf(timeframe)) * time.Minute
	diff := time.Until(last)

	return -int(diff / dur)
}

// Split returns the symbol and timeframe from a SymbolTimeframe string
func Split(s string) (string, string) {
	if len(s) < 3 {
		return "", ""
	}

	// check lenth 3-1 for matching timeframes
	for i := 3; i > 0; i-- {
		symbol := s[:len(s)-i]
		timeframe := s[len(s)-i:]
		if tf := Tf(timeframe); tf != 0 {
			return symbol, timeframe
		}
	}

	// not found
	return "", ""
}
