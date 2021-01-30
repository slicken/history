package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/slicken/history/charts"

	"github.com/slicken/history"
)

var (
	data   = new(history.Data)          // this is where we store all historys/bars and there settings
	events = new(history.Events)        // results (events) from strategy that we can later can plot
	evl    = new(history.EventListener) // we add our strategy to eventlistener witch is looking at the data

	strat = &Signals{sh: 0}       // strategy for this example
	chart = charts.DefaultChart() // small chart module (highcharts.com)
	conf  = new(Config)           // config from args
)

// Config ..
type Config struct {
	tf    string
	quote string
	// chart settings
	limit  int
	ctype  string
	volume bool
}

func main() {
	// ----------------------------------------------------------------------------------------------
	// shutdown properly
	// ----------------------------------------------------------------------------------------------
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	go func() { log.Fatalln(<-interrupt) }()

	// ----------------------------------------------------------------------------------------------
	// config from args
	// ----------------------------------------------------------------------------------------------
	flag.StringVar(&conf.tf, "tf", "", "timeframe")
	flag.StringVar(&conf.quote, "quote", "BTC", "build symbols from quote")

	flag.IntVar(&conf.limit, "limit", 210, "max bars")
	flag.StringVar(&conf.ctype, "ctype", "candlestick", "chartType: candlestick|ohlc|line|spline")
	flag.BoolVar(&conf.volume, "vol", true, "volume axis")
	flag.Parse()

	// ----------------------------------------------------------------------------------------------
	// check for some errors in tmeframe arg
	// ----------------------------------------------------------------------------------------------
	if history.Tfs(history.Tf(conf.tf)) == "" {
		log.Fatalln("unknown timeframe")
	}
	conf.tf = history.Tfs(history.Tf(conf.tf))
	// ----------------------------------------------------------------------------------------------
	//create slice of charts we will be loading
	// ----------------------------------------------------------------------------------------------
	symbols, err := MakeSymbolTimeframe(conf.quote, conf.tf)
	if err != nil {
		log.Fatal("could not make symbols:", err)
	}

	// symbols := []string{"BTCUSDT", "ETHBTC", "DOTBTC", "XMRBTC", "BNBBTC", "LTCBTC", "ENJBTC", "ADABTC", "RENBTC"}
	// var tmp []string
	// for _, sym := range symbols {
	// 	s := fmt.Sprintf("%v%v", sym, conf.tf)
	// 	fmt.Println(s)
	// 	tmp = append(tmp, s)
	// }
	// symbols = tmp
	// ----------------------------------------------------------------------------------------------
	// chart settings (highcharts)
	// ----------------------------------------------------------------------------------------------
	chart.Type = charts.ChartType(conf.ctype)
	chart.Volume = conf.volume
	chart.SMA = []int{20, 40}

	log.Println("initalizing...")
	// ----------------------------------------------------------------------------------------------
	// add the downloader interface Binance
	// ----------------------------------------------------------------------------------------------
	data.Downloader = &Binance{}
	// ----------------------------------------------------------------------------------------------
	// load sybols
	// ----------------------------------------------------------------------------------------------
	data.Load(symbols) //[]string{"BTCUSDT1m"})
	// ----------------------------------------------------------------------------------------------
	// keep looking for new bars and keep updating the data struct
	// ----------------------------------------------------------------------------------------------
	data.Update(true)
	// ----------------------------------------------------------------------------------------------
	// add strategy to event listener
	// ----------------------------------------------------------------------------------------------
	// evl.Add(&MaCrossing{8, 20})
	// evl.Add(&TD{1})
	// evl.Add(&Engulf{1})
	// evl.Add(&MaCrossing{10, 21})
	evl.Add(strat)
	// ----------------------------------------------------------------------------------------------
	// start event listener
	// ----------------------------------------------------------------------------------------------
	evl.Start(data, events)

	// ----------------------------------------------------------------------------------------------
	// http routes for visual results and backtesting
	// ----------------------------------------------------------------------------------------------
	http.HandleFunc("/", httpIndex)
	http.HandleFunc("/sort", httpSort)
	http.HandleFunc("/test", httpTest)
	http.HandleFunc("/events", httpEvents)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// run strategy on current bar and plot event on chart
func httpIndex(w http.ResponseWriter, r *http.Request) {
	// limit data

	min := time.Duration(conf.limit*int(history.Tf(conf.tf))) * time.Minute
	start := time.Now().Add(-min)
	datacut := data.TimeSpan(start, time.Now())

	ev := *events
	// build chart
	c, err := chart.BuildChartEvents(datacut, ev, conf.tf)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(c)
}

// sort events
func httpSort(w http.ResponseWriter, r *http.Request) {
	// limit data
	min := time.Duration(conf.limit*int(history.Tf(conf.tf))) * time.Minute
	start := time.Now().Add(-min)
	datacut := data.TimeSpan(start, time.Now())

	ev := *events
	// sort event by text (strategies)
	var eventList = make(map[string]history.Events)
	for _, event := range ev {
		t := event.Text
		eventList[t] = append(eventList[t], event)
	}

	var chartList = make(map[string][]byte)
	for k, v := range eventList {
		c, err := chart.BuildChartEvents(datacut, v, conf.tf)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		chartList[k] = c
	}

	// make data and insert topic
	var b []byte
	for k, v := range chartList {
		// add a topic
		topic := fmt.Sprintf(`<h1>%s (%d)</h1>`, k, len(eventList[k]))
		b = append(b, []byte(topic)...)
		// add charts
		b = append(b, v...)
	}

	w.Write(b)
}

// backtest strategy and plot events on chart
func httpTest(w http.ResponseWriter, r *http.Request) {
	// limit data
	min := time.Duration(conf.limit*int(history.Tf(conf.tf))) * time.Minute
	start := time.Now().Add(-min)
	datacut := data.TimeSpan(start, time.Now())

	// run strategy backtest on all data
	ev, err := datacut.Test(strat, datacut.FirstTime(), datacut.LastTime())
	if err != nil {
		log.Fatal(err)
	}

	// build charts
	c, err := chart.BuildChartEvents(datacut, ev, conf.tf)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(c)
}

// list all events
func httpEvents(w http.ResponseWriter, r *http.Request) {
	// var buf []byte

	// for _, event := range *events {
	// 	buf = append(buf, fmt.Sprintf("%s%s %s %s %.8f %v\n", event.Symbol, event.Timeframe, event.Type, event.Text, event.Price, event.Time)...)
	// }

	// w.Write(buf)

	b, err := json.MarshalIndent(events, "", "\t")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	w.Write(b)
}

// Engulf ..
type Engulf struct {
	shift int
}

// Event Engulf signal
func (s *Engulf) Event(bars history.Bars) (history.Event, bool) {
	var event history.Event

	if s.shift+3 > len(bars) {
		return event, false
	}

	if bars[s.shift:s.shift+2].IsEngulfBuy() && bars[s.shift].Bullish() {
		event.Add("BUY", "ENGULF", bars[0].Time, bars[0].O())
		return event, true
	}
	if bars[s.shift:s.shift+2].IsEngulfSell() && bars[s.shift].Bearish() {
		event.Add("SELL", "ENGULF", bars[0].Time, bars[0].O())
		return event, true
	}

	return event, false
}

// TD Sequential 9
type TD struct {
	shift int
}

// Event TD ..
func (s *TD) Event(bars history.Bars) (history.Event, bool) {
	var event history.Event

	if s.shift+30 > len(bars) {
		return event, false
	}

	td := bars[s.shift : s.shift+30].TD()

	if td < 0 {
		if td == -2 {
			event.Add("BUY", "TDP", bars[0].Time, bars[0].O())
		} else {
			event.Add("BUY", "TD", bars[0].Time, bars[0].O())
		}
		return event, true
	}
	if td > 0 {
		if td == 2 {
			event.Add("SELL", "TDP", bars[0].Time, bars[0].O())
		} else {
			event.Add("SELL", "TD", bars[0].Time, bars[0].O())
		}
		return event, true
	}

	return event, false
}

// MaCrossing simple ma crossover NOW
type MaCrossing struct {
	fast int
	slow int
}

// Event Test ..
func (s *MaCrossing) Event(bars history.Bars) (history.Event, bool) {
	var event history.Event

	if s.fast == 0 {
		s.fast = 10
	}
	if s.slow == 0 {
		s.slow = 21
	}

	if len(bars) < s.slow+1 {
		return event, false
	}

	fast0 := bars[1 : 1+s.fast].EMA(history.C)
	fast1 := bars[2 : 2+s.fast].EMA(history.C)
	slow0 := bars[1 : 1+s.slow].EMA(history.C)
	slow1 := bars[2 : 2+s.slow].EMA(history.C)

	name := fmt.Sprintf("EMA CROSS %d&%d", s.fast, s.slow)

	if slow1 >= fast1 && slow0 < fast0 {
		event.Add("BUY", name, bars[0].Time, bars[0].O())
		return event, true
	}

	if fast1 >= slow1 && fast0 < slow0 {
		event.Add("SELL", name, bars[0].Time, bars[0].O())
		return event, true
	}

	return event, false
}

// CountDecimal counts decimals
func CountDecimal(v float64) int {
	s := strconv.FormatFloat(v, 'f', -1, 64)
	i := strings.IndexByte(s, '.')
	if i > -1 {
		return len(s) - i - 1
	}
	return 0
}

// Signals many
type Signals struct {
	sh int
}

// Event Test ..
func (s *Signals) Event(bars history.Bars) (history.Event, bool) {
	var event history.Event

	if s.sh+44 > len(bars) {
		return event, false
	}
	// EXCLUDE SYMBOLS PRICE "0.000000xx"
	if p := strconv.FormatFloat(bars[0].O(), 'f', -1, 64); len(p) >= 7 {
		if "0.00000" == p[:7] {
			return event, false
		}
	}

	o := bars[s.sh].O()
	// h := bars[s.sh].H()
	l := bars[s.sh].L()
	c := bars[s.sh].C()
	// body := bars[1].Body()

	// ema21 := bars[1:22].EMA(history.C)

	sma20 := bars[s.sh : s.sh+20].SMA(history.C)
	preslope20 := (bars[s.sh+4:s.sh+24].SMA(history.C) / bars[s.sh+5:s.sh+25].SMA(history.C)) - 1
	slope20 := (bars[s.sh:s.sh+20].SMA(history.C) / bars[s.sh+1:s.sh+21].SMA(history.C)) - 1

	slope40 := (bars[1:41].SMA(history.C) / bars[2:42].SMA(history.C)) - 1
	// preslope40 := (bars[5:45].SMA(history.C) / bars[6:46].SMA(history.C)) - 1

	atr := bars[s.sh+1 : s.sh+15].ATR(history.C)
	// wick := bars[1].WickDn()

	//	if wickdn > 2.8*atr { // && wickdn > body {

	// if wick > 1.5*atr {
	// 	event.Add("BUY", "WICK", bars[0].Time, bars[1].O())
	// 	return event, true
	// }

	// SLOPE
	if preslope20 > 0 && slope20 > 0 && slope20 > preslope20 && slope40 > 0 &&
		// ((l < sma20 && c > sma20) || (l < ema21 && c > ema21)) {
		// l < sma20 && c > sma20 {
		o-atr > sma20 && l <= sma20 && c > sma20 {

		event.Add("BUY", "SLOPE", bars[s.sh].Time, bars[s.sh].C())
		return event, true
	}

	// SLOPE2
	// if preslope > 0 && slope > 0 &&  {
	// 	event.Add("BUY", "TEST", bars[0].Time, bars[1].O())
	// 	return event, true
	// }

	// // WICK20
	// if o > sma20 && l < sma20 && c > sma20 {
	// 	event.Add("BUY", "WICK MA20", bars[0].Time, bars[1].O())
	// 	return event, true
	// }

	// VOLUME CLIMAX
	// if bars[1].V() > 2.8*bars[2:37].EMA(history.V) {
	// 	event.Add("BUY", "VOL CLIMAX", bars[1].Time, bars[1].O())
	// 	return event, true
	// }

	return event, false
}
