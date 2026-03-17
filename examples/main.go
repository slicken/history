package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/slicken/history"
	"github.com/slicken/history/charts"
)

var (
	strategy     = NewPercScalper()    // percentage scalper strategy
	hist         *history.History      // main struct to control bars data
	eventHandler *history.EventHandler // event handler for managing events and strategies
	events       *history.Events       // we store our events here, if we want to save them
	config       *Config               // store argument configurations for example app
	symbols      []string              // list of symbols to handle bars
	chart        charts.ChartBuilder
)

// Config holds app arguments
type Config struct {
	// symbol settingss
	tf     string
	quote  string
	symbol string
	// history settings
	update bool
	limit  int
	// chart settings
	chart string
	ctype string
	// temp
	saveAI_data bool
}

func main() {
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
	flag.BoolVar(&config.update, "update", false, "update bars")
	flag.IntVar(&config.limit, "limit", 0, "limit bars (0=off)")
	flag.StringVar(&config.chart, "chart", "highcharts", "chart library: highcharts|tradingview")
	flag.StringVar(&config.ctype, "ctype", "candlestick", "chartType: candlestick|ohlc|line|spline")
	flag.StringVar(&config.symbol, "symbol", "", "singlle symboltf")
	flag.BoolVar(&config.saveAI_data, "saveAI_data", false, "save dataset for predictor")
	// Customize flag.Usage
	flag.Usage = func() {
		fmt.Printf(`Usage: %s [options]

Options:
  -tf string                  Specify the timeframe for the operation (default '1d')
  -quote string               Build pairs from quote (default 'USDT')
  -update bool                Update bars (default false)
  -limit int                  Limit bars (0=off) (default 300)
  -chart string               Chart library: highcharts|tradingview (default 'highcharts')
  -ctype string               Chart type: candlestick|ohlc|line|spline (default 'candlestick')
  -symbol string              Single symboltf. e.g., 'BTCUSDT1d'
  -saveAI_data bool           Save dataset fir LSTM predictor strategy (default false)

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
	// Initialize history with database
	// ----------------------------------------------------------------------------------------------
	var err error
	hist, err = history.New()
	if err != nil {
		log.Fatal("could not create history:", err)
	}
	// ----------------------------------------------------------------------------------------------
	// create list of symbols form data grabbet from my exchanges
	// ----------------------------------------------------------------------------------------------
	symbols = []string{config.symbol}
	if config.symbol == "" {
		symbols, err = MakeSymbolMultiTimeframe(config.quote, config.tf)
		if err != nil {
			symbols, err = hist.StoredSymbols(config.tf)
			if err != nil {
				log.Printf("Error getting stored symbols: %v", err)
			}
		}
	}
	log.Printf("initalizing %d symbols...\n", len(symbols))
	// ----------------------------------------------------------------------------------------------
	// add a downloader to the interface.
	// ----------------------------------------------------------------------------------------------
	hist.Downloader = &Binance{}
	// ----------------------------------------------------------------------------------------------
	// update bars when there is new bars avaliable
	// ----------------------------------------------------------------------------------------------
	hist.Update(config.update)
	// ----------------------------------------------------------------------------------------------
	// load symbols. use a list to load multiple.
	// ----------------------------------------------------------------------------------------------
	hist.Load(symbols...)
	// ----------------------------------------------------------------------------------------------
	// limit bars
	// ----------------------------------------------------------------------------------------------
	if config.limit > 0 {
		hist.Limit(config.limit)
	}

	// Force re-download bars
	// hist.Reprocess(5000)

	// ----------------------------------------------------------------------------------------------
	// Setup EventHandler and subscribe any functions to Events
	// ----------------------------------------------------------------------------------------------
	eventHandler.AddStrategy(strategy)
	eventHandler.Subscribe(history.MARKET_BUY, func(event history.Event) error {
		log.Printf("--- Bind your function to MARKET_BUY event\n")
		return nil
	})
	eventHandler.Subscribe(history.MARKET_SELL, func(event history.Event) error {
		log.Printf("--- Bind your function to MARKET_SELL event\n")
		return nil
	})
	if err := eventHandler.Start(hist, events); err != nil {
		log.Fatal("could not start event handler:", err)
	}
	// ----------------------------------------------------------------------------------------------
	// chart (highcharts or tradingview)
	// ----------------------------------------------------------------------------------------------
	switch config.chart {
	case "tradingview":
		tv := charts.NewTradingView()
		tv.SMA = []int{20, 200}
		chart = tv
	default:
		hc := charts.NewHighChart()
		hc.SMA = []int{20, 200}
		chart = hc
	}
	// ----------------------------------------------------------------------------------------------
	// http routes for visual results and backtesting
	// ----------------------------------------------------------------------------------------------

	log.Println("starting web server...")
	http.HandleFunc("/", httpPlot)
	http.HandleFunc("/test", httpStrategyTest)
	http.HandleFunc("/favicon.ico", http.NotFound)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// plot all symbol charts loaded in history
func httpPlot(w http.ResponseWriter, r *http.Request) {
	// build charts. if we pass empty events, it will only plot bars data
	var ev history.Events
	c, err := chart.BuildCharts(hist.Map(), ev.Map())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(c)
}

// httpStrategyTest runs backtest with portfolio tracking and prints performance
func httpStrategyTest(w http.ResponseWriter, r *http.Request) {
	// Reset the strategy to start fresh
	tester := history.NewTester(hist, strategy)
	results, err := tester.Test(hist.FirstTime(), hist.LastTime())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// build charts with the events from tester
	c, err := chart.BuildCharts(hist.Map(), results.Events.Map())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(c)
}
