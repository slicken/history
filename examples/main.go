package main

import (
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

	"github.com/slicken/history"
	"github.com/slicken/history/charts"
)

var (
	data   = new(history.Data)          // this is where we store all historys/bars and there settings
	events = new(history.Events)        // results (events) from strategy that we can later can plot
	evl    = new(history.EventListener) // we add our strategy to eventlistener witch is looking at the data

	//strat = &Gainers{days: 30}    // strategy for this example
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
	chart.SMA = []int{20, 200} //, 50, 100} // only working ikognito somtimes.
	// ----------------------------------------------------------------------------------------------
	// check for some errors in tmeframe arg
	// ----------------------------------------------------------------------------------------------
	var tfFix []string
	for _, v := range conf.tf {
		if history.Tfs(history.Tf(v)) == "" {
			log.Fatalln("unknown timeframe")
		}
		fix := history.Tfs(history.Tf(v))
		tfFix = append(tfFix, fix)
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
	data.Downloader = &Binance{}
	// ----------------------------------------------------------------------------------------------
	// load sybols
	// ----------------------------------------------------------------------------------------------
	// symbols = []string{"BTCUSDT15m", "BTCUSDT1h", "BTCUSDT4h", "BTCUSDT1d"}
	data.Load(symbols)
	// ----------------------------------------------------------------------------------------------
	// keep looking for new bars and keep updating the data struct
	// ----------------------------------------------------------------------------------------------
	data.Update(true)
	// ----------------------------------------------------------------------------------------------
	// add strategy to event listener
	// ----------------------------------------------------------------------------------------------
	// evl.Add(strat)
	// ----------------------------------------------------------------------------------------------
	// start event listener
	// ----------------------------------------------------------------------------------------------
	// evl.Start(data, events)

	// ----------------------------------------------------------------------------------------------
	// http routes for visual results and backtesting
	// ----------------------------------------------------------------------------------------------
	http.HandleFunc("/", httpIndex)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func httpRoutes(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "not allowed", http.StatusMethodNotAllowed)
		return
	}

	symbol := r.URL.Path[1:]
	symbol = strings.Replace(symbol, "favicon.ico", "", -1)

	fmt.Println(symbol)
	return
}

func httpIndex(w http.ResponseWriter, r *http.Request) {
	arg := r.URL.Path[1:]

	var err error
	var n int
	n, err = strconv.Atoi(arg)
	if err != nil {
		n = 30
	}

	// clear events
	eventsNew := new(history.Events)
	events = eventsNew
	evlNew := new(history.EventListener)
	evl = evlNew
	evl.Add(&Gainers{days: n})
	evl.Start(data, events)

	data.CUpdate()
	time.Sleep(1 * time.Second)

	events := *events
	events.Sort()

	var datacut = new(history.Data)
	i := 0
	for _, e := range events {
		if i > 10 {
			break
		}
		// debug
		// fmt.Println(e.Symbol+e.Timeframe, "\t", e.Value)

		for _, hist := range data.History {
			if hist.Symbol == e.Symbol && hist.Timeframe == e.Timeframe {
				datacut.History = append(datacut.History, hist)
				// datacut.History = append([]*history.History{hist}, datacut.History...)
			}
		}
		i++
	}

	// limit data
	datacut = datacut.Limit(conf.limit)
	c, err := chart.BuildCharts(datacut)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// add html topic
	var b []byte
	topic := fmt.Sprintf(`<h1>TOP GAINERS PAST %d DAYS</h1>`, n)
	b = append(b, []byte(topic)...)
	b = append(b, c...)

	w.Write(b)
}

// Gainers
type Gainers struct {
	days int
}

// Event Signals ..
func (s *Gainers) Event(bars history.Bars) (history.Event, bool) {
	var event history.Event

	if s.days+1 > len(bars) {
		return event, false
	}

	gain := 100 * ((bars[0].C() - bars[0:s.days].Lowest(history.O)) / bars[0:s.days].Lowest(history.C))

	if gain > 0 {
		event.Add("BUY", fmt.Sprintf("%.1f", gain), bars.LastBar().T(), bars.LastBar().H())
		event.Value = gain
		return event, true
	}
	// if gain < 0 {
	// 	event.Add("SELL", fmt.Sprintf("%.1f", gain), bars.LastBar().T(), bars.LastBar().H())
	// 	event.Value = gain
	// 	return event, true
	// }

	return event, false
}

func Stats() {
	var okokokokoko = []time.Weekday{
		time.Sunday,
		time.Monday,
		time.Tuesday,
		time.Wednesday,
		time.Thursday,
		time.Friday,
		time.Saturday,
	}

	var Stats = make(map[string]map[time.Weekday]float64, 0)

	for _, s := range data.List() {

		sym, tf := history.Split(s)
		if sym == "" || tf == "" {
			continue
		}
		bars := data.Bars(sym, tf)
		if 100 > len(bars) {
			continue
		}

		var Day = make(map[time.Weekday]float64, 0)

		for _, b := range bars {
			Day[b.T().Weekday()] += b.PercMove()
		}

		for _, i := range okokokokoko {
			Day[i] /= float64(len(bars)) / float64(len(okokokokoko))
		}

		Stats[sym+tf] = Day

	}

	var stats string

	var Sum = make(map[time.Weekday]float64, 0)
	for k, v := range Stats {

		stats += fmt.Sprintln(k, "-----------------------------------------")

		for i, j := range v { //okokokokoko {
			stats += fmt.Sprintf("%-10s  %-10.2f\n", i, j)
			Sum[i] += j
		}
	}

	stats += "\n"

	for i, _ := range Sum {
		Sum[i] /= float64(len(Stats))

		stats += fmt.Sprintln(i, Sum[i])
	}

	fmt.Println(stats)

	// write to file
	f, err := os.Create("stats.txt")
	if err != nil {
		panic(err)
	}

	defer f.Close()
	if _, err := f.WriteString(stats); err != nil {
		panic(err)
	}

}
