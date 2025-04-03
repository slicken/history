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
type MyStrategy struct {
    history.BaseStrategy
}

func (s *MyStrategy) Name() string {
    return "MyStrategy"
}

func (s *MyStrategy) OnBar(symbol string, bars history.Bars) (history.Event, bool) {
    if len(bars) < 20 {
        return history.Event{}, false
    }

    // Calculate indicators
    sma := bars[:20].SMA(history.C)  // 20-period SMA on close price
    lastClose := bars[0].Close

    // Generate buy signal
    if lastClose > sma {
        return history.Event{
            Symbol: symbol,
            Name:   s.Name(),
            Type:   history.MARKET_BUY,
            Time:   bars[0].Time,
            Price:  lastClose,
            Text:   "Price crossed above SMA",
        }, true
    }

    return history.Event{}, false
}

// Using the strategy
func main() {
    hist, _ := history.New()
    strategy := &MyStrategy{}
    
    // Create event handler
    eh := history.NewEventHandler()
    eh.AddStrategy(strategy)
    
    // Start processing events
    eh.Start(hist, new(history.Events))
}
```

## API Documentation

### History (history.go)

Core data management and persistence:

```go
type History struct {
    Downloader           // Interface for data downloading
}

// Core Functions
func New() (*History, error)                                    // Create new History instance with SQLite DB
func (h *History) GetBars(symbol string) Bars                   // Get bars for a symbol
func (h *History) Map() map[string]Bars                        // Get all bars
func (h *History) Load(symbols ...string) error                // Load symbols from database
func (h *History) Add(symbol string, bars Bars) error          // Add new bars to history
func (h *History) Update(enabled bool)                         // Enable/disable auto-updates
func (h *History) Limit(length int) *History                   // Limit data length
func (h *History) LimitTime(start, end time.Time) *History     // Limit data to time range

// Database Operations
func (h *History) StoredSymbols() ([]string, error)           // Get all symbols from database
func (h *History) ReadBars(symbol string) (Bars, error)        // Load bars from database
func (h *History) WriteBars(symbol string, bars Bars) error    // Save bars to database
```

### Bar (bar.go)

Price data structure:

```go
type Bar struct {
    Time   time.Time
    Open   float64
    High   float64
    Low    float64
    Close  float64
    Volume float64
}

// Methods
func (b Bar) Mode(mode Price) float64   // Get price based on mode (O, H, L, C, etc.)
```

### Bars (bars.go)

Collection of price bars with analysis methods:

```go
type Bars []Bar

// Core Methods
func (bars Bars) Sort() Bars                                  // Sort bars by time
func (bars Bars) Reverse() Bars                              // Reverse bar order
func (bars Bars) Period() time.Duration                      // Get timeframe interval
func (bars Bars) FirstBar() Bar                              // Get first bar
func (bars Bars) LastBar() Bar                               // Get last bar
func (bars Bars) Find(dt time.Time) (int, Bar)              // Find bar at specific time
func (bars Bars) TimeSpan(start, end time.Time) Bars        // Get bars within time range
```

### Events (events.go)

Trading signals and event management:

```go
type Event struct {
    Symbol string    // Trading symbol
    Name   string    // Event name
    Text   string    // Additional information
    Type   EventType // Event type
    Time   time.Time // Event time
    Price  float64   // Price at event
    Size   float64   // Position size
}

type Events []Event

// Methods
func (events Events) Sort() Events
func (events Events) Symbol(symbol string) Events
func (events Events) Exists(event Event) bool
func (events *Events) Add(event Event) bool
```

### EventHandler (eventhandler.go)

Strategy execution and event processing:

```go
type EventHandler struct {
    handlers   map[EventType][]EventCallback
    strategies []Strategy
}

// Core Functions
func NewEventHandler() *EventHandler
func (eh *EventHandler) Subscribe(eventType EventType, callback EventCallback)
func (eh *EventHandler) AddStrategy(strategy Strategy) error
func (eh *EventHandler) Start(hist *History, events *Events) error
```

### Utils (utils.go)

Utility functions:

```go
func (h *History) SetMaxLimit(v int)                         // Set maximum limit for data requests
func SplitSymbol(s string) (pair string, tf string)         // Split symbol into pair and timeframe
func ToUnixTime(t time.Time) int64                          // Convert time to Unix timestamp
```

### Streamer (streamer.go)

Data streaming capabilities:

```go
type Streamer interface {
    <-chan Bar
}

func (bars Bars) Stream() <-chan Bar
func (bars Bars) StreamDuration(duration time.Duration) <-chan Bar
func (bars Bars) StreamInterval(start, end time.Time, interval time.Duration) <-chan Bar
```

### Tester (tester.go) - 🚧 In Development

Backtesting engine for strategy testing.

### Strategy (strategy.go) - 🚧 In Development

Strategy interface and base implementations.

### Portfolio (portfolio.go) - 🚧 In Development

Portfolio management and position tracking.

### Indicators (indicators.go)

Technical analysis functions:

```go
// Moving Averages
func (bars Bars) SMA(mode Price) float64    // Simple Moving Average
func (bars Bars) LWMA(mode Price) float64   // Linear Weighted Moving Average
func (bars Bars) EMA(mode Price) float64    // Exponential Moving Average

// Volatility
func (bars Bars) ATR() float64              // Average True Range
func (bars Bars) StDev(mode Price) float64  // Standard Deviation
func (bars Bars) Range() float64            // Price Range

// Price Analysis
func (bars Bars) Highest(mode Price) float64
func (bars Bars) Lowest(mode Price) float64
```

### Charts

#### Highcharts (charts/highcharts.go)

```go
type HighChart struct {
    Type      ChartType
    SMA       []int
    EMA       []int
    Volume    bool
}

func NewHighChart() *HighChart
func (c *HighChart) BuildCharts(bars map[string]Bars, events map[string]Events) ([]byte, error)
```

#### TradingView (charts/tradingview.go)

TradingView chart integration (documentation coming soon).

## Contributing

Contributions are welcome! Areas where we need help:

1. Portfolio management implementation
2. Additional technical indicators
3. Strategy development tools
4. Performance optimizations
5. Documentation improvements
6. Testing and bug fixes

Please follow these steps:
1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

[Your chosen license]

## Disclaimer

This software is for educational purposes only. Use at your own risk. Past performance does not guarantee future results.
