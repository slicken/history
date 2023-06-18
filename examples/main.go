package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"syscall"

	"github.com/slicken/history"
	"github.com/slicken/history/highcharts"
)

var (
	hist   = new(history.History)       // this is where we store all historys/bars and there settings
	evl    = new(history.EventListener) // we add our strategy to eventlistener witch is looking at the hist
	events = new(history.Events)        // we store our events here, if we want to save them

	strat = &Engulf{}
	//	strat = new(test)             // strategy for this example
	chart = highcharts.DefaultChart() // we use highcharts for plotting

	conf = new(Config) // config from args
	// other
	symbols []string
	// topPreformers []string
)

// Config holds app arguments
type Config struct {
	// symbols
	tf    string
	quote string
	// top preformers - write to file
	file bool
	// chart settings
	limit int
	ctype string
	// force is for loading one symbol only
	force string
}

// type slice []string

// func (i *slice) String() string {
// 	return fmt.Sprintf("%v", *i)
// }
// func (i *slice) Set(value string) error {
// 	for _, v := range strings.Split(value, ",") {
// 		*i = append(*i, v)
// 	}
// 	return nil
// }

func main() {
	// log.SetOutput(io.Discard)
	// ----------------------------------------------------------------------------------------------
	// shutdown properly
	// ----------------------------------------------------------------------------------------------
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	go func() { log.Fatalln(<-interrupt) }()
	// ----------------------------------------------------------------------------------------------
	// example of app args
	// ----------------------------------------------------------------------------------------------
	flag.StringVar(&conf.tf, "tf", "1d", "timeframe")
	flag.StringVar(&conf.quote, "quote", "USDT", "build pairs from quote")

	flag.BoolVar(&conf.file, "file", false, "save '/top/' preformers to file")

	flag.IntVar(&conf.limit, "limit", 300, "limit bars (0=off)")
	flag.StringVar(&conf.ctype, "ctype", "candlestick", "chartType: candlestick|ohlc|line|spline")

	flag.StringVar(&conf.force, "force", "", "force one symbol")
	flag.Parse()
	// ----------------------------------------------------------------------------------------------
	// chart settings (highcharts)
	// ----------------------------------------------------------------------------------------------
	chart.Type = highcharts.ChartType(conf.ctype)
	chart.SMA = []int{20, 200}
	// ----------------------------------------------------------------------------------------------
	// create list of symbols form data grabbet from my exchanges
	// ----------------------------------------------------------------------------------------------
	symbols = []string{conf.force}
	if conf.force == "" {
		var err error
		symbols, err = MakeSymbolMultiTimeframe(conf.quote, conf.tf)
		if err != nil {
			log.Fatal("could not make symbols:", err)
		}
	}

	log.Println("initalizing...")
	// ----------------------------------------------------------------------------------------------
	// add a downloader to the interface.
	// ----------------------------------------------------------------------------------------------
	hist.Downloader = &Binance{}
	// ----------------------------------------------------------------------------------------------
	// change the default directory to store history data from exchange
	// ----------------------------------------------------------------------------------------------
	hist.SetDataDir("download")
	// ----------------------------------------------------------------------------------------------
	// update new history bars when there is a fresh bar for your symbol
	// ----------------------------------------------------------------------------------------------
	hist.Update(true)
	// ----------------------------------------------------------------------------------------------
	// load symbols. use a list to load multiple.
	// ----------------------------------------------------------------------------------------------
	hist.Load(symbols...)
	// ----------------------------------------------------------------------------------------------
	// limit bars
	// ----------------------------------------------------------------------------------------------
	hist.Limit(conf.limit)
	// ----------------------------------------------------------------------------------------------
	// add strategy to event listener
	// ----------------------------------------------------------------------------------------------
	// bars := hist.Bars("BTCUSDT1d").Reverse()[:10]
	// fmt.Println()
	// fmt.Println()
	// fmt.Println(bars.Reverse())
	// // os.Exit(0)
	// os.Exit(0)
	fmt.Printf("\n!http len(hist.Map)=%d\n\n", len(hist.Map()))

	evl.Add(strat)
	// ----------------------------------------------------------------------------------------------
	// start event listener
	// ----------------------------------------------------------------------------------------------
	evl.Start(hist, events)
	// ----------------------------------------------------------------------------------------------
	// http routes for visual results and backtesting
	// ----------------------------------------------------------------------------------------------

	http.HandleFunc("/", httpIndex)
	http.HandleFunc("/test", httpTest)
	http.HandleFunc("/backtest", httpBacktest)
	http.HandleFunc("/top/", httpTopPreformers)
	http.HandleFunc("/gainers/", httpTopGainers)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// run strategy on current bar and plot event on chart
func httpIndex(w http.ResponseWriter, r *http.Request) {
	// limit bars
	if conf.limit > 0 {
		hist.Limit(conf.limit)
	}

	// build charts
	var ev = history.Events{}
	c, err := chart.BuildCharts(hist.Map(), ev.Map())
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
	c, err := chart.BuildCharts(hist.Map(), ev.Map())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(c)
}

// Backtest strategy and plot events on chart
func httpBacktest(w http.ResponseWriter, r *http.Request) {
	// limit bars
	if conf.limit > 0 {
		hist.Limit(conf.limit)
	}

	// run strategy backtest on all data
	ev, err := hist.PTest(strat, hist.FirstTime(), hist.LastTime())
	if err != nil {
		log.Fatal(err)
	}

	// build charts
	c, err := chart.BuildCharts(hist.Map(), ev.Map())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(c)
}

// TopPreformers over 'url:8080/top/x' x = number of bars
func httpTopPreformers(w http.ResponseWriter, r *http.Request) {
	n := conf.limit
	// http://127.0.0.1/top/N	where N is number of bars
	if len(r.URL.Path) > 5 {
		if v, err := strconv.Atoi(r.URL.Path[5:]); err == nil {
			n = v
		}
	}

	// new copy of history, so we dont loose bars on histofy master data
	var copyHist = new(history.History)
	copyHist.Downloader = &Binance{}
	copyHist.Update(false)
	copyHist.Load(symbols...)

	// limit history if http://127.0.0.1/top/N is used
	if n > 0 && n != conf.limit {
		copyHist.Limit(n + 1)
	}

	var results = history.Events{}
	for symbol, bars := range copyHist.Map() {
		if event, ok := (&Gained{n, false}).Event(symbol, bars); ok && !results.Exists(event) {
			event.Pair, event.Timeframe = history.SplitPairTf(symbol)
			results = append(results, event)
		}
	}

	fmt.Println("-----------  events", len(results))

	// sort by price where the gains value is stored
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].Price > results[j].Price
	})

	for _, event := range results {
		fmt.Printf("%-12s %.2f %%\n", event.Pair+event.Timeframe, event.Price)
	}

	// build charts
	c, err := chart.BuildCharts(copyHist.Map(), results.Map())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(c)
}

