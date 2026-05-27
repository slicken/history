# Go Trading History Library

A powerful Go library for managing historical market data, implementing trading strategies, and backtesting. Features include data management, strategy development, event handling, and visualization tools.

## Overview

- SQLite database for efficient data storage
- Event-driven architecture for strategy development
- Built-in technical indicators
- Real-time data streaming capabilities
- Visualization with Highcharts and TradingView
- Portfolio management (in development)

## Quick Start

### Managing Historical Data

```go
package main

import (
    "time"
    "github.com/slicken/history"
)

func main() {
    // Initialize history with SQLite database
    hist, err := history.New()
    if err != nil {
        panic(err)
    }

    // Add a data downloader (implement the Downloader interface)
    hist.Downloader = &MyDataDownloader{}

    // Load symbols
    hist.Load("BTCUSDT1h", "ETHUSDT1h")

    // Get bars for a symbol
    bars := hist.GetBars("BTCUSDT1h")

    // Enable auto-updates
    hist.Update(true)

	// Add strategy to event handler
	eventHandler.AddStrategy(MyStrategy)

    // Subscribe your function to MARKET_BUY event outputed by your strategy
	eventHandler.Subscribe(history.MARKET_BUY, func(event history.Event) error {
		log.Printf("--- Bind your function to MARKET_BUY event\n")
		return nil
	})

	// Start event handler that will run the strategies everyime we have new bars data
	if err := eventHandler.Start(hist, events); err != nil {
		log.Fatal("could not start event handler:", err)
    }

    // backtest strategies
    testerEvents := history.NewTester(hist, strategy)
	events, err := testerEvents.Test(hist.FirstTime(), hist.LastTime())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// build charts with and display event markers
	c, err := chart.BuildCharts(hist.Map(), events.Map())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

    // and much more...
```

### Implementing a Strategy

```go
// SMA Crossover strategy example
type SMACrossover struct {
    history.BaseStrategy
}

func NewSMACrossover() *SMACrossover {
    return &SMACrossover{
        BaseStrategy: *history.NewBaseStrategy("SMACrossover"),
    }
}

func (s *SMACrossover) OnBar(symbol string, bars history.Bars) (history.Event, bool) {
    if len(bars) < 20 {
        return history.Event{}, false
    }

    // Calculate SMAs
    sma10 := bars[0:10].SMA(history.C)
    sma20 := bars[0:20].SMA(history.C)
    prevSma10 := bars[1:11].SMA(history.C)
    prevSma20 := bars[1:21].SMA(history.C)

    // Get current position
    portfolio := s.GetPortfolioManager()
    position, hasPosition := portfolio.Positions[symbol]

    // Close position if exists and crossover in opposite direction
    if hasPosition {
        if (position.Side && prevSma10 > prevSma20 && sma10 < sma20) || // Close long on death cross
           (!position.Side && prevSma10 < prevSma20 && sma10 > sma20) {  // Close short on golden cross
            return s.Close(), true
        }
        return history.Event{}, false
    }

    // Buy on golden cross (10 SMA crosses above 20 SMA)
    if prevSma10 < prevSma20 && sma10 > sma20 {
        return s.Buy(), true // Uses default size of 1000
        // Or with custom size: return s.BuyEvent(2000, bars[0].Close), true
    }

    // Sell on death cross (10 SMA crosses below 20 SMA)
    if prevSma10 > prevSma20 && sma10 < sma20 {
        return s.Sell(), true // Uses default size of 1000
        // Or with custom size: return s.SellEvent(2000, bars[0].Close), true
    }

    return history.Event{}, false
}

// Name returns the strategy name
func (s *SMACrossover) Name() string {
    return "SMACrossover"
}

// Usage:
func main() {
    // Initialize components
    hist, _ := history.New()
    strategy := NewSMACrossover()
    
    // Backtest the strategy
    tester := history.NewTester(hist, strategy)
    results, _ := tester.Test(hist.FirstTime(), hist.LastTime())
    
    // Access results
    fmt.Printf("Final Balance: %.2f\n", results.PortfolioStats.Balance)
    fmt.Printf("Win Rate: %.2f%%\n", results.PortfolioStats.WinRate * 100)
    fmt.Printf("Max Drawdown: %.2f%%\n", results.PortfolioStats.MaxDrawdown * 100)
}
```

## API Documentation

### History (history.go)

