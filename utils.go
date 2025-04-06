package history

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	maxlimit = 1000
	fileDir  = "data"
)

// Setmaxlimit limits new data request
func (h *History) SetMaxLimit(v int) {
	maxlimit = v
}

// StoredSymbols returns all unique symbols from the database
func (h *History) StoredSymbols() ([]string, error) {
	rows, err := h.db.Query(`
		SELECT DISTINCT symbol 
		FROM bars 
		ORDER BY symbol ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var symbols []string
	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			return nil, err
		}
		symbols = append(symbols, symbol)
	}

	if err = rows.Err(); err != nil {
		return nil, err
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

// WriteJSON writes bars to a JSON file
func (bars Bars) WriteJSON(filename string) error {
	if len(bars) == 0 {
		return fmt.Errorf("no bars data to write")
	}

	if err := os.MkdirAll(fileDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	if !strings.HasSuffix(filename, ".json") {
		filename += ".json"
	}

	filepath := filepath.Join(fileDir, filename)
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	log.Println("Wrote to JSON file:", filepath)
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(bars)
}

// WriteCSV writes bars to a CSV file
func (bars Bars) WriteCSV(filename string) error {
	if len(bars) == 0 {
		return fmt.Errorf("no bars data to write")
	}

	if err := os.MkdirAll(fileDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	if !strings.HasSuffix(filename, ".csv") {
		filename += ".csv"
	}

	filepath := filepath.Join(fileDir, filename)
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"Time", "Open", "High", "Low", "Close", "Volume"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %v", err)
	}

	// Write data rows
	for _, bar := range bars {
		row := []string{
			bar.Time.Format(time.RFC3339),
			fmt.Sprintf("%f", bar.Open),
			fmt.Sprintf("%f", bar.High),
			fmt.Sprintf("%f", bar.Low),
			fmt.Sprintf("%f", bar.Close),
			fmt.Sprintf("%f", bar.Volume),
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %v", err)
		}
	}

	log.Println("Wrote to CSV file:", filepath)
	return nil
}

// ByteData holds byte data and error
type ByteData struct {
	data []byte
	err  error
}

// ToFile writes ByteData to a file
func (b ByteData) ToFile(filename string) error {
	if b.err != nil {
		return b.err
	}

	if err := os.MkdirAll(fileDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	filepath := filepath.Join(fileDir, filename)
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	// Pretty print the JSON with indentation
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, b.data, "", "  "); err != nil {
		return fmt.Errorf("failed to indent JSON: %v", err)
	}

	_, err = prettyJSON.WriteTo(file)
	if err != nil {
		return fmt.Errorf("failed to write data: %v", err)
	}

	log.Println("Wrote to file:", filepath)
	return nil
}
