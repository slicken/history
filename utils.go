package history

import (
	"os"
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

// StoredSymbols
func StoredSymbols() ([]string, error) {
	files, err := os.ReadDir(datadir)
	if err != nil {
		return nil, err
	}

	var symbols []string
	for _, f := range files {
		symbols = append(symbols, f.Name()[:len(f.Name())-5])
	}
	return symbols, nil
}

// ReadBars loads bars from database
func (h *History) ReadBars(symbol string) (Bars, error) {
	var bars Bars

	// Query all bars for the symbol
	rows, err := h.db.Query(`
		SELECT time, open, high, low, close, volume
		FROM bars
		WHERE symbol = ?
		ORDER BY time DESC
	`, symbol)
	if err != nil {
		return bars, err
	}
	defer rows.Close()

	// Read each bar
	for rows.Next() {
		var bar Bar
		var timestamp int64
		err := rows.Scan(
			&timestamp,
			&bar.Open,
			&bar.High,
			&bar.Low,
			&bar.Close,
			&bar.Volume,
		)
		if err != nil {
			return bars, err
		}
		bar.Time = time.Unix(timestamp, 0)
		bars = append(bars, bar)
	}

	if err = rows.Err(); err != nil {
		return bars, err
	}

	return bars, nil
}

// WriteBars saves bars to database
func (h *History) WriteBars(symbol string, bars Bars) error {
	// merge if bars already exist
	if old, err := h.ReadBars(symbol); err == nil {
		// skip if new last equals old
		if bars.LastBar() == old.LastBar() {
			return nil
		}
		bars = merge(old, bars)
	}

	// Begin transaction
	tx, err := h.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// First delete all existing bars for this symbol
	_, err = tx.Exec("DELETE FROM bars WHERE symbol = ?", symbol)
	if err != nil {
		return err
	}

	// Prepare statement for inserting bars
	stmt, err := tx.Prepare(`
		INSERT INTO bars (symbol, time, open, high, low, close, volume)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Insert each bar
	for _, bar := range bars {
		_, err = stmt.Exec(
			symbol,
			bar.Time.Unix(),
			bar.Open,
			bar.High,
			bar.Low,
			bar.Close,
			bar.Volume,
		)
		if err != nil {
			return err
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

// calculates how many bars between time.now and time.last
func calcLimit(last time.Time, period time.Duration) int {
	t := time.Until(last)
	return -int(t / period)
}

// Splits Sumbol to Pair and Timeframe
func SplitSymbol(s string) (pair string, tf string) {
	// split pair and timeframe
	for i := len(s); i >= 0; i-- {
		pair = s[:len(s)-i]
		tf = s[len(s)-i:]
		if tf := TFInterval(tf); tf != 0 {
			s = s[:len(s)-i]
			break
		}
	}

	return pair, tf
}

// ToUnixTime converts time to Unix time
func ToUnixTime(t time.Time) int64 {
	return t.Unix() / 1e6
}