```go
type History struct {
    bars       map[string]Bars
    update     bool
    C          chan string    // Notify channel for new bars
    Downloader                // Interface for data downloading
    db         *sql.DB        // SQLite3 database connection
}

// Core Methods
func New() (*History, error)                                    // Create new History instance with SQLite DB
func (h *History) SetMaxLimit(v int)                           // Set maximum limit for data requests
func (h *History) StoredSymbols() ([]string, error)           // Get all symbols from database
func (h *History) ReadBars(symbol string) (Bars, error)        // Load bars from database
func (h *History) WriteBars(symbol string, bars Bars) error    // Save bars to database
func (h *History) GetBars(symbol string) Bars                  // Get bars for a symbol
func (h *History) Map() map[string]Bars                        // Get all bars
func (h *History) MinPeriod() time.Duration                    // Get minimum period across all histories
func (h *History) FirstTime() time.Time                        // Get earliest time across all histories
func (h *History) LastTime() time.Time                         // Get latest time across all histories
func (h *History) Load(symbols ...string) error                // Load symbols from database
func (h *History) Add(symbol string, bars Bars) error          // Add new bars to history
func (h *History) Update(enabled bool)                         // Enable/disable auto-updates
func (h *History) Limit(length int) *History                   // Limit data length
func (h *History) LimitTime(start, end time.Time) *History     // Limit data to time range
func (h *History) Unload(symbol string) error                  // Remove symbol from memory
```

### Bar (bar.go)

```go
type Bar struct {
    Time   time.Time `json:"time"`
    Open   float64   `json:"open"`
    High   float64   `json:"high"`
    Low    float64   `json:"low"`
    Close  float64   `json:"close"`
    Volume float64   `json:"volume,omitempty"`
}

// Methods
func (b Bar) MarshalJSON() ([]byte, error)
func (b *Bar) UnmarshalJSON(data []byte) error
func (b Bar) Mode(mode Price) float64
func (b Bar) T() time.Time
func (b Bar) O() float64
func (b Bar) H() float64
func (b Bar) L() float64
func (b Bar) C() float64
func (b Bar) HL2() float64
func (b Bar) HLC3() float64
func (b Bar) OHLC4() float64
func (b Bar) V() float64
func (b Bar) Range() float64
func (b Bar) Body() float64
func (b Bar) BodyHigh() float64
func (b Bar) BodyLow() float64
func (b Bar) Bull() bool
func (b Bar) Bear() bool
func (b Bar) Bullish() bool
func (b Bar) Bearish() bool
func (b Bar) WickUp() float64
func (b Bar) WickDn() float64
func (b Bar) PercMove() float64

// Additional Bar Methods
func (b Bar) Bullish() bool                                    // Returns true if bar closes in upper 33%
func (b Bar) Bearish() bool                                    // Returns true if bar closes in bottom 33%
func (b Bar) PercMove() float64                                // Calculate percentage move (close-open)/open * 100
```

### Bars (bars.go)

```go
type Bars []Bar

// Core Methods
func (bars Bars) Sort() Bars
func (bars Bars) Reverse() Bars
func (bars Bars) Period() time.Duration
func (bars Bars) FirstBar() Bar
func (bars Bars) LastBar() Bar
func (bars Bars) Find(dt time.Time) (int, Bar)
func (bars Bars) TimeSpan(start, end time.Time) Bars
func (bars Bars) JSON() []byte

// Data Export Methods
func (bars Bars) FormatBytes(inputQuery string) ByteData
func (bars Bars) WriteJSON(filename string) error
func (bars Bars) WriteCSV(filename string) error
func (bars Bars) WriteToFile(filename string) error
```

### Events (events.go)

```go
type Event struct {
    Symbol string    
    Name   string    
    Text   string    
    Type   EventType // BUY, SELL, CLOSE
    Time   time.Time 
    Price  float64   
    Size   float64   
}

type Events []Event

// Methods
func NewEvent(symbol string) Event
func (events Events) Sort() Events
func (events Events) Symbol(symbol string) Events
func (events Events) Exists(event Event) bool
func (events Events) FirstEvent() Event
func (events Events) LastEvent() Event
func (events Events) Find(dt time.Time) (int, Event)

// Event Management
func (events *Events) Add(event Event) bool
func (events *Events) Delete(event Event) bool
func (events Events) Map() map[string]Events
func MapEvents(events ...Event) map[string]Events
func ListEvents(ev ...Event) Events
func (ev Events) RemoveIndex(index int) Events
```

### EventHandler (eventhandler.go)

