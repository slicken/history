package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/slicken/history"
)

// Binance data loaders
type Binance struct{}

// Download new data from Binance exchange
func (d Binance) Download(symbol, timeframe string, limit int) (history.Bars, error) {
	path := fmt.Sprintf(
		"https://api.binance.com/api/v1/klines?symbol=%s&interval=%s&limit=%v",
		strings.ToUpper(symbol), strings.ToLower(timeframe), limit)

	req, _ := http.NewRequest("GET", path, nil)
	req.Header.Add("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := ioutil.ReadAll(resp.Body)

	return Process(&b)
}

// Process downloaded data to Bars slice
func Process(data *[]byte) (history.Bars, error) {
	var err error
	js := [][]interface{}{}
	if err = json.Unmarshal(*data, &js); err != nil {
		return nil, err
	}

	// process OHLC data to into Bar struct
	var list = make(history.Bars, 0)
	for i, v := range js {

		bar := history.Bar{}
		bar.Time = time.Unix(int64(v[0].(float64))/1000, 0) // .UTC()
		bar.Open, err = strconv.ParseFloat(v[1].(string), 64)
		if err != nil {
			log.Printf("error bars[%d].Open\n", i)
		}
		bar.High, err = strconv.ParseFloat(v[2].(string), 64)
		if err != nil {
			log.Printf("error bars[%d].High\n", i)
		}
		bar.Low, err = strconv.ParseFloat(v[3].(string), 64)
		if err != nil {
			log.Printf("error bars[%d].Low\n", i)
		}
		bar.Close, err = strconv.ParseFloat(v[4].(string), 64)
		if err != nil {
			log.Printf("error bars[%d].Close\n", i)
		}
		bar.Volume, err = strconv.ParseFloat(v[5].(string), 64)
		if err != nil {
			log.Printf("error bars[%d].Volume\n", i)
		}
		// insert
		list = append(history.Bars{bar}, list...)
	}

	return list, nil
}

// SetPairs adds all symbols to exchange memory
func (e *Binance) SetPairs() (m map[string]string) {

	info, err := e.ExchangeInfo()
	if err != nil {
		return nil
	}

	for _, v := range info.Symbols {
		m[v.Symbol] = ""
	}
	return nil
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

// ExchangeInfo returns exchange information.
func (e *Binance) ExchangeInfo() (ExchangeInfo, error) {
	url := "https://api.binance.com/api/v1/exchangeInfo"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Accept", "application/json")

	ei := ExchangeInfo{}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ei, err
	}
	defer resp.Body.Close()
	b, _ := ioutil.ReadAll(resp.Body)

	json.Unmarshal(b, &ei)
	return ei, err
}

// MakeSymbolTimeframe ..
func MakeSymbolTimeframe(currencie, timeframe string) ([]string, error) {
	ei, err := binance.ExchangeInfo()
	if err != nil {
		return nil, err
	}

	var result []string
	for _, symbol := range ei.Symbols {
		if symbol.QuoteAsset != currencie || symbol.Status != "TRADING" {
			continue
		}
		result = append(result, symbol.Symbol+timeframe)
	}

	return result, nil
}