// TopPreformers over 'url:8080/gainers/  shows top gainers for 180, 90, 30 days
func httpTopGainers(w http.ResponseWriter, r *http.Request) {
	n := []int{180, 90, 30}

	var results = history.Events{}
	for i := range n {
		for symbol, bars := range hist.Map() {
			if new, ok := (&Gained{n[i], false}).Event(symbol, bars); ok && !results.Exists(new) {
				new.Pair, new.Timeframe = history.SplitPairTf(symbol)
				results = append(results, new)
			}
		}
	}

	fmt.Println("-----------  events", len(results))

	// sort by price where the gains value is stored
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].Price > results[j].Price
	})

	var m = make(map[string]bool, 0)
	for _, event := range results {
		if _, found := m[event.Pair+event.Timeframe]; found {
			continue
		}
		m[event.Pair+event.Timeframe] = true
		fmt.Printf("%-12s %.2f %%\n", event.Pair, event.Price)
	}

	// build charts
	c, err := chart.BuildCharts(hist.Map(), results.Map())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(c)
}

// ----- STRATEGIES -----------

type Gained struct {
	Limit      int
	LowestOpen bool
}

// Event Signals ..
func (s *Gained) Event(symbol string, bars history.Bars) (history.Event, bool) {
	var event history.Event

	if s.Limit+1 > len(bars) {
		return event, false
	}

	var perc float64
	if s.LowestOpen {
		perc = 100 * ((bars[0].C() - bars[0:s.Limit].Lowest(history.O)) / bars[0:s.Limit].Lowest(history.O))
	} else {
		perc = 100 * ((bars[0].C() - bars[s.Limit].O()) / bars[s.Limit].O())
	}

	event.Type = history.OTHER
	event.Name = fmt.Sprintf("%.1f", perc)
	event.Time = bars.LastBar().T()
	event.Price = perc
	return event, true
}

// Power location
type Power struct {
	shift int
}

