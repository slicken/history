package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/slicken/history"
)

// Binance data loaders
type Binance struct{}

// BinanceError represents the error response from Binance API
type BinanceError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// GetKlines new data from Binance exchange
func (e Binance) GetKlines(pair, timeframe string, limit int) (history.Bars, error) {
	var allBars history.Bars
	seenTimes := make(map[int64]bool)

	// Calculate the batch size for the API request
	batchSize := limit
	if batchSize > 1000 {
		batchSize = 1000
	}

	// For first request, don't specify endTime to get most recent bars
	path := fmt.Sprintf(
		"https://api.binance.com/api/v1/klines?symbol=%s&interval=%s&limit=%d",
		strings.ToUpper(pair), strings.ToLower(timeframe), batchSize)

	for {
		resp, err := http.Get(path)
		if err != nil {
			return nil, fmt.Errorf("failed to get klines: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		var data [][]interface{}
		if err := json.Unmarshal(body, &data); err != nil {
			// Try to parse as error response
			var errResp BinanceError
			if err := json.Unmarshal(body, &errResp); err == nil && errResp.Code == -1003 {
				// Extract ban timestamp from message
				msg := errResp.Msg
				if idx := strings.Index(msg, "until "); idx != -1 {
					timestampStr := strings.Split(msg[idx+6:], ".")[0]
					if banUntil, err := strconv.ParseInt(timestampStr, 10, 64); err == nil {
						now := time.Now().UnixMilli()
						sleepDuration := time.Duration(banUntil-now) * time.Millisecond
						if sleepDuration > 0 {
							log.Printf("IP banned for %s, sleeping until ban is lifted...", sleepDuration)
							time.Sleep(sleepDuration)
							continue
						}
					}
				}
			}
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		if len(data) == 0 {
			break // No more data available
		}

		// Convert batch to Bars
		batchBars := make(history.Bars, 0, len(data))
		var oldestTimestamp int64 = time.Now().UnixMilli()
		for _, kline := range data {
			// Stop if we have enough bars
			if len(allBars) >= limit {
				break
			}

			timestamp := int64(kline[0].(float64))
			// Track oldest timestamp in this batch
			if timestamp < oldestTimestamp {
				oldestTimestamp = timestamp
			}

			// Skip if we've already seen this timestamp
			if seenTimes[timestamp] {
				continue
			}
			seenTimes[timestamp] = true

			t := time.Unix(timestamp/1000, 0)
			open, _ := strconv.ParseFloat(kline[1].(string), 64)
			high, _ := strconv.ParseFloat(kline[2].(string), 64)
			low, _ := strconv.ParseFloat(kline[3].(string), 64)
			close, _ := strconv.ParseFloat(kline[4].(string), 64)
			volume, _ := strconv.ParseFloat(kline[5].(string), 64)

			bar := history.Bar{
				Time:   t,
				Open:   open,
				High:   high,
				Low:    low,
				Close:  close,
				Volume: volume,
			}
			batchBars = append(batchBars, bar)
		}

		// Sort batch in descending order
		sort.SliceStable(batchBars, func(i, j int) bool {
			return batchBars[i].Time.After(batchBars[j].Time)
		})

		// Append this batch
		allBars = append(allBars, batchBars...)

		// Stop if we have enough bars or no more data
		if len(allBars) >= limit || len(data) < batchSize {
			break
		}

		// Calculate remaining bars needed
		remaining := limit - len(allBars)
		if remaining < batchSize {
			batchSize = remaining
		}

		// Get next batch using endTime from oldest bar
		path = fmt.Sprintf(
			"https://api.binance.com/api/v1/klines?symbol=%s&interval=%s&limit=%d&endTime=%d",
			strings.ToUpper(pair), strings.ToLower(timeframe), batchSize, oldestTimestamp-1)

		// Respect rate limits
		time.Sleep(2 * time.Second)
	}

	// Ensure we don't return more bars than requested
	if len(allBars) > limit {
		allBars = allBars[:limit]
	}

	return allBars, nil
}

// MakeSymbolMultiTimeframe helper func for binance that makes slice of requested symbols and timeframes
func MakeSymbolMultiTimeframe(currencie string, timeframes ...string) ([]string, error) {
	// run func
	ei, err := GetExchangeInfo()
	if err != nil {
		return nil, err
	}

	// make pair slice
	var result []string
	for _, pair := range ei.Symbols {
		if pair.QuoteAsset != currencie || pair.Status != "TRADING" {
			continue
		}

		// exclude list
		ok := true
		for _, x := range []string{"DOWN", "UP", "BULL", "BEAR", "AUD", "BUSD", "BIDR", "BKRW", "DAI", "EUR", "GBP", "IDRT", "NGN", "PAX", "RUB", "TUSD", "TRY", "UAH", "USDC", "ZAR", "BUSD", "SUSD", "USDP"} {
			if strings.Contains(pair.QuoteAsset, x) || strings.Contains(pair.BaseAsset, x) {
				ok = false
			}

		}
		if !ok {
			continue
		}

		for _, tf := range timeframes {
			if !history.TFIsValid(tf) {
				log.Println("unkown timeframe", tf)
			}
			result = append(result, pair.Symbol+tf)
		}
	}
	return result, nil
}

// ExchangeInfo holds the full exchange information type
type ExchangeInfo struct {
	Symbols []struct {
		Symbol             string   `json:"symbol"`
		Status             string   `json:"status"`
		BaseAsset          string   `json:"baseAsset"`
		BaseAssetPrecision int      `json:"baseAssetPrecision"`
		QuoteAsset         string   `json:"quoteAsset"`
		QuotePrecision     int      `json:"quotePrecision"`
		OrderTypes         []string `json:"orderTypes"`
		IcebergAllowed     bool     `json:"icebergAllowed"`
		Filters            []struct {
			FilterType  string  `json:"filterType"`
			MinPrice    float64 `json:"minPrice,string"`
			MaxPrice    float64 `json:"maxPrice,string"`
			TickSize    float64 `json:"tickSize,string"`
			MinQty      float64 `json:"minQty,string"`
			MaxQty      float64 `json:"maxQty,string"`
			StepSize    float64 `json:"stepSize,string"`
			MinNotional float64 `json:"minNotional,string"`
		} `json:"filters"`
	} `json:"symbols"`
}

// func that download and return exchange info
func GetExchangeInfo() (ExchangeInfo, error) {
	url := "https://api.binance.com/api/v1/exchangeInfo"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Accept", "application/json")

	ei := ExchangeInfo{}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ei, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)

	json.Unmarshal(b, &ei)
	return ei, err
}
