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

	"github.com/slicken/history"
	"github.com/slicken/history/charts"
)

var (
	hist         = new(history.History)      // main struct to control bars data
	events       = new(history.Events)       // we store our events here, if we want to save them
	eventHandler = history.NewEventHandler() // event handler for managing events and strategies
	strategy     = NewEngulfing()            // lstn predicto strategy
	predictor    = NewPredictor(60)
	chart        = charts.NewHighChart()

	config  = new(Config) // store argument configurations for example app
	symbols []string      // list of symbols to handle bars
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
	flag.IntVar(&config.limit, "limit", 300, "limit bars (0=off)")
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
	// create list of symbols form data grabbet from my exchanges
	// ----------------------------------------------------------------------------------------------
	var err error
	symbols = []string{config.symbol}
	if config.symbol == "" {
		symbols, err = MakeSymbolMultiTimeframe(config.quote, config.tf)
		if err != nil {
			log.Fatal("could not make symbols:", err)
		}
	}
	log.Printf("initalizing %d symbols...\n", len(symbols))
	// ----------------------------------------------------------------------------------------------
	// Initialize history with database
	// ----------------------------------------------------------------------------------------------
	hist, err = history.New()
	if err != nil {
		log.Fatal("could not create history:", err)
	}
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
	hist.Limit(config.limit)
	// ----------------------------------------------------------------------------------------------
	// Setup EventHandler and subscribe to events
	// ----------------------------------------------------------------------------------------------
	// eventHandler.AddStrategy(strategy)
	// eventHandler.Subscribe(history.MARKET_BUY, func(event history.Event) error {
	// 	log.Printf("--- Bind your function to MARKET_BUY event\n")
	// 	return nil
	// })
	// eventHandler.Subscribe(history.MARKET_SELL, func(event history.Event) error {
	// 	log.Printf("--- Bind your function to MARKET_SELL event\n")
	// 	return nil
	// })
	// if err := eventHandler.Start(hist, events); err != nil {
	// 	log.Fatal("could not start event handler:", err)
	// }
	// ----------------------------------------------------------------------------------------------
	// highchart
	// ----------------------------------------------------------------------------------------------
	// chart.Type = charts.ChartType(charts.Spline)
	// chart.SMA = []int{20, 200}
	// ----------------------------------------------------------------------------------------------
	// http routes for visual results and backtesting
	// ----------------------------------------------------------------------------------------------

	if config.saveAI_data {
		time.Sleep(10 * time.Second)

		log.Println("saving AI data...")
		for _, symbol := range symbols {
			saveAI_Data(hist, symbol)
		}
	}

	log.Println("starting web server...")
	http.HandleFunc("/", httpPlot)
	http.HandleFunc("/test", httpStrategyTest)
	http.HandleFunc("/predictor", httpPredictor)
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
	strategy := NewEngulfing()

	// limit bars
	if config.limit > 0 {
		hist.Limit(config.limit)
	}

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

// httpPredictor plots predicted price on chart
func httpPredictor(w http.ResponseWriter, r *http.Request) {
	// Reset the strategy to start fresh

	// limit bars
	if config.limit > 0 {
		hist.Limit(config.limit)
	}

	tester := history.NewTester(hist, predictor)
	results, err := tester.Test(hist.FirstTime(), hist.LastTime())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// print results
	fmt.Println("Predicted events:", predictor.num)
	fmt.Printf("Wins: %d\n", predictor.win)
	fmt.Printf("Loss: %d\n", predictor.loss)
	winRatio := float64(predictor.win) / float64(predictor.num) * 100
	lossRatio := float64(predictor.loss) / float64(predictor.num) * 100
	fmt.Printf("Win ratio: %.2f%%\nLoss ratio: %.2f%%\n", winRatio, lossRatio)

	// build charts with the events from tester
	c, err := chart.BuildCharts(hist.Map(), results.Events.Map())
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
	return &Engulfing{
		BaseStrategy: *history.NewBaseStrategy("Engulfing"),
	}
}

// Event EngulfingN..
func (s *Engulfing) OnBar(symbol string, bars history.Bars) (history.Event, bool) {
	if len(bars) < 20 {
		return history.Event{}, false
	}

	// close position on next bar
	if _, exists := s.GetPortfolioManager().Positions[symbol]; exists {
		return s.Close(), true
	}

	SMA := bars[0:20].SMA(history.C)
	ATR := bars[1:4].ATR()

	// MARKET_BUY signal
	if bars.LastBearIdx() < 5 &&
		bars[0].C()-SMA < 2*ATR &&
		bars[0].C() > bars[bars.LastBearIdx()].H() &&
		bars[0].O() < bars[bars.LastBearIdx()].H() &&
		bars[0].Body() > ATR &&
		bars[0].O()-SMA < 2*ATR &&
		bars[0].C() > SMA {

		return s.BuyEvent(2000, bars[0].C()), true
	}

	// MARKET_SELL signal
	if bars.LastBullIdx() < 5 &&
		bars[0].O() > bars[bars.LastBullIdx()].L() &&
		bars[0].C() < bars[bars.LastBullIdx()].L() &&
		bars[0].Body() > ATR &&
		bars[0].O()-SMA < 2*ATR &&
		bars[0].C() < SMA {

		return s.SellEvent(2000, bars[0].C()), true
	}

	return history.Event{}, false
}

// Name returns the strategy name
func (s *Engulfing) Name() string {
	return "Engulfing"
}
