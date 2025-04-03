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
	hist         = new(history.History)      // main struct to control bars data
	strategy     = NewEngulfing()            // example strategy
	eventHandler = history.NewEventHandler() // event handler for managing events and strategies
	events       = new(history.Events)       // we store our events here, if we want to save them
	chart        = charts.NewHighChart()

	config  = new(Config) // store argument configurations for example app
	symbols []string      // list of symbols to handle bars
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
	// limit bars
	// ----------------------------------------------------------------------------------------------
	hist.Limit(config.limit)
	// ----------------------------------------------------------------------------------------------
	// load symbols. use a list to load multiple.
	// ----------------------------------------------------------------------------------------------
	hist.Load(symbols...)
	// ----------------------------------------------------------------------------------------------
	// Setup EventHandler and subscribe to events
	// ----------------------------------------------------------------------------------------------
	// Add strategy to event handler
	// eventHandler.AddStrategy(strategy)
	// Subscribe to MARKET_BUY event
	// eventHandler.Subscribe(history.MARKET_BUY, func(event history.Event) error {
	// 	log.Printf("--- Bind your function to MARKET_BUY event\n")
	// 	return nil
	// })
	// eventHandler.Subscribe(history.MARKET_SELL, func(event history.Event) error {
	// 	log.Printf("--- Bind your function to MARKET_SELL event\n")
	// 	return nil
	// })
	// Start event handler that will run strategies
	// and handle events every time we got new bars
	// if err := eventHandler.Start(hist, events); err != nil {
	// 	log.Fatal("could not start event handler:", err)
	// }
	// ----------------------------------------------------------------------------------------------
	// highchart
	// ----------------------------------------------------------------------------------------------
	// chart.Type = charts.ChartType(charts.Spline)
	chart.SMA = []int{20, 200}
	// ----------------------------------------------------------------------------------------------
	// http routes for visual results and backtesting
	// ----------------------------------------------------------------------------------------------
	http.HandleFunc("/", httpPlot)
	http.HandleFunc("/test", httpStrategyTest)
	http.HandleFunc("/favicon.ico", http.NotFound)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// plot all symbol charts loaded in history
func httpPlot(w http.ResponseWriter, r *http.Request) {
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

// httpStrategyTest runs backtest with portfolio tracking and prints performance
func httpStrategyTest(w http.ResponseWriter, r *http.Request) {
	// Reset the strategy to start fresh
	strategy = NewEngulfing()

	// limit bars
	if config.limit > 0 {
		hist.Limit(config.limit)
	}

	tester := history.NewTester(hist, strategy)
	events, err := tester.Test(hist.FirstTime(), hist.LastTime())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// build charts with the events from tester
	c, err := chart.BuildCharts(hist.Map(), events.Map())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(c)
}

// ----------------------------------------------------------------------------------------------
// S T R A T E G I E S
// ----------------------------------------------------------------------------------------------

// Engulfing test strategy
type Engulfing struct {
	history.BaseStrategy
}

func NewEngulfing() *Engulfing {
	return &Engulfing{}
}

// Event EngulfingN..
func (s *Engulfing) OnBar(symbol string, bars history.Bars) (history.Event, bool) {
	if len(bars) < 20 {
		return history.Event{}, false
	}

	SMA := bars[0:20].SMA(history.C)
	ATR := bars[1:4].ATR()

	// MARKET_BUY
	if bars.LastBearIdx() < 5 &&
		bars[0].C()-SMA < 2*ATR &&
		bars[0].C() > bars[bars.LastBearIdx()].H() &&
		bars[0].O() < bars[bars.LastBearIdx()].H() &&
		bars[0].Body() > ATR &&
		bars[0].O()-SMA < 2*ATR &&
		bars[0].C() > SMA {

		// send helper function for BaseStrategy MarketBuy
	}

	// MARKET_SELL
	if bars.LastBullIdx() < 5 &&
		bars[0].O() > bars[bars.LastBullIdx()].L() &&
		bars[0].C() < bars[bars.LastBullIdx()].L() &&
		bars[0].Body() > ATR &&
		bars[0].O()-SMA < 2*ATR &&
		bars[0].C() < SMA {

		// send helper function for BaseStrategy MarketBuy
	}

	return history.Event{}, false
}

// Name returns the strategy name
func (s *Engulfing) Name() string {
	return "Engulfing"
}
