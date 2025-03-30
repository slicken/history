package history

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	mindur = time.Duration(1 * time.Minute)
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

// EXPORT_QUERY defines how bars should be exported using SQL-like syntax
// Example queries:
// TradingView compatible format (default):
// "SELECT UNIX_TIMESTAMP(time)*1000 as time, open, high, low, close"
//
// Custom formats:
// "SELECT DATE_FORMAT(time, '%Y-%m-%d %H:%M:%S') as date, CAST(open as CHAR) as opening_price, high, low, ROUND(close,4) as closing_price"
// "SELECT UNIX_TIMESTAMP(time) as timestamp, ROUND(open,4) as open_price, ROUND(close,2) as close_price"
// "SELECT time, open, high, low, close" -- standard fields
// "SELECT DATE_FORMAT(time, '%Y-%m-%dT%H:%i:%sZ') as timestamp, *" -- ISO8601 time with all fields
var EXPORT_QUERY string = "SELECT UNIX_TIMESTAMP(time)*1000 as time, open, high, low, close"

// Export returns bars as JSON based on EXPORT_QUERY
func (bars Bars) Export() ([]byte, error) {
	if len(bars) == 0 {
		return []byte("[]"), nil
	}

	// Parse the query to determine field selection and transformations
	fields, err := parseExportQuery(EXPORT_QUERY)
	if err != nil {
		return nil, fmt.Errorf("invalid export query: %v", err)
	}

	// Build the result array
	var result []map[string]interface{}
	for _, bar := range bars {
		item := make(map[string]interface{})

		// Process each field based on the query
		for _, field := range fields {
			var value interface{}

			switch {
			case field.isTimeFunction():
				value = formatTimeSQL(bar.Time, field)
			case field.isRoundFunction():
				value = roundValue(bar, field)
			case field.isCastFunction():
				value = castValue(bar, field)
			default:
				value = getBarValue(bar, field.name)
			}

			item[field.alias] = value
		}
		result = append(result, item)
	}

	return json.Marshal(result)
}

type exportField struct {
	name     string   // original field name
	alias    string   // renamed field (if any)
	function string   // SQL function (ROUND, CAST, DATE_FORMAT, etc.)
	args     []string // function arguments
}

func (f exportField) isTimeFunction() bool {
	return f.function == "UNIX_TIMESTAMP" || f.function == "DATE_FORMAT"
}

func (f exportField) isRoundFunction() bool {
	return f.function == "ROUND"
}

func (f exportField) isCastFunction() bool {
	return f.function == "CAST"
}

// formatTimeSQL handles SQL time functions
func formatTimeSQL(t time.Time, field exportField) interface{} {
	switch field.function {
	case "UNIX_TIMESTAMP":
		unix := t.Unix()
		if len(field.args) > 0 && field.args[0] == "*1000" {
			return t.UnixMilli()
		}
		return unix
	case "DATE_FORMAT":
		if len(field.args) > 0 {
			format := convertMySQLToGoFormat(field.args[0])
			return t.Format(format)
		}
		return t.Format("2006-01-02 15:04:05")
	default:
		return t
	}
}

// roundValue handles ROUND function
func roundValue(bar Bar, field exportField) interface{} {
	value := getBarValue(bar, field.name)
	if v, ok := value.(float64); ok && len(field.args) > 0 {
		if precision, err := strconv.Atoi(field.args[0]); err == nil {
			return roundFloat(v, precision)
		}
	}
	return value
}

// castValue handles CAST function
func castValue(bar Bar, field exportField) interface{} {
	value := getBarValue(bar, field.name)
	if len(field.args) == 0 {
		return value
	}

	switch strings.ToUpper(field.args[0]) {
	case "CHAR", "VARCHAR", "TEXT":
		return fmt.Sprintf("%v", value)
	case "SIGNED", "INT":
		if v, ok := value.(float64); ok {
			return int64(v)
		}
	case "DECIMAL", "FLOAT":
		return value
	}
	return value
}

// getBarValue returns the value of a field from a bar
func getBarValue(bar Bar, field string) interface{} {
	switch field {
	case "time":
		return bar.Time
	case "open":
		return bar.Open
	case "high":
		return bar.High
	case "low":
		return bar.Low
	case "close":
		return bar.Close
	case "volume":
		return bar.Volume
	default:
		return nil
	}
}

// convertMySQLToGoFormat converts MySQL date format to Go time format
func convertMySQLToGoFormat(mysqlFormat string) string {
	replacements := map[string]string{
		"%Y": "2006",
		"%m": "01",
		"%d": "02",
		"%H": "15",
		"%i": "04",
		"%s": "05",
		"%T": "15:04:05",
		"%Z": "Z07:00",
	}

	result := mysqlFormat
	for mysql, go_fmt := range replacements {
		result = strings.ReplaceAll(result, mysql, go_fmt)
	}
	return result
}

// roundFloat rounds a float to the specified precision
func roundFloat(v float64, precision int) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(v*ratio) / ratio
}

// parseExportQuery parses the SQL-like query string into field specifications
func parseExportQuery(query string) ([]exportField, error) {
	// Remove SELECT and any whitespace
	query = strings.TrimSpace(strings.TrimPrefix(query, "SELECT"))

	// Split into individual field specifications
	var fields []exportField
	for _, part := range strings.Split(query, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		var field exportField

		// Handle * wildcard
		if part == "*" {
			return []exportField{
				{name: "time"},
				{name: "open"},
				{name: "high"},
				{name: "low"},
				{name: "close"},
				{name: "volume"},
			}, nil
		}

		// Parse functions like ROUND(field,n) or DATE_FORMAT(time,'%Y-%m-%d')
		if strings.Contains(part, "(") && strings.Contains(part, ")") {
			funcStart := strings.Index(part, "(")
			funcEnd := strings.LastIndex(part, ")")
			if funcStart > 0 && funcEnd > funcStart {
				field.function = strings.TrimSpace(part[:funcStart])
				args := strings.Split(part[funcStart+1:funcEnd], ",")
				field.name = strings.TrimSpace(args[0])
				if len(args) > 1 {
					field.args = make([]string, len(args)-1)
					for i, arg := range args[1:] {
						field.args[i] = strings.Trim(strings.TrimSpace(arg), "'")
					}
				}
				part = part[:funcStart] + field.name + part[funcEnd+1:]
			}
		}

		// Check for alias
		if strings.Contains(strings.ToUpper(part), " AS ") {
			parts := strings.Split(part, " AS ")
			if len(parts) == 2 {
				if field.name == "" {
					field.name = strings.TrimSpace(parts[0])
				}
				field.alias = strings.TrimSpace(parts[1])
			}
		} else if field.name == "" {
			field.name = strings.TrimSpace(part)
		}

		// If no alias specified, use the field name
		if field.alias == "" {
			field.alias = field.name
		}

		fields = append(fields, field)
	}

	return fields, nil
}
