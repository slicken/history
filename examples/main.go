package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
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
	evl    = new(history.EventListener) // we add our strategy to eventlistener witch is looking at the hist
	events = new(history.Events)        // we store our events here, if we want to save them

	// strat = &slk{iLen: 30, lenBounces: 20} // strategy for this example
	// strat = &Power{}              // strategy for this example
	strat = &Engulf{}             // strategy for this example
	chart = charts.DefaultChart() // we use highcharts for plotting

	conf = new(Config) // config from args
	// other
	symbols       []string
	topPreformers []string
)

// Config holds app arguments
type Config struct {
	// symbols
	tf    string
	quote string
	// top preformers
	file bool // output top preformers to file (for tradingview imports)
	// chart settings
	limit int
	ctype string
	// force is for loading one symbol only
	force string
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
	chart.Type = charts.ChartType(conf.ctype)
	chart.SMA = []int{8, 20, 200}
	// ----------------------------------------------------------------------------------------------
	//create slice of charts we will be loading
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
	// add the downloader interface
	// ----------------------------------------------------------------------------------------------
	hist.Downloader = &Binance{}
	// hist.SetDataDir("./exchangedata")
	// ----------------------------------------------------------------------------------------------
	// keep looking for new bars and keep updating the hist struct
	// ----------------------------------------------------------------------------------------------
	hist.Update(true)
	// ----------------------------------------------------------------------------------------------
	// load sybols
	// ----------------------------------------------------------------------------------------------
	hist.Load(symbols...)
	// ----------------------------------------------------------------------------------------------
	// add strategy to event listener
	// ----------------------------------------------------------------------------------------------
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
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// run strategy on current bar and plot event on chart
func httpIndex(w http.ResponseWriter, r *http.Request) {
	// limit bars
	if conf.limit > 0 {
		hist.Limit(conf.limit)
	}

	// var tev = new(history.Events)
	// if len(topPreformers) > 0 {
	// 	for _, e := range *events {
	// 		for _, top := range topPreformers {
	// 			if top != e.Pair {
	// 				continue
	// 			}
	// 			*tev = append(*tev, e)
	// 		}
	// 	}
	// } else {
	// 	*tev = *events
	// }

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
	c, err := chart.BuildCharts(hist.Bars, ev...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(c)
}

// TopPreformers over url:8080/top/x = bars
func httpTopPreformers(w http.ResponseWriter, r *http.Request) {
	bars := 30
	if len(r.URL.Path) > 5 {
		if v, err := strconv.Atoi(r.URL.Path[5:]); err == nil {
			bars = v
		}
	}

	// limit bars
	if conf.limit > 0 {
		hist.Limit(conf.limit)
	}

	var ev = history.Events{}
	strat := &Gained{Bars: bars}

	for symbol, bars := range hist.Bars {
		if new, ok := strat.Event(bars); ok && !events.Exists(new) {
			new.Pair, new.Timeframe = history.Split(symbol)
			ev = append(ev, new)
		}
	}

	ev.Sort()

	topPreformers = nil
	for _, e := range ev {
		topPreformers = append(topPreformers, e.Pair)
		log.Printf("%-12s %.2f %%\n", e.Pair, e.Price)
	}

	// save to file (compatible with tradingview imports)
	if conf.file {
		var buf = bytes.NewBuffer(nil)
		var tmp []string
		for _, v := range topPreformers {
			tmp = append(tmp, v+",") // v[:len(v)-2]
		}

		l := int(math.Min(30, float64(len(tmp))))
		buf.WriteString(fmt.Sprintf("%v", tmp[:l]))
		ioutil.WriteFile(conf.quote+".txt", buf.Bytes(), 0644)
	}

	// build charts
	c, err := chart.BuildCharts(hist.Bars, ev...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(c)
}

// ----- STRATEGIES -----------

type Gained struct {
	Bars       int
	LowestOpen bool
}

// Event Signals ..
func (s *Gained) Event(bars history.Bars) (history.Event, bool) {
	var event history.Event

	if s.Bars+1 > len(bars) {
		return event, false
	}

	var perc float64
	if s.LowestOpen {
		perc = 100 * ((bars[0].C() - bars[0:s.Bars].Lowest(history.O)) / bars[0:s.Bars].Lowest(history.O))
	} else {
		perc = 100 * ((bars[0].C() - bars[s.Bars].O()) / bars[s.Bars].O())
	}
	event.Add(history.OTHER, fmt.Sprintf("%.1f", perc), bars.LastBar().T(), perc)

	return event, true
}

// Power location
type Power struct {
	shift int
}

// Event Signals ..
func (s *Power) Event(bars history.Bars) (history.Event, bool) {
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

	// 0=10, 1=20, 2=50
	var ma = make(map[int][]float64, 3)
	for i := 0; i < 30; i++ {
		for n, v := range []float64{bars[i : i+10].SMA(3), bars[i : i+20].SMA(3), bars[i : i+50].SMA(3)} {
			ma[n] = append(ma[n], v)
		}
	}

	var slope = make(map[int][]float64, 3)
	for i := 0; i < 30; i++ {
		for n, v := range []float64{(bars[i:i+10].SMA(3) / bars[i+1:i+11].SMA(3)) - 1, (bars[i:i+20].SMA(3) / bars[i+1:i+21].SMA(3)) - 1, (bars[i:i+50].SMA(3) / bars[i+1:i+51].SMA(3)) - 1} {
			slope[n] = append(slope[n], v)
		}
	}

	// temporary shifted
	if bars[s.shift].Bullish() &&

		bars[s.shift].Range() > bars[s.shift:10+s.shift].ATR() &&
		bars[s.shift].L() <= ma[1][s.shift] && bars[s.shift].C() > ma[1][s.shift] &&
		bars[s.shift].C() > bars[s.shift+1:s.shift+3].Highest(history.H) &&

		bars[s.shift].O()-bars[s.shift+1:s.shift+7].Lowest(history.O) < bars[1:20].ATR() &&

		ma[1][s.shift+1] > ma[2][s.shift+1] &&
		history.WithinRange(ma[1][s.shift], bars[s.shift].C(), bars[1:20].ATR()) {

		event.Add(0, "POWER BUY", bars[0].T(), bars[0].O())

		return event, true
	}

	// closing sell or close signal.
	// placeholder for positions etc. must be made in strategy struct

	sh := s.shift + 3
	if bars[1+sh].Bullish() &&

		bars[1+sh].Range() > bars[1+sh:11+sh].ATR() &&
		bars[1+sh].L() <= ma[1][1+sh] && bars[1+sh].C() > ma[1][1+sh] &&
		bars[1+sh].C() > bars[2+sh:4+sh].Highest(history.H) &&

		bars[1+sh].O()-bars[2+sh:8+sh].Lowest(history.O) < bars[1:20].ATR() &&

		ma[1][2+sh] > ma[2][2+sh] &&
		history.WithinRange(ma[1][1+sh], bars[1+sh].C(), bars[1:20].ATR()) {

		event.Add(4, "SELL", bars[0].T(), bars[0].O())

		return event, true
	}

	return event, false
}

// slk location
type slk struct{}

// Event Signals .. sma/ema uses chart settings MA's
func (s *slk) Event(bars history.Bars) (history.Event, bool) {
	var event history.Event

	// filter symbols with less bars and price prefix is equals "0.00000"
	if 210 > len(bars) {
		return event, false
	}
	if price := strconv.FormatFloat(bars[0].O(), 'f', -1, 64); len(price) >= 7 {
		if "0.00000" == price[:7] {
			return event, false
		}
	}

	/*
		MovingAverages: this code uses ma's you have choosen for chart settings (chart.SMA & charts.EMA)
		- history
		- slopes (> 0 = sloping up, < 0 = sloing down
		- bounces
	*/

	var SMA = make(map[int][]float64, len(chart.SMA))
	var SMASlope = make(map[int][]float64, len(chart.SMA))
	var SMABounce = make([]int, len(chart.SMA))
	if len(chart.SMA) > 0 {
		for i := 0; i < 10; i++ {
			for n := range chart.SMA {
				v := bars[i : i+chart.SMA[n]].SMA(history.C)
				SMA[n] = append(SMA[n], v)
			}
			for n := range chart.SMA {
				v := bars[i:i+chart.SMA[n]].SMA(history.C)/bars[i+1:i+chart.SMA[n]+1].SMA(history.C) - 1
				SMASlope[n] = append(SMASlope[n], v)
			}
			if i > 0 && i < 10 {
				i--
				for n := range chart.SMA {
					if bars[i].O() >= SMA[n][i] && bars[i].L() <= SMA[n][i] && bars[i].C() >= SMA[n][i] ||
						bars[i+1].O() >= SMA[n][i+1] && bars[i+1].C() <= SMA[n][i+1] && bars[i].O() <= SMA[n][i] && bars[i].C() >= SMA[n][i] {
						SMABounce[n]++
					}
				}
				i++
			}
		}
	}
	var EMA = make(map[int][]float64, len(chart.EMA))
	var EMASlope = make(map[int][]float64, len(chart.EMA))
	var EMABounce = make([]int, len(chart.SMA))
	if len(chart.EMA) > 0 {
		for i := 0; i < 10; i++ {
			for n := range chart.EMA {
				v := bars[i : i+chart.EMA[n]].EMA(history.C)
				EMA[n] = append(EMA[n], v)
			}
			for n := range chart.EMA {
				v := bars[i:i+chart.EMA[n]].EMA(history.C)/bars[i+1:i+chart.EMA[n]+1].EMA(history.C) - 1
				EMASlope[n] = append(EMASlope[n], v)
			}
			if i > 0 && i < 10 {
				i--
				for n := range chart.EMA {
					if bars[i].O() >= EMA[n][i] && bars[i].L() <= EMA[n][i] && bars[i].C() >= EMA[n][i] ||
						bars[i+1].O() >= EMA[n][i+1] && bars[i+1].C() <= EMA[n][i+1] && bars[i].O() <= EMA[n][i] && bars[i].C() >= EMA[n][i] {
						EMABounce[n]++
					}
				}
				i++
			}
		}
	}

	if SMASlope[1][1] > 0 && SMASlope[1][5] > 0 && SMASlope[1][10] > 0 &&
		bars[2].Bullish() && bars[2].Body() > bars[3:13].ATR() && bars[2].Range() < 4*bars[3:13].ATR() &&
		bars[1].Bear() && bars[1].BodyLow() > (bars[2].BodyLow()+bars[2].BodyHigh())*0.3 &&

		// bars[2].L() <= SMA[1][2] && bars[2].C() > SMA[1][2] &&
		// bars[2].C() > bars[2+1:2+3].Highest(history.H) &&

		// bars[s.shift].O()-bars[s.shift+1:s.shift+7].Lowest(history.O) < atr &&

		// bars[2].Bull() && Bullish() && bars[1].Bear() &&
		//		bars[2].Body() > 2*bars[1].Body() &&
		//		bars[1].BodyLow() >= (bars[2].H()+bars[2].L()/2) &&
		bars[0].O() == bars[0].O() {
		//		bars[1].L() <= SMA[1][1] && bars[1].C() > SMA[1][1] {
		event.Add(0, "RBI", bars[0].T(), bars[0].C())
		return event, true
	}

	// atr := bars[1:32].ATR()

	// // SMA 0
	// if len(chart.SMA) > 0 && SMASlope[0][0] > 0 &&
	// 	bars[0].O() > SMA[0][0] && bars[0].L() < SMA[0][0] && bars[0].C() > SMA[0][0] {
	// 	event.Add(0, fmt.Sprintf("SMA%d %.8f", chart.SMA[0], SMA[0][0]), bars[0].T(), bars[0].C())
	// 	return event, true
	// }
	// SMA 1
	// if len(chart.SMA) > 1 && SMASlope[1][0] > 0 &&
	// 	bars[0].O() > SMA[1][0] && bars[0].L() < SMA[1][0] && bars[0].C() > SMA[1][0] {
	// 	event.Add(0, fmt.Sprintf("SMA%d %.8f", chart.SMA[1], SMA[1][0]), bars[0].T(), bars[0].C())
	// 	return event, true
	// }
	// // SMA 2
	// if len(chart.SMA) > 2 && SMASlope[2][0] > 0 &&
	// 	bars[0].O() > SMA[2][0] && bars[0].L() < SMA[2][0] && bars[0].C() > SMA[2][0] {
	// 	event.Add(0, fmt.Sprintf("SMA%d %.8f", chart.SMA[2], SMA[2][0]), bars[0].T(), bars[0].C())
	// 	return event, true
	// }

	// // EMA 0
	// if len(chart.EMA) > 0 && EMASlope[0][0] > 0 &&
	// 	bars[0].O() > EMA[0][0] && bars[0].L() < EMA[0][0] && bars[0].C() > EMA[0][0] {
	// 	event.Add(0, fmt.Sprintf("EMA%d %.8f", chart.EMA[0], EMA[0][0]), bars[0].T(), bars[0].C())
	// 	return event, true
	// }
	// // EMA 1
	// if len(chart.EMA) > 1 && EMASlope[1][0] > 0 &&
	// 	bars[0].O() > EMA[1][0] && bars[0].L() < EMA[1][0] && bars[0].C() > EMA[1][0] {
	// 	event.Add(0, fmt.Sprintf("EMA%d %.8f", chart.EMA[1], EMA[1][0]), bars[0].T(), bars[0].C())
	// 	return event, true
	// }
	// // EMA 2
	// if len(chart.EMA) > 2 && EMASlope[2][0] > 0 &&
	// 	bars[0].O() > EMA[2][0] && bars[0].L() < EMA[2][0] && bars[0].C() > EMA[2][0] {
	// 	event.Add(0, fmt.Sprintf("EMA%d %.8f", chart.EMA[2], EMA[2][0]), bars[0].T(), bars[0].C())
	// 	return event, true
	// }

	return event, false

}

// Engulf location
type Engulf struct {
	shift int
}

// Event Engulf ..
func (s *Engulf) Event(bars history.Bars) (history.Event, bool) {
	var event history.Event

	if 210 > len(bars) {
		return event, false
	}

	// EXCLUDE SYMBOLS PRICES MATCHING PREFEX "0.000000xx"
	if p := strconv.FormatFloat(bars[0].O(), 'f', -1, 64); len(p) >= 7 {
		if "0.00000" == p[:7] {
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

	// temporary shifted
	if bars.IsEngulfBuy() &&

		// bars[s.shift].C() > ma[1][s.shift] &&
		// bars[s.shift].C() > ma[2][s.shift] {

		(bars[s.shift].L() < ma[0][s.shift] && bars[s.shift].C() > ma[0][s.shift] ||
			bars[s.shift].L() < ma[1][s.shift] && bars[s.shift].C() > ma[1][s.shift] ||
			bars[s.shift].L() < ma[2][s.shift] && bars[s.shift].C() > ma[2][s.shift]) {

		event.Add(0, "ENGULF", bars[0].T(), bars[0].O())

		return event, true
	}

	return event, false
}
