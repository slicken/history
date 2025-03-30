package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/slicken/history"
)

// Binance data loaders
type Binance struct{}

// GetKlines new data from Binance exchange
func (e Binance) GetKlines(pair, timeframe string, limit int) (history.Bars, error) {
	path := fmt.Sprintf(
		"https://api.binance.com/api/v1/klines?symbol=%s&interval=%s&limit=%v",
		strings.ToUpper(pair), strings.ToLower(timeframe), limit)

	req, _ := http.NewRequest("GET", path, nil)
	req.Header.Add("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bytes, _ := io.ReadAll(resp.Body)

	var raw [][]interface{}
	if err := json.Unmarshal(bytes, &raw); err != nil {
		return nil, err
	}

	bars := make(history.Bars, 0, len(raw))
	for _, v := range raw {
		bar := history.Bar{
			Time: time.Unix(int64(v[0].(float64))/1000, 0),
		}

		if bar.Open, err = strconv.ParseFloat(v[1].(string), 64); err != nil {
			log.Printf("error parsing Open: %v", err)
		}
		if bar.High, err = strconv.ParseFloat(v[2].(string), 64); err != nil {
			log.Printf("error parsing High: %v", err)
		}
		if bar.Low, err = strconv.ParseFloat(v[3].(string), 64); err != nil {
			log.Printf("error parsing Low: %v", err)
		}
		if bar.Close, err = strconv.ParseFloat(v[4].(string), 64); err != nil {
			log.Printf("error parsing Close: %v", err)
		}
		if bar.Volume, err = strconv.ParseFloat(v[5].(string), 64); err != nil {
			log.Printf("error parsing Volume: %v", err)
		}

		bars = append(history.Bars{bar}, bars...)
	}

	return bars, nil
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
	Code       int    `json:"code"`
	Msg        string `json:"msg"`
	Timezone   string `json:"timezone"`
	Servertime int64  `json:"serverTime"`
	RateLimits []struct {
		RateLimitType string `json:"rateLimitType"`
		Interval      string `json:"interval"`
		Limit         int    `json:"limit"`
	} `json:"rateLimits"`
	ExchangeFilters interface{} `json:"exchangeFilters"`
	Symbols         []struct {
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
