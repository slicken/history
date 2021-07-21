package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/slicken/history"
	"github.com/slicken/history/charts"
)

var (
	hist   = new(history.History)       // this is where we store all historys/bars and there settings
	events = new(history.Events)        // results (events) from strategy that we can later can plot
	evl    = new(history.EventListener) // we add our strategy to eventlistener witch is looking at the hist

	// strat = &MaCrossing{fast: 10, slow: 20} // strategy for this example
	strat = &Power2{}             // strategy for this example
	chart = charts.DefaultChart() // small chart module (highcharts.com)
	conf  = new(Config)           // config from args

	exclude = []string{"DOWN", "UP", "BULL", "BEAR", "AUD", "BUSD", "BIDR", "BKRW", "DAI", "EUR", "GBP", "IDRT", "NGN", "PAX", "RUB", "TUSD", "TRY", "UAH", "USDC", "ZAR", "BUSD"}
)

// Config holds app arguments
type Config struct {
	tf    slice
	quote string
	force string // only for one
	// chart settings
	limit int
	ctype string
}

type slice []string

func (i *slice) String() string {
	return fmt.Sprintf("%v", *i)
}
func (i *slice) Set(value string) error {
	for _, v := range strings.Split(value, ",") {
		*i = append(*i, v)
	}
	return nil
}

var symbols []string

