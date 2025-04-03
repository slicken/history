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

```go
type History struct {
    Downloader           // Interface for data downloading
}

// Core Functions
func New() (*History, error)                                    // Create new History instance with SQLite DB
func (h *History) GetBars(symbol string) Bars                   // Get bars for a symbol
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
func (h *History) ReprocessHistory(limit int) error            // Redownload and process history

// Database Operations
func (h *History) StoredSymbols() ([]string, error)           // Get all symbols from database
func (h *History) ReadBars(symbol string) (Bars, error)        // Load bars from database
func (h *History) WriteBars(symbol string, bars Bars) error    // Save bars to database
```

### Bar (bar.go)

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
func (b Bar) Mode(mode Price) float64   // Get price based on mode
func (b Bar) T() time.Time              // Get bar time
func (b Bar) O() float64                // Get open price
func (b Bar) H() float64                // Get high price
func (b Bar) L() float64                // Get low price
func (b Bar) C() float64                // Get close price
func (b Bar) V() float64                // Get volume
func (b Bar) Range() float64            // Get price range (high - low)
func (b Bar) MarshalJSON() ([]byte, error)    // JSON marshaling
func (b *Bar) UnmarshalJSON(data []byte) error // JSON unmarshaling
```

### Bars (bars.go)

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
func (bars Bars) MarshalJSON() ([]byte, error)              // JSON marshaling
func (bars *Bars) UnmarshalJSON(data []byte) error          // JSON unmarshaling

// Time Functions
func (bars Bars) T() []time.Time                            // Get all bar times
func (bars Bars) FirstTime() time.Time                      // Get first bar time
func (bars Bars) LastTime() time.Time                       // Get last bar time

// Price Functions
func (bars Bars) O() []float64                             // Get all open prices
func (bars Bars) H() []float64                             // Get all high prices
func (bars Bars) L() []float64                             // Get all low prices
func (bars Bars) C() []float64                             // Get all close prices
func (bars Bars) V() []float64                             // Get all volumes
```

### Events (events.go)

```go
type Event struct {
    Symbol string    
    Name   string    
    Text   string    
    Type   EventType 
    Time   time.Time 
    Price  float64   
    Size   float64   
}

type Events []Event

// Methods
func NewEvent(symbol string) Event                          // Create new event
func (events Events) Sort() Events                         // Sort events by time
func (events Events) Symbol(symbol string) Events          // Filter events by symbol
func (events Events) Exists(event Event) bool              // Check if event exists
func (events Events) FirstEvent() Event                    // Get first event
func (events Events) LastEvent() Event                     // Get last event
func (events Events) Find(dt time.Time) (int, Event)      // Find event at time
func (events *Events) Add(event Event) bool               // Add new event
func (events *Events) Delete(event Event) bool            // Delete event
func (events Events) Map() map[string]Events              // Map events by symbol
```

### EventHandler (eventhandler.go)

```go
type EventHandler struct {
    handlers   map[EventType][]EventCallback
    strategies []Strategy
    running    bool
}

type EventCallback func(Event) error

// Core Functions
func NewEventHandler() *EventHandler                       // Create new event handler
func (eh *EventHandler) Subscribe(eventType EventType, callback EventCallback)   // Subscribe to event type
func (eh *EventHandler) Unsubscribe(eventType EventType, callback EventCallback) // Unsubscribe from event type
func (eh *EventHandler) Handle(event Event) error         // Handle single event
func (eh *EventHandler) HandleEvents(events Events) error // Handle multiple events
func (eh *EventHandler) Clear()                          // Clear all handlers
func (eh *EventHandler) Start(hist *History, events *Events) error  // Start event processing
func (eh *EventHandler) Stop() error                     // Stop event processing
func (eh *EventHandler) AddStrategy(strategy Strategy) error        // Add strategy
func (eh *EventHandler) RemoveStrategy(strategy Strategy) error     // Remove strategy
```

### Indicators (indicators.go)

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
func (bars Bars) Highest(mode Price) float64    // Highest price in period
func (bars Bars) HighestIdx(mode Price) int     // Index of highest price
func (bars Bars) Lowest(mode Price) float64     // Lowest price in period
func (bars Bars) LowestIdx(mode Price) int      // Index of lowest price

// Price Mode Constants
const (
    O     Price = iota  // Open
    H                   // High
    L                   // Low
    C                   // Close
    HL2                 // (High + Low) / 2
    HLC3                // (High + Low + Close) / 3
    OHLC4               // (Open + High + Low + Close) / 4
    V                   // Volume
)
```

### Utils (utils.go)

```go
// History Utils
func (h *History) SetMaxLimit(v int)                         // Set maximum limit for data requests

// Symbol Utils
func SplitSymbol(s string) (pair string, tf string)         // Split symbol into pair and timeframe
func ToUnixTime(t time.Time) int64                          // Convert time to Unix timestamp
func TFInterval(tf string) time.Duration                    // Convert timeframe string to duration
```

### Streamer (streamer.go)

```go
type Streamer interface {
    <-chan Bar
}

// Streaming Methods
func (bars Bars) Stream() <-chan Bar                                           // Stream bars sequentially
func (bars Bars) StreamDuration(duration time.Duration) <-chan Bar            // Stream bars with delay
func (bars Bars) StreamInterval(start, end time.Time, interval time.Duration) <-chan Bar  // Stream bars at intervals
```

### Tester (tester.go)

```go
type Tester struct {
    hist     *History
    strategy Strategy
    events   *Events
}

// Core Functions
func NewTester(hist *History, strategy Strategy) *Tester    // Create new tester
func (t *Tester) Test(start, end time.Time) (*Events, error) // Run backtest
func (t *Tester) ClearEvents()                             // Clear test events
```

### Strategy (strategy.go)

```go
type Strategy interface {
    OnBar(symbol string, bars Bars) (Event, bool)  // Called for each new bar
    Name() string                                  // Strategy identifier
}

type BaseStrategy struct {
    portfolio *PortfolioManager
}

type PortfolioStrategy interface {
    Strategy
    GetPortfolioManager() *PortfolioManager
}
```

### Portfolio (portfolio.go) - 🚧 In Development

```go
type Position struct {
    Symbol     string    // Trading pair
    Side       bool      // true for long, false for short
    EntryTime  time.Time // When position was opened
    EntryPrice float64   // Entry price
    Size       float64   // Position size
    Current    float64   // Current price
}

type PortfolioManager struct {
    Balance   float64             // Available balance
    Positions map[string]Position // Open positions by symbol
}

// Core Functions
func NewPortfolioManager(initialBalance float64) *PortfolioManager
```

### Charts

#### Highcharts (charts/highcharts.go)

```go
type HighChart struct {
    Type      ChartType
    SMA       []int
    EMA       []int
    Volume    bool
    VolumeSMA int
    Shadow    bool
    SetWidth  string
    SetHeight string
    SetMargin string
}

// Chart Types
type ChartType string
const (
    Candlestick ChartType = "candlestick"
    Ohlc        ChartType = "ohlc"
    Line        ChartType = "line"
    Spline      ChartType = "spline"
)

// Core Functions
func NewHighChart() *HighChart
func (c *HighChart) BuildCharts(bars map[string]Bars, events map[string]Events) ([]byte, error)
func MakeOHLC(bars Bars) ([]byte, error)
func MakeVolume(bars Bars) ([]byte, error)
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
