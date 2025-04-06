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

// Timeframe Functions
func TFInterval(tf string) Timeframe
func TFString(tf Timeframe) string
func TFIsValid(tf string) bool
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
    Type   EventType 
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
}

type PortfolioManager struct {
    Balance   float64
    Positions map[string]Position
}

// Methods
func NewPortfolioManager(initialBalance float64) *PortfolioManager
```

### Utility Functions (utils.go)

```go
// Symbol Management
func SplitSymbol(s string) (pair string, tf string)
func ToUnixTime(t time.Time) int64
func calcLimit(last time.Time, period time.Duration) int

// File Operations
func (b ByteData) ToFile(filename string) error
```

### Binance Integration (examples/downloader.go)

```go
type Binance struct{}

// Core Methods
func (e Binance) GetKlines(pair, timeframe string, limit int) (history.Bars, error)
func MakeSymbolMultiTimeframe(currencie string, timeframes ...string) ([]string, error)
func GetExchangeInfo() (ExchangeInfo, error)
```

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