func main() {
	// log.Fatalln(sentiment.GetSentiment(0))
	// ----------------------------------------------------------------------------------------------
	// shutdown properly
	// ----------------------------------------------------------------------------------------------
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	go func() { log.Fatalln(<-interrupt) }()

	// ----------------------------------------------------------------------------------------------
	// config from args
	// ----------------------------------------------------------------------------------------------
	flag.Var(&conf.tf, "tf", "timeframes 4h or 1d,4h,1h...")
	flag.StringVar(&conf.quote, "quote", "USDT", "build symbols from quote")
	flag.StringVar(&conf.force, "force", "", "force")

	flag.IntVar(&conf.limit, "limit", 210, "max bars (0=no limit)")
	flag.StringVar(&conf.ctype, "ctype", "candlestick", "chartType: candlestick|ohlc|line|spline")
	flag.Parse()

	// ----------------------------------------------------------------------------------------------
	// chart settings (highcharts)
	// ----------------------------------------------------------------------------------------------
	chart.Type = charts.ChartType(conf.ctype)
	chart.EMA = []int{20, 50} // only working ikognito somtimes.
	// ----------------------------------------------------------------------------------------------
	// check for some errors in tmeframe arg
	// ----------------------------------------------------------------------------------------------
	var tfFix []string
	for _, v := range conf.tf {
		if history.TFs(history.TFi(v)) == "" {
			log.Fatalln("unknown timeframe")
			continue
		}
		fix := history.TFs(history.TFi(v))
		tfFix = append(tfFix, fix)
	}
	if len(tfFix) == 0 {
		tfFix = append(tfFix, "1d")
	}
	conf.tf = tfFix
	// ----------------------------------------------------------------------------------------------
	//create slice of charts we will be loading
	// ----------------------------------------------------------------------------------------------
	symbols = []string{conf.force}
	if conf.force == "" {
		var err error
		symbols, err = MakeSymbolMultiTimeframe(conf.quote, conf.tf...)
		if err != nil {
			log.Fatal("could not make symbols:", err)
		}
	}

	log.Println("initalizing...")
	// ----------------------------------------------------------------------------------------------
	// add the downloader interface Binance
	// ----------------------------------------------------------------------------------------------
	hist.Downloader = &Binance{}
	// ----------------------------------------------------------------------------------------------
	// load sybols
	// ----------------------------------------------------------------------------------------------
	hist.Load(symbols...)
	// ----------------------------------------------------------------------------------------------
	// keep looking for new bars and keep updating the hist struct
	// ----------------------------------------------------------------------------------------------
	hist.Update(true)
	// ----------------------------------------------------------------------------------------------
	// add strategy to event listener
	// ----------------------------------------------------------------------------------------------
	// evl.Add(strat)
	// ----------------------------------------------------------------------------------------------
	// start event listener
	// ----------------------------------------------------------------------------------------------
	// evl.Start(hist, events)

	// ----------------------------------------------------------------------------------------------
	// http routes for visual results and backtesting
	// ----------------------------------------------------------------------------------------------
	http.HandleFunc("/", httpIndex)
	http.HandleFunc("/test", httpTest)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// run strategy on current bar and plot event on chart
func httpIndex(w http.ResponseWriter, r *http.Request) {
	// limit bars
	if conf.limit > 0 {
		hist.Limit(conf.limit)
	}

	// build charts
	c, err := chart.BuildCharts(hist.Bars, *events...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(c)
}

// backtest strategy and plot events on chart
func httpTest(w http.ResponseWriter, r *http.Request) {
	// limit bars
	if conf.limit > 0 {
		hist.Limit(conf.limit)
	}

	// run strategy backtest on all data
	ev, err := hist.Test(strat, hist.FirstTime(), hist.LastTime())
	if err != nil {
		log.Fatal(err)
	}

	// build charts
	c, err := chart.BuildCharts(hist.Bars, ev...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(c)
}

// Power location
type Power struct{}

// Event Signals ..
func (s *Power) Event(bars history.Bars) (history.Event, bool) {
	var event history.Event

	if 60 > len(bars) {
		return event, false
	}

	// EXCLUDE SYMBOLS PRICES MATCHING PREFEX "0.000000xx"
	if p := strconv.FormatFloat(bars[0].O(), 'f', -1, 64); len(p) >= 7 {
		if "0.00000" == p[:7] {
			return event, false
		}
	}

	// --------------

	isNear := func(src, dest, r float64) bool {
		return math.Abs(src-dest) < r
	}

	mid := bars[0:20].EMA(3)
	slow := bars[0:50].EMA(3)
	midSlope := (bars[0:20].EMA(3) / bars[1:21].EMA(3)) - 1
	slowSlope := (bars[0:50].EMA(3) / bars[1:51].EMA(3)) - 1
	pp := mid > slow
	atr := bars[1:32].ATR()

	var ma = make(map[int]int, 0)
	for i := 0; i < 5; i++ {
		for n, v := range []float64{bars[i : i+20].EMA(3), bars[i : i+50].EMA(3)} {
			if bars[i].L() < v && bars[i].C() > v {
				ma[n]++
			}
		}
	}

	if //bars[0].Bullish() &&
	(bars[0].Range() > atr || bars[0].Bullish()) &&

		bars[0].C()-bars[0:10].Lowest(history.L) < 2*atr &&

		// bars[0].L() <= mid && bars[0].C() > mid &&
		// bars[0].L() <= slow && bars[0].C() > slow &&

		(ma[0] >= 3 || ma[1] >= 3) &&
		(bars[0].L() <= mid && bars[0].C() > mid || bars[0].L() <= slow && bars[0].C() > slow) &&

		bars[0].C() > bars[bars.LastBearIdx()].H() &&

		midSlope >= 0 &&
		slowSlope >= 0 &&
		isNear(mid, slow, .6*atr) &&
		(isNear(bars[0].O(), mid, .3*atr) || isNear(bars[0].O(), slow, .3*atr)) &&
		pp {

		event.Add(history.MARKET_BUY, "POWER LOCATION", bars[0].Time, bars[0].O())
		return event, true
	}

	return event, false
}

// Power2 location
type Power2 struct{}

// Event Signals ..
func (s *Power2) Event(bars history.Bars) (history.Event, bool) {
	var event history.Event

	if 150 > len(bars) {
		return event, false
	}

	// EXCLUDE SYMBOLS PRICES MATCHING PREFEX "0.000000xx"
	if p := strconv.FormatFloat(bars[0].O(), 'f', -1, 64); len(p) >= 7 {
		if "0.00000" == p[:7] {
			return event, false
		}
	}

	// --------------

	isNear := func(src, dest, r float64) bool {
		return math.Abs(src-dest) < r
	}

	var ma = make(map[int][]float64, 3)
	for i := 0; i < 30; i++ {
		for n, v := range []float64{bars[i : i+10].EMA(3), bars[i : i+20].EMA(3), bars[i : i+50].EMA(3)} {
			ma[n] = append(ma[n], v)
		}
	}

	var slope = make(map[int][]float64, 3)
	for i := 0; i < 30; i++ {
		for n, v := range []float64{(bars[i:i+10].EMA(3) / bars[i+1:i+11].EMA(3)) - 1, (bars[i:i+20].EMA(3) / bars[i+1:i+21].EMA(3)) - 1, (bars[i:i+50].EMA(3) / bars[i+1:i+51].EMA(3)) - 1} {
			slope[n] = append(slope[n], v)
		}
	}

	atr := bars[1:32].ATR()

	if bars[0].Bullish() &&

		bars[0].L() <= ma[1][0] && bars[0].C() > ma[1][0] &&
		bars[0].C() > bars[1:3].Highest(history.H) &&

		bars[0].O()-bars[1:7].Lowest(history.O) < atr &&

		(isNear(bars[0].O(), ma[1][0], 0.3*atr) || isNear(bars[0].O(), ma[2][0], 0.3*atr)) &&
		ma[1][0] > ma[2][0] {

		event.Add(history.MARKET_BUY, "POWER2", bars[0].Time, bars[0].O())
		return event, true
	}

	return event, false
}
