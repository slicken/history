package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/slicken/history/charts"

	"github.com/slicken/history"
)

var (
	data   = new(history.Data)          // this is where we store all historys/bars and there settings
	events = new(history.Events)        // results (events) from strategy that we can later can plot
	evl    = new(history.EventListener) // we add our strategy to eventlistener witch is looking at the data

	strat = &MyStrategy3{}        // strategy for this example
	chart = charts.DefaultChart() // small chart module (highcharts.com)
	conf  = new(Config)           // config from args
)

// Config ..
type Config struct {
	tf          string
	quote       string
	emax, limit int
	arg         float64
	// chart settings
	ctype  string
	volume bool
}

func main() {
	// shutdown properly
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	go func() {
		log.Println(<-interrupt)
		os.Exit(0)
	}()

	// config from args
	flag.StringVar(&conf.tf, "tf", "1d", "timeframe")
	flag.StringVar(&conf.quote, "quote", "BTC", "build symbols from quote")
	flag.Float64Var(&conf.arg, "arg", 0, "argument")
	flag.IntVar(&conf.emax, "emax", 50, "max events per chart")
	flag.IntVar(&conf.limit, "limit", 210, "max bars")
	flag.StringVar(&conf.ctype, "ctype", "candlestick", "chartType: candlestick|ohlc|line|spline")
	flag.BoolVar(&conf.volume, "vol", true, "volume axis")
	flag.Parse()

	// strategu with arg for this example
	strat = &MyStrategy3{arg: conf.arg}

	// check for some errors in tmeframe arg
	if history.Tfs(history.Tf(conf.tf)) == "" {
		log.Fatalln("unknown timeframe")
	}
	conf.tf = history.Tfs(history.Tf(conf.tf))

	//create slice of charts we will be loading
	symbols, err := MakeSymbolTimeframe(conf.quote, conf.tf)
	if err != nil {
		log.Fatal("could not make symbols:", err)
	}

	// chart settings
	chart.Type = charts.ChartType(conf.ctype)
	chart.Volume = conf.volume
	chart.EMA = []int{8}

	// ----------------------------------------------------------------------------------------------
	log.Println("initalizing...")

	// add the downloader interface Binance
	data.Downloader = &Binance{}
	// load sybols
	data.Load(symbols) // data.Load([]string{"BTCUSDT1h"})
	// keep looking for new bars and keep updating data struct and files
	data.Update(true)

	// add strategy to event listener
	evl.Add(strat)
	evl.List()
	// start event listener
	evl.Start(data, events)

	// http routes for visual results and backtesting
	http.HandleFunc("/", httpIndex)
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
	c, err := chart.BuildChartEvents(datacut, ev)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(c)
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
	c, err := chart.BuildChartEvents(datacut, ev)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(c)
}

// list all events
func httpEvents(w http.ResponseWriter, r *http.Request) {
	var buf []byte

	for _, event := range *events {
		buf = append(buf, fmt.Sprintf("%s%s %s %s %.8f %v\n", event.Symbol, event.Timeframe, event.Type, event.Text, event.Price, event.Time)...)
	}

	w.Write(buf)
}

// MyStrategy create event at 50% of big wicks
type MyStrategy struct{}

// Event Test ..
func (s *MyStrategy) Event(bars history.Bars) (history.Event, bool) {
	var event history.Event

	if 30 > len(bars) {
		return event, false
	}

	var i int
	var buyprice, sellprice float64
	// how many bars back is a valid wick
	for i = 1; i < 4; i++ {
		atr := bars[i+1 : i+13].ATR(history.C)
		wickup := bars[i].WickUp()
		wickdn := bars[i].WickDn()
		//body := bars[i].Body()

		if wickdn > 2.8*atr { // && wickdn > body {
			buyprice = bars[i].BodyLow() - (bars[i].BodyLow()-bars[i].Low)/2
			break
		}

		if wickup > 2.8*atr { //&& wickup > body {
			sellprice = bars[i].BodyHigh() + (bars[i].High-bars[i].BodyHigh())/2
			break
		}
	}

	if buyprice != 0 && bars[0].Low < buyprice {
		event.Add("BUY", "WICK", bars[0].Time, buyprice)
		return event, true
	}
	if sellprice != 0 && bars[0].High > sellprice {
		event.Add("SELL", "WICK", bars[0].Time, sellprice)
		return event, true
	}

	return event, false
}

// MyStrategy2 create event when big wick is seen
type MyStrategy2 struct{}

// Event Test ..
func (s *MyStrategy2) Event(bars history.Bars) (history.Event, bool) {
	var event history.Event

	if 15 > len(bars) {
		return event, false
	}

	atr := bars[2:14].ATR(history.C)

	if bars[1].WickDn() > 2.8*atr {
		event.Add("BUY", "WICK", bars[1].Time, 0)
		return event, true
	}

	return event, false
}

// MyStrategy3 percent move
type MyStrategy3 struct {
	arg float64
}

// Event Test ..
func (s *MyStrategy3) Event(bars history.Bars) (history.Event, bool) {
	var event history.Event

	wick := 15.0
	if s.arg != 0. {
		wick = s.arg
	}
	//barsPercent := 1

	if len(bars) < 3 {
		return event, false
	}

	/* var u int
	for i := 1; i < lookBack; i++ {

		if wickPercent < (100 * (bars[i].WickDn() / bars[i].O())) {
			u++
		}
	}

	if u == 0 {
		return event, false
	}

	//if 100*u/lookBack > barsPercent {
	*/
	if wick < (100 * (bars[1].WickDn() / bars[1].O())) {
		//event.Add("BUY", fmt.Sprintf("%d%%", 100*u/lookBack), bars[1].Time, bars[1].C())
		event.Add("BUY", fmt.Sprintf("%f", 100*bars[1].WickDn()/bars[1].O()), bars[1].T(), bars[1].O())
		return event, true
	}

	return event, false
}

// MyStrategy4 .. tf=15m
type MyStrategy4 struct{}

// Event Test ..
func (s *MyStrategy4) Event(bars history.Bars) (history.Event, bool) {
	var event history.Event

	if len(bars) < 50 {
		return event, false
	}

	minPercent := 4.0

	//	ema8 := bars[1:9].EMA(history.C)
	ema8_2 := bars[2:10].EMA(history.C)

	low2 := ema8_2 - ((minPercent * bars[2].O()) / 100)
	high2 := ema8_2 + ((minPercent * bars[2].O()) / 100)

	if /*bars[1].O() < low && */ bars[2].H() < low2 && bars[1].H() > bars[2].H() {
		event.Add("BUY", "MA", bars[1].Time, bars[1].O())
		return event, true
	}

	if /*bars[1].O() > high && */ bars[2].L() > high2 && bars[1].L() < bars[2].L() {
		event.Add("SELL", "MA", bars[1].Time, bars[1].O())
		return event, true
	}

	return event, false
}

// FindSpikeSymbols ..
func FindSpikeSymbols(tf string) []string {
	var m = make(map[string]float64)
	_ = m

	// limit data
	min := time.Duration(conf.limit*int(history.Tf(tf))) * time.Minute
	start := time.Now().Add(-min)
	datacut := data.TimeSpan(start, time.Now())

	// run strategy backtest on all data
	ev, err := datacut.Test(&MyStrategy3{}, datacut.FirstTime(), datacut.LastTime())
	if err != nil {
		log.Fatal(err)
	}

	// make slice
	var ret = []string{}
	for _, e := range ev {
		ret = append(ret, e.Symbol)
	}

	return ret
}
