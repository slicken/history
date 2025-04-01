package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/slicken/history"
	"github.com/slicken/history/highcharts"
)

var (
	hist          = new(history.History)       // main struct to control bars data
	eventListener = new(history.EventListener) // we add our strategy to eventlistener witch is looking at the hist
	events        = new(history.Events)        // we store our events here, if we want to save them
	strategy      = NewDoubleWick()            // improved power strategy with better signal generation
	chart         = highcharts.DefaultChart()  // we use highcharts for plotting

	config = new(Config) // store argument configurations for example app
	// other
	symbols []string // list of symbols to handle bars
)

// Config holds app arguments
type Config struct {
	// symbols
	tf    string
	quote string
	// chart settings
	limit int
	ctype string
	// force is for loading one symbol only
	force string
}

func main() {
	log.SetOutput(io.Discard)
	// ----------------------------------------------------------------------------------------------
	// shutdown properly
	// ----------------------------------------------------------------------------------------------
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	go func() { log.Fatalln(<-interrupt) }()
	// ----------------------------------------------------------------------------------------------
	// example of app args
	// ----------------------------------------------------------------------------------------------
	flag.StringVar(&config.tf, "tf", "1d", "timeframe")
	flag.StringVar(&config.quote, "quote", "USDT", "build pairs from quote")
	flag.IntVar(&config.limit, "limit", 300, "limit bars (0=off)")
	flag.StringVar(&config.ctype, "ctype", "candlestick", "chartType: candlestick|ohlc|line|spline")
	flag.StringVar(&config.force, "force", "", "force one symbol")
	// Customize flag.Usage
	flag.Usage = func() {
		fmt.Printf(`Usage: %s [options]

Options:
  -tf string                  Specify the timeframe for the operation (default '1d')
  -quote string               Build pairs from quote (default 'USDT')
  -limit int                  Limit bars (0=off) (default 300)
  -ctype string               Chart type: candlestick|ohlc|line|spline (default 'candlestick')
  -force string               Force one symbol, e.g., 'BTC/USDT'

URLs:
http://127.0.0.1/             Standard
http://127.0.0.1/test         Test strategy on history
http://127.0.0.1/ptest        Test strategy on history with portfolio
http://127.0.0.1/top/<days>   Show top preformers for nr of days

  `, os.Args[0])
	}
	// Parse flags
	flag.Parse()
	// If --help or -h is provided, we show the custom help
	if len(os.Args) < 2 || (os.Args[1] == "-h" || os.Args[1] == "--help") {
		flag.Usage()
		os.Exit(0) // Exit after displaying the help message
	}
	// ----------------------------------------------------------------------------------------------
	// create list of symbols form data grabbet from my exchanges
	// ----------------------------------------------------------------------------------------------
	symbols = []string{config.force}
	if config.force == "" {
		var err error
		symbols, err = MakeSymbolMultiTimeframe(config.quote, config.tf)
		if err != nil {
			log.Fatal("could not make symbols:", err)
		}
	}

	log.Println("initalizing...")
	// ----------------------------------------------------------------------------------------------
	// Initialize history with database
	// ----------------------------------------------------------------------------------------------
	var err error
	hist, err = history.New()
	if err != nil {
		log.Fatal("could not create history:", err)
	}
	// ----------------------------------------------------------------------------------------------
	// add a downloader to the interface.
	// ----------------------------------------------------------------------------------------------
	hist.Downloader = &Binance{}
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
	hist.Limit(config.limit)
	// ----------------------------------------------------------------------------------------------
	// add strategy to event listener
	// ----------------------------------------------------------------------------------------------
	eventListener.Add(strategy)
	// ----------------------------------------------------------------------------------------------
	// start event listener
	// ----------------------------------------------------------------------------------------------
	eventListener.Start(hist, events)
	// ----------------------------------------------------------------------------------------------
	// highchart settings (highcharts)
	// ----------------------------------------------------------------------------------------------
	// chart.Type = highcharts.ChartType(highcharts.Spline)
	chart.SMA = []int{20, 200}
	// ----------------------------------------------------------------------------------------------
	// http routes for visual results and backtesting
	// ----------------------------------------------------------------------------------------------
	http.HandleFunc("/", httpIndex)
	http.HandleFunc("/test", httpBacktest)
	http.HandleFunc("/top/", httpTopPreformers)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// run strategy on current bar and plot event on chart
func httpIndex(w http.ResponseWriter, r *http.Request) {
	// limit bars
	if config.limit > 0 {
		hist.Limit(config.limit)
	}

	// build charts. if we pass empty events, it will only plot bars data
	var ev history.Events
	c, err := chart.BuildCharts(hist.Map(), ev.Map())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(c)
}

// httpBacktest strategy and plot events
func httpBacktest(w http.ResponseWriter, r *http.Request) {
	// limit bars
	if config.limit > 0 {
		hist.Limit(config.limit)
	}
	// runbacktest on all history that 'hist' handles
	ev, err := hist.Test(strategy, hist.FirstTime(), hist.LastTime())
	if err != nil {
		log.Fatal(err)
	}
	// build charts with events
	c, err := chart.BuildCharts(hist.Map(), ev.Map())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(c)
}

// httPortfolioTest with portfolio
// func httPortfolioTest(w http.ResponseWriter, r *http.Request) {
// 	// limit bars
// 	if config.limit > 0 {
// 		hist.Limit(config.limit)
// 	}
// 	// run strategy backtest on all data
// 	ev, err := hist.PortfolioTest(strategy, hist.FirstTime(), hist.LastTime())
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	// build charts
// 	c, err := chart.BuildCharts(hist.Map(), ev.Map())
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	w.Write(c)
// }

// TopPreformers over 'url:8080/top/x' x = number of bars
func httpTopPreformers(w http.ResponseWriter, r *http.Request) {
	n := config.limit
	// http://127.0.0.1/top/N	where N is number of bars
	if len(r.URL.Path) > 5 {
		if v, err := strconv.Atoi(r.URL.Path[5:]); err == nil {
			n = v
		}
	}
	// new copy of history, so we dont cut any bars from 'hist' struct
	copyHist, err := history.New()
	if err != nil {
		log.Printf("could not create history copy: %v\n", err)
		return
	}
	copyHist.Downloader = &Binance{}
	copyHist.Update(false)
	copyHist.Load(symbols...)
	// limit history if 'http://127.0.0.1/top/N' is used
	if n > 0 && n != config.limit {
		copyHist.Limit(n + 1)
	}
	// run strategy and save events to new bucket
	var results history.Events
	for symbol, bars := range copyHist.Map() {
		strategy := &Preformance{n, false}
		if event, ok := strategy.Run(symbol, bars); ok {
			// add event, in this case its only a percentage value
			results.Add(event)
		}
	}
	// sort by price where the gains value is stored
	results.Sort()
	for _, event := range results {
		fmt.Printf("%-12s %.2f %%\n", event.Symbol, event.Price)
	}
	fmt.Println("-----------  events", len(results))
	/*

		build charts with custom order flow

	*/
	buf, err := chart.MakeHeader()
	if err != nil {
		log.Println(err)
		return
	}
	for _, ev := range results {
		bars := copyHist.GetBars(ev.Symbol)
		title := fmt.Sprintf("%-20s  (%.2f%%)", ev.Symbol, ev.Price)
		chart, err := chart.MakeChart(title, bars, results.Symbol(ev.Symbol))
		if err != nil {
			log.Println(err)
			continue
		}
		// append to slice
		buf = append(buf, chart...)
	}
	w.Write(buf)
}
