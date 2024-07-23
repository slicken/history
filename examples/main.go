package main

import (
	"flag"
	"fmt"
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
	strategy      = &test{}                    // engulfing strategy (create you owrn strategies)
	chart         = highcharts.DefaultChart()  // we use highcharts for plotting

	config = new(Config) // store argument configurations
	// other
	symbols []string // list of symbols to handle bars
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
	flag.StringVar(&config.tf, "tf", "1d", "timeframe")
	flag.StringVar(&config.quote, "quote", "USDT", "build pairs from quote")
	flag.BoolVar(&config.file, "file", false, "save '/top/' preformers to file")
	flag.IntVar(&config.limit, "limit", 300, "limit bars (0=off)")
	flag.StringVar(&config.ctype, "ctype", "candlestick", "chartType: candlestick|ohlc|line|spline")
	flag.StringVar(&config.force, "force", "", "force one symbol")
	flag.Parse()
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

	// var portfolio history.Portfolio

	// var ev1 history.Event
	// ev1.Name = "Test Buy"
	// ev1.Symbol = "BTCUSDT1d"
	// ev1.Price = 30122.2
	// ev1.Type = history.MARKET_SELL
	// ev1.Time = time.Now().Add(-3 * 24 * time.Hour)

	// fmt.Println("initial", 1000)
	// size := 1000 / ev1.Price
	// fmt.Println("size", size)
	// pos1 := history.MakePosition(ev1, size)

	// // now
	// price := 40002.1
	// _, _ = portfolio.Add(pos1)

	// // get position
	// n, lpos := portfolio.Open.GetLast("BTCUSDT1d")
	// // close position index
	// ok := portfolio.Close(n, price, time.Now())
	// if !ok {
	// 	fmt.Println("ERROR Close")
	// }

	// _ = lpos
	// _ = n
	// _ = portfolio

	// os.Exit(0)

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
	http.HandleFunc("/test", httPortfolioTest)
	http.HandleFunc("/backtest", httpBacktest)
	http.HandleFunc("/top/", httpTopPreformers) // top preformers for x days
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

// backtest strategy and plot events
func httPortfolioTest(w http.ResponseWriter, r *http.Request) {
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

// Backtest with portfolio
func httpBacktest(w http.ResponseWriter, r *http.Request) {
	// limit bars
	if config.limit > 0 {
		hist.Limit(config.limit)
	}
	// run strategy backtest on all data
	ev, err := hist.PortfolioTest(strategy, hist.FirstTime(), hist.LastTime())
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
	n := config.limit
	// http://127.0.0.1/top/N	where N is number of bars
	if len(r.URL.Path) > 5 {
		if v, err := strconv.Atoi(r.URL.Path[5:]); err == nil {
			n = v
		}
	}
	// new copy of history, so we dont cut any bars from 'hist' struct
	var copyHist = new(history.History)
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
		bars := copyHist.Bars(ev.Symbol)
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

/*

	------ STRATEGIES ------

*/

type Preformance struct {
	Limit      int
	LowestOpen bool
}

// Event Signals ...
func (s *Preformance) Run(symbol string, bars history.Bars) (history.Event, bool) {
	var event = history.NewEvent(symbol)

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

// Power strategy
func (s *Power) Run(symbol string, bars history.Bars) (history.Event, bool) {
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

// Engulfing location
type Engulfing struct{}

// Event Engulfing ..
func (s *Engulfing) Run(symbol string, bars history.Bars) (history.Event, bool) {
	var event = history.NewEvent(symbol)

	if 21 > len(bars) {
		return event, false
	}
	// EXCLUDE SYMBOLS PRICES MATCHING PREFEX "0.000000xx"
	if price := strconv.FormatFloat(bars[0].O(), 'f', -1, 64); len(price) >= 7 {
		if price[:7] == "0.00000" {
			return event, false
		}
	}

	// --------------
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

		event.Type = history.MARKET_BUY
		event.Name = "Engulfing"
		event.Time = bars[0].T()
		event.Price = bars[0].C()
		return event, true
	}

	// MARKET_SELL
	if bars.LastBullIdx() < 5 &&
		bars[0].O() > bars[bars.LastBullIdx()].L() &&
		bars[0].C() < bars[bars.LastBullIdx()].L() &&
		bars[0].Body() > ATR &&
		bars[0].O()-SMA < 2*ATR &&
		bars[0].C() < SMA {

		event.Type = history.CLOSE_BUY
		event.Name = "Engulfing"
		event.Time = bars[0].T()
		event.Price = bars[0].C()
		return event, true
	}

	return event, false
}

// test strategy
type test struct{}

// Event Signals
func (s *test) Run(symbol string, bars history.Bars) (history.Event, bool) {
	var event = history.NewEvent(symbol)

	if 260 > len(bars) {
		return event, false
	}
	// EXCLUDE SYMBOLS PRICES MATCHING PREFEX "0.000000xx"
	if price := strconv.FormatFloat(bars[0].O(), 'f', -1, 64); len(price) >= 7 {
		if price[:7] == "0.00000" {
			return event, false
		}
	}

	// --------------
	lookback := 50
	// 0=20, 1=200
	var ma = make(map[int][]float64, 3)
	for i := 0; i < lookback; i++ {
		for n, v := range []float64{bars[i : i+20].SMA(3), bars[i : i+200].SMA(3)} {
			ma[n] = append(ma[n], v)
		}
	}

	recentCrossUp := false
	for i := 0; i < lookback-1; i++ {
		if ma[0][i] > ma[1][i] && (ma[0][i+1] < ma[1][i+1]) {
			recentCrossUp = true
		}
	}

	if recentCrossUp &&
		bars[0].Bullish() &&
		// bars[0].Range() > bars[0:10].ATR() &&
		bars[0].O()-bars[1:7].Lowest(history.O) < bars[1:21].ATR() &&
		bars[0].L() <= ma[0][0] && bars[0].C() > ma[0][0] &&
		bars[0].C() > bars[1:4].Highest(history.H) &&

		//ma[0][1] > ma[1][1] &&
		// (ma[0][0] > ma[0][1]) || (ma[1][0] > ma[1][1]) &&

		history.WithinRange(ma[0][0], ma[1][0], 2*bars[1:21].ATR()) {
		// history.WithinRange(ma[0][0], bars[0].O(), bars[1:21].ATR()) {

		event.Type = history.MARKET_BUY
		event.Name = "TEST"
		event.Time = bars[0].T()
		event.Price = bars[0].C()
		return event, true
	}

	return event, false
}
