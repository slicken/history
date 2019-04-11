package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/slicken/history/charts"

	"github.com/julienschmidt/httprouter"
	"github.com/slicken/history"
)

var (
	binance = &Binance{}
	data    = new(history.Data)
	chart   = charts.DefaultChart()
	events  history.Events
	strat   []history.Strategy

	conf       *Config
	start, end time.Time
)

// Config ..
type Config struct {
	// strategies
	MA, Engulf, Pin, Dildo bool
	// app settings
	tf                 string
	quote              string
	emax, limit, shift int
	// chart settings
	ctype  string
	volume bool
}

func main() {
	conf = new(Config)
	// MAIN SETTINGS
	flag.StringVar(&conf.tf, "tf", "1d", "timeframe")
	flag.StringVar(&conf.quote, "quote", "BTC", "build symbol from quote currencie")
	flag.IntVar(&conf.shift, "shift", 1, "shift analys")
	// STRATEGIES
	flag.BoolVar(&conf.MA, "ma", true, "SMAEMA signals")
	flag.BoolVar(&conf.Engulf, "engulf", true, "Engulf signals")
	flag.BoolVar(&conf.Pin, "pin", true, "Pinpar signals")
	flag.BoolVar(&conf.Dildo, "dildo", true, "Dildo signals")
	// LIMITS
	flag.IntVar(&conf.emax, "emax", 50, "maximum events to show on site")
	flag.IntVar(&conf.limit, "limit", 210, "maximum bars to limit the history too")
	// CHART SERTTINGS
	flag.StringVar(&conf.ctype, "ctype", "candlestick", "chartType: candlestick|ohlc|line|spline")
	flag.BoolVar(&conf.volume, "vol", true, "volume axis")

	flag.Parse()

	// STRATEGIES
	if conf.MA {
		strat = append(strat, &SMAEMA{shift: conf.shift})
	}
	if conf.Engulf {
		strat = append(strat, &Engulf{shift: conf.shift})
	}
	if conf.Pin {
		strat = append(strat, &Pin{shift: conf.shift})
	}
	if conf.Dildo {
		strat = append(strat, &Dildo{shift: conf.shift})
	}
	if len(strat) == 0 {
		log.Fatal("no strategies enabled.")
	}

	// MAKE SYMBOLS
	symbols, err := MakeSymbolTimeframe(conf.quote, conf.tf)
	if err != nil {
		log.Fatal("could not build symbols from exchange data:", err)
	}
	log.Println("initalizing...")

	data.Downloader = &Binance{}
	data.Update(true)
	data.Load(symbols)

	// APP LOOP -->
	go func() {
		var snames string
		for _, strat := range strat {
			snames += fmt.Sprintf("%T ", strat)[6:]
		}

		log.Printf("Running strategies %s\n", snames)

		for {

			select {
			case s, ok := <-data.C:
				if !ok {
					log.Println("Strategies stopped.")
					return
				}

				symbol, timeframe := history.Split(s)
				if symbol == "" || timeframe == "" {
					continue
				}
				bars := data.Bars(symbol, timeframe)
				if event, ok := bars.Events(strat...); ok && !events.Exist(event) {
					event.Symbol = symbol
					event.Timeframe = timeframe
					events = append(events, event)

					fmt.Printf("%s%s [%s] %s\n", symbol, timeframe, event.Type, event.Name)
					// limit the events to maximum 50 events
					num := len(events)
					if num > conf.emax {
						events = events[(num - conf.emax):]
					}
				}
			default:
				time.Sleep(time.Second)
			}

		}
	}()

	// CHART SETTINGS
	chart.Type = charts.ChartType(conf.ctype)
	chart.Volume = conf.volume
	chart.EMA = []int{21, 55}
	chart.SMA = []int{200}

	r := httprouter.New()
	r.GET("/", httpEvents)
	log.Fatal(http.ListenAndServe(":8080", r))

}

func httpEvents(w http.ResponseWriter, r *http.Request, q httprouter.Params) {
	c, err := chart.BuildCharts(data, events)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.Write(c)
}

// wait key
func wait(v string) {
	fmt.Printf("> Press 'Enter' to: %s\n", v)
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

// SMAEMA ..
type SMAEMA struct {
	shift int
}

// Event ..
func (s *SMAEMA) Event(bars history.Bars) (history.Event, bool) {
	var event history.Event

	if s.shift+201 > len(bars) {
		return event, false
	}

	ema21 := bars[s.shift : s.shift+21].EMA(history.C)
	ema55 := bars[s.shift : s.shift+55].EMA(history.C)
	// ema89 := bars[s.shift : s.shift+89].EMA(history.C)
	// sma100 := bars[s.shift : s.shift+100].SMA(history.C)
	sma200 := bars[s.shift : s.shift+200].SMA(history.C)

	w21 := bars[s.shift].Open > ema21 && bars[s.shift].Low < ema21 // && bars[s.shift].Close > ema21
	w55 := bars[s.shift].Open > ema55 && bars[s.shift].Low < ema55 // && bars[s.shift].Close > ema55
	// w89 := bars[s.shift].Open > ema89 && bars[s.shift].Low < ema89 // && bars[s.shift].Close > ema89
	// w100 := bars[s.shift].Open > sma100 && bars[s.shift].Low < sma100 // && bars[s.shift].Close > sma100
	w200 := bars[s.shift].Open > sma200 && bars[s.shift].Low < sma200 // && bars[s.shift].Close > sma200

	var wicks []int
	if w21 {
		wicks = append(wicks, 21)
	}
	if w55 {
		wicks = append(wicks, 55)
	}
	// if w89 {
	// 	wicks = append(wicks, 89)
	// }
	// if w100 {
	// 	wicks = append(wicks, 100)
	// }
	if w200 {
		wicks = append(wicks, 200)
	}

	if len(wicks) > 1 {
		event.Buy(bars[s.shift].Time, bars[s.shift].Price(history.O))
		event.Name = fmt.Sprintf("%v", wicks)
		event.Name = "SMAEMA"
		return event, true
	}

	return event, false
}

// Engulf ..
type Engulf struct {
	shift int
}

// Event FractalWick..
func (s *Engulf) Event(bars history.Bars) (history.Event, bool) {
	var event history.Event

	if s.shift+3 > len(bars) {
		return event, false
	}

	if bars.IsEngulfBuy() {
		event.Buy(bars[s.shift].Time, bars[s.shift].Price(history.O))
		event.Name = "Engulf"
		return event, true
	}

	return event, false
}

// Pin ..
type Pin struct {
	shift int
}

// Event FractalWick..
func (s *Pin) Event(bars history.Bars) (history.Event, bool) {
	var event history.Event

	if s.shift+3 > len(bars) {
		return event, false
	}

	if bars[s.shift : s.shift+5].IsPinBuy() {
		event.Buy(bars[s.shift].Time, bars[s.shift].Price(history.C))
		event.Name = "Pinbar"
		return event, true
	}

	return event, false
}

// Dildo ..
type Dildo struct {
	shift int
}

// Event ..
func (s *Dildo) Event(bars history.Bars) (history.Event, bool) {
	var event history.Event

	if s.shift+22 > len(bars) {
		return event, false
	}

	atr := bars[s.shift+2 : s.shift+22].ATR(history.HL)

	if bars[s.shift].Range() > atr*2.3 {
		event.Buy(bars[s.shift].Time, bars[s.shift].Price(history.O))
		event.Name = "Dildo"
		return event, true
	}

	return event, false
}