```go
type EventHandler struct {
    handlers   map[EventType][]EventCallback
    strategies []Strategy
    running    bool
}

type EventCallback func(Event) error

// Methods
func NewEventHandler() *EventHandler
func (eh *EventHandler) Subscribe(eventType EventType, callback EventCallback)
func (eh *EventHandler) Unsubscribe(eventType EventType, callback EventCallback)
func (eh *EventHandler) Handle(event Event) error
func (eh *EventHandler) HandleEvents(events Events) error

// Additional EventHandler Methods
func (eh *EventHandler) Clear()                                // Remove all event handlers
func (eh *EventHandler) Start(hist *History, events *Events) error    // Start processing events
func (eh *EventHandler) Stop() error                           // Stop processing events
func (eh *EventHandler) AddStrategy(strategy Strategy) error   // Add a strategy to the handler
func (eh *EventHandler) RemoveStrategy(strategy Strategy) error // Remove a strategy
func (eh *EventHandler) ListStrategies()                      // List all added strategies
```

### Streamer (streamer.go)

```go
type Streamer interface {
    <-chan Bar
}

// Streaming Methods
func (bars Bars) Stream() <-chan Bar
func (bars Bars) StreamDuration(duration time.Duration) <-chan Bar
func (bars Bars) StreamInterval(start, end time.Time, interval time.Duration) <-chan Bar
```

### Indicators (indicators.go)

```go
// Moving Averages
func (bars Bars) SMA(mode Price) float64
func (bars Bars) LWMA(mode Price) float64
func (bars Bars) EMA(mode Price) float64

// Volatility Indicators
func (bars Bars) ATR() float64
func (bars Bars) StDev(mode Price) float64
func (bars Bars) Range() float64

// Price Analysis
func (bars Bars) Highest(mode Price) float64
func (bars Bars) HighestIdx(mode Price) int
func (bars Bars) Lowest(mode Price) float64
func (bars Bars) LowestIdx(mode Price) int
func (bars Bars) LastBullIdx() int
func (bars Bars) LastBearIdx() int

// Pattern Recognition
func (bars Bars) IsEngulfBuy() bool
func (bars Bars) IsEngulfSell() bool

// Helper Functions
func WithinRange(src, dest, r float64) bool
func CalcPercentage(n, total float64) float64

// Additional Technical Indicators
func (bars Bars) RSI(period int) float64                       // Relative Strength Index
func (bars Bars) Stochastic(period int) (k, d float64)        // Stochastic Oscillator (%K and %D)
func (bars Bars) IsPinbarBuy() bool                           // Check for bullish pinbar pattern
func (bars Bars) IsPinbarSell() bool                          // Check for bearish pinbar pattern
func (bars Bars) TDSequential() int                           // TD Sequential indicator
```

### Strategy (strategy.go)

```go
type Strategy interface {
    OnBar(symbol string, bars Bars) (Event, bool)
    Name() string
}

type BaseStrategy struct {
    portfolio *PortfolioManager
}
```

### Portfolio (portfolio.go)

```go
type Position struct {
    Symbol     string
    Side       bool
    EntryTime  time.Time
    EntryPrice float64
    Size       float64
    Current    float64
    PnL        float64
    OpenEvent  Event    // Reference to the event that opened this position
}

type PortfolioManager struct {
    Balance   float64
    Positions map[string]Position
}

// Methods
func NewPortfolioManager(initialBalance float64) *PortfolioManager

// Portfolio Management Features:
// - Dynamic position sizing based on account balance
// - Accurate PnL calculations for both long and short positions
// - Position tracking with reference to opening events
// - Support for multiple positions across different symbols
// - Real-time unrealized PnL tracking
```

## Example Strategies

The `examples/` app ships with a set of production-oriented strategies designed for longer holding periods and swing-to-position trading. Each strategy implements the `history.Strategy` interface and can be backtested via the built-in tester or run live through the event handler.

Select a strategy with the `-strategy` flag:

```bash
go run . -symbol=BTCUSDT12h -strategy=memory1
```
or run on all loaded symbols

```bash
go run . -quote=USDT -tf=1d -strategy=memory1
```

Backtest results can be viewed at `http://localhost:8080/test` after starting the example server.

### Memory

A trend-following strategy built around a decaying price memory line with adaptive deviation bands. Entries trigger on band breakouts; the system flips or opens positions on confirmed trend changes. Designed primarily for **12h** bars, but viable on **4h and daily** timeframes with tuned parameters.

Position sizing uses 65% of current equity per trade.