// Event Signals ..
func (s *Power) Event(symbol string, bars history.Bars) (history.Event, bool) {
	var event history.Event

	if 150 > len(bars) {
		return event, false
	}

	// EXCLUDE SYMBOLS PRICES MATCHING PREFEX "0.000000xx"
	if price := strconv.FormatFloat(bars[0].O(), 'f', -1, 64); len(price) >= 7 {
		if price[:7] == "0.00000" {
			return event, false
		}
	}

	// --------------

	// 0=10, 1=20, 2=50
	var ma = make(map[int][]float64, 3)
	for i := 0; i < 30; i++ {
		for n, v := range []float64{bars[i : i+10].SMA(3), bars[i : i+20].SMA(3), bars[i : i+50].SMA(3)} {
			ma[n] = append(ma[n], v)
		}
	}

	if bars[s.shift].Bullish() &&

		bars[s.shift].Range() > bars[s.shift:10+s.shift].ATR() &&
		bars[s.shift].L() <= ma[1][s.shift] && bars[s.shift].C() > ma[1][s.shift] &&
		bars[s.shift].C() > bars[s.shift+1:s.shift+4].Highest(history.H) &&

		bars[s.shift].O()-bars[s.shift+1:s.shift+7].Lowest(history.O) < bars[s.shift+1:s.shift+21].ATR() &&

		//ma[1][s.shift+1] > ma[2][s.shift+1] &&
		(ma[1][s.shift] > ma[1][s.shift+1]) || (ma[2][s.shift] > ma[2][s.shift+1]) &&

		//history.WithinRange(ma[1][s.shift], ma[2][s.shift], 2*bars[s.shift+1:s.shift+21].ATR()) &&
		history.WithinRange(ma[1][s.shift], bars[s.shift].O(), bars[s.shift+1:s.shift+21].ATR()) {

		event.Type = history.MARKET_BUY
		event.Name = "POWER BUY"
		event.Time = bars[0].T()
		event.Price = bars[0].O()
		return event, true
	}

	return event, false
}

// Engulf location
type Engulf struct{}

// Event Engulf ..
func (s *Engulf) Event(symbol string, bars history.Bars) (history.Event, bool) {
	var event history.Event

	if 210 > len(bars) {
		return event, false
	}
	// EXCLUDE SYMBOLS PRICES MATCHING PREFEX "0.000000xx"
	if price := strconv.FormatFloat(bars[0].O(), 'f', -1, 64); len(price) >= 7 {
		if price[:7] == "0.00000" {
			return event, false
		}
	}

	// --------------

	// 0=8, 1=20, 2=50
	var ma = make(map[int][]float64, 3)
	for i := 0; i < 10; i++ {
		for n, v := range []float64{
			bars[i : i+8].SMA(3),
			bars[i : i+20].SMA(3),
			bars[i : i+200].SMA(3)} {
			ma[n] = append(ma[n], v)
		}
	}

	var slope = make(map[int][]float64, 3)
	for i := 0; i < 10; i++ {
		for n, v := range []float64{
			(bars[i:i+8].SMA(3) / bars[i+1:i+9].SMA(3)) - 1,
			(bars[i:i+20].SMA(3) / bars[i+1:i+21].SMA(3)) - 1,
			(bars[i:i+200].SMA(3) / bars[i+1:i+201].SMA(3)) - 1} {
			slope[n] = append(slope[n], v)
		}
	}

	if bars.LastBearIdx() < 5 &&
		//bars[1].Bear() &&
		bars[0].C() > bars[bars.LastBearIdx()].H() &&
		//slope[1][0] > 0 &&
		ma[1][0] > ma[2][0] &&
		//		history.WithinRange(ma[1][0], bars[0].C(), bars[1:20].ATR()) &&
		bars[0].L() <= ma[1][0] && bars[0].C() > ma[1][0] {

		event.Type = history.MARKET_BUY
		event.Name = "ENGULF"
		event.Time = bars[0].T()
		event.Price = bars[0].O()
		return event, true
	}

	return event, false
}

// test strategy
type test struct{}

// Event Signals
func (s *test) Event(symbol string, bars history.Bars) (history.Event, bool) {
	event := history.NewEvent(symbol)
	// event.Name =
	// filter symbols with less bars and price prefix is equals "0.00000"
	if 210 > len(bars) {
		return event, false
	}
	if price := strconv.FormatFloat(bars[0].O(), 'f', -1, 64); len(price) >= 7 {
		if price[:7] == "0.00000" {
			return event, false
		}
	}
	//
	//
	//
	if bars[2].Bullish() && bars[2].Body() > bars[3:13].ATR() && bars[2].Range() < 4*bars[3:13].ATR() &&
		bars[1].Bear() && bars[1].BodyLow() > (bars[2].BodyLow()+bars[2].BodyHigh())*0.3 {

		event.Type = history.MARKET_BUY
		event.Name = "RBI"
		event.Time = bars[0].T()
		event.Price = bars[0].O()
		return event, true
	}

	return event, false

}