| Flag       | MemoryStrength | MemoryDecay | DevLen | BandMult | Profile         |
|------------|----------------|-------------|--------|----------|-----------------|
| `memory1`  | 0.12           | 0.96        | 34     | 1.6      | Default         |
| `memory2`  | 0.22           | 0.60        | 23     | 2.1      | 12h short swing |
| `memory3`  | 0.22           | 0.51        | 16     | 2.0      | Experimental    |
| `memory4`  | 0.14           | 1.10        | 38     | 1.6      | 12h long swing  |

#### memory1 results (start balance 10000 on 5000 bars on 12h timeframe)
```
2026/05/27 10:24:25 [TEST] Memory completed with 130 Events (129 Trades)
2026/05/27 10:24:25 [TEST] Final Balance: 32728.91 (+227.29%)
2026/05/27 10:24:25 [TEST] 1 open position(s), Unrealized P&L: +244.71, Equity: 32973.62
2026/05/27 10:24:25 [TEST] Win Rate: 37.21% [W:48|L:81]
2026/05/27 10:24:25 [TEST] Max Drawdown: 43.22%

2026/05/27 10:27:16 [TEST] Memory completed with 87 Events (86 Trades)
2026/05/27 10:27:16 [TEST] Final Balance: 1184618.52 (+11746.19%)
2026/05/27 10:27:16 [TEST] 1 open position(s), Unrealized P&L: -16263.95, Equity: 1168354.57
2026/05/27 10:27:16 [TEST] Win Rate: 48.84% [W:42|L:44]
2026/05/27 10:27:16 [TEST] Max Drawdown: 59.50%

2026/05/27 10:28:36 [TEST] Memory completed with 124 Events (123 Trades)
2026/05/27 10:28:36 [TEST] Final Balance: 188390.82 (+1783.91%)
2026/05/27 10:28:36 [TEST] 1 open position(s), Unrealized P&L: -213.66, Equity: 188177.15
2026/05/27 10:28:36 [TEST] Win Rate: 42.28% [W:52|L:71]
2026/05/27 10:28:36 [TEST] Max Drawdown: 53.15%
```

### Turtle

Classic Donchian breakout system modeled after the original Turtle Trading rules. Enters on a **35-bar** channel breakout, exits on a **20-bar** channel reversal, and uses a **2× ATR(20)** stop from entry. Suited for multi-day to multi-week holds on higher timeframes.

#### turtle results (start balance 10000 on 5000 bars on d1 timeframe)
```
2026/05/27 10:32:52 [TEST] Turtle completed with 142 Events (71 Trades)
2026/05/27 10:32:52 [TEST] Final Balance: 120848.85 (+1108.49%)
2026/05/27 10:32:52 [TEST] Win Rate: 38.03% [W:27|L:44]
2026/05/27 10:32:52 [TEST] Max Drawdown: 60.44%

2026/05/27 10:35:56 [TEST] Turtle completed with 92 Events (46 Trades)
2026/05/27 10:35:56 [TEST] Final Balance: 401034.12 (+3910.34%)
2026/05/27 10:35:56 [TEST] Win Rate: 39.13% [W:18|L:28]
2026/05/27 10:35:56 [TEST] Max Drawdown: 67.84%

2026/05/27 10:36:31 [TEST] Turtle completed with 127 Events (63 Trades)
2026/05/27 10:36:31 [TEST] Final Balance: 201569.88 (+1915.70%)
2026/05/27 10:36:31 [TEST] 1 open position(s), Unrealized P&L: +3745.46, Equity: 205315.34
2026/05/27 10:36:31 [TEST] Win Rate: 49.21% [W:31|L:32]
2026/05/27 10:36:31 [TEST] Max Drawdown: 51.93%
```

### PercScalper

A percentage-dip accumulation strategy (default: buy on **2.5%** intrabar dips). Supports pyramiding up to 20 positions with collective profit tracking, trailing stops, and optional time-based exits. More active than the other strategies but still oriented toward capturing larger moves rather than scalping tick-by-tick.

This is the **default** strategy when no `-strategy` flag is provided.

#### percscalper results (start balance 10000 on 5000 bars on d1 timeframe)
```
2026/05/27 10:42:30 [TEST] PercScalper completed with 622 Events (301 Trades)
2026/05/27 10:42:30 [TEST] Final Balance: 66723.48 (+567.23%)
2026/05/27 10:42:30 [TEST] 20 open position(s), Unrealized P&L: -1204.45, Equity: 65519.03
2026/05/27 10:42:30 [TEST] Win Rate: 77.08% [W:232|L:69]
2026/05/27 10:42:30 [TEST] Max Drawdown: 34.73%
```

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.
