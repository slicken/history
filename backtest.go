/*
	backtest.go is work in progress..

	Portfolio
	- Open
	- Closed

	Position
	- symbol
	- isBuy
	- openTime
	- closeTime
	- openPrice
	- closePrice
	- size
	- profit
	- isClosed

	# MakePosition
	# Add
	# Close

	# getLastPosition
	# UpdateProfit

*/

package history

import (
	"errors"
	"fmt"
	"log"
	"time"
)

var (
	initial = 1000.
)

type Portfolio struct {
	Open       Positions
	Closed     Positions
	Balance    float64
	Unreleased float64
}

type Position struct {
	symbol     string
	isBuy      bool
	openTime   time.Time
	closeTime  time.Time
	openPrice  float64
	closePrice float64
	size       float64
	profit     float64
	perc       float64
	isClosed   bool
}

type Positions []Position

// MakePosition converts Event to Position
func MakePosition(ev Event, size float64) Position {
	var new Position
	new.symbol = ev.Symbol
	new.isBuy = ev.IsBuy()
	new.openTime = ev.Time
	new.openPrice = ev.Price
	new.size = size // ?
	return new
}

// Add a position to portfolio
func (p *Portfolio) Add(new Position) (bool, error) {
	if new.symbol == "" {
		return false, errors.New("symbol is missing")
	}
	if new.openTime.IsZero() {
		return false, errors.New("openTime is zero")
	}
	if new.openPrice == 0. {
		return false, errors.New("openPrice is nil")
	}
	for _, tmp := range p.Open {
		if new.symbol == tmp.symbol && new.openTime == tmp.openTime && new.openPrice == tmp.openPrice {
			return false, errors.New("alredy exist")
		}
	}
	// add to portfolio
	p.Open = append(p.Open, new)
	fmt.Printf("added %s (len=%d) @%.8f isBuy:%v %v\n", new.symbol, len(p.Open), new.openPrice, new.isBuy, new.openTime)
	return true, nil
}

func (p Positions) GetLast(symbol string) (n int, po Position) {
	for n, po = range p {
		if po.symbol == symbol {
			return n, po
		}
	}
	return -1, po
}

func (p Positions) GetLastType(symbol string, isBuy bool) (n int, po Position) {
	for n, po = range p {
		if po.symbol == symbol && po.isBuy == isBuy {
			return n, po
		}
	}
	return -1, po
}

func (p Positions) GetFirst(symbol string) (n int, po Position) {
	for n = len(p) - 1; n >= 0; n-- {
		if po.symbol == symbol {
			return n, po
		}
	}
	return -1, po
}

func (p Positions) GetFirstType(symbol string, isBuy bool) (n int, po Position) {
	for n = len(p) - 1; n >= 0; n-- {
		if po.symbol == symbol && po.isBuy == isBuy {
			return n, po
		}
	}
	return -1, po
}

func (p *Position) Profit(price float64) float64 {
	if p.isClosed {
		return p.profit
	}

	p.perc = 0.
	p.profit = 0.
	if p.isBuy {
		p.perc = price / p.openPrice
		p.profit = (price - p.openPrice) * p.size
	} else {
		p.perc = p.openPrice / price
		p.profit = (p.openPrice - price) * p.size
	}

	// fmt.Println("--------")
	// fmt.Println("openPrice", p.openPrice)
	// fmt.Println("price", price)
	// fmt.Println("size", p.size)
	// fmt.Println("perc", p.perc)
	// fmt.Println("profit", p.profit)
	// fmt.Println("total", (p.size * p.perc))
	// p.perc = perc
	// p.profit = profit
	return p.profit
}

// close the index of given position
func (p *Portfolio) Close(n int, closePrice float64, closeTime time.Time) bool {
	if n > len(p.Open) {
		return false
	}
	pos := p.Open[n]
	pos.closeTime = closeTime
	pos.closePrice = closePrice
	pos.profit = pos.Profit(closePrice)
	pos.isClosed = true

	p.Closed = append(p.Closed, pos)
	// p.Open = append(p.Open[:n], p.Open[n+1:]...)
	p.Open = remove(p.Open, n)
	fmt.Printf("closed pos=%d. %s @%.8f profit=%.2f\n", n, pos.symbol, pos.closePrice, pos.profit)
	return true
}

// PortfolioTest strategies with fake proftfolio balance
func (h *History) PortfolioTest(strategy Strategy, start, end time.Time) (Events, error) {
	if len(h.bars) == 0 {
		return nil, errors.New("no history")
	}

	var Wallet = new(Portfolio)

	var events Events
	log.Printf("[BACKTEST] %s (start: %v ==> end: %v)\n", fmt.Sprintf("%T", strategy)[6:], start.Format(dt_stamp), end.Format(dt_stamp))

	for symbol, bars := range h.bars {
		for streamedBars := range bars.StreamInterval(start, end, bars.Period()) {
			if event, ok := strategy.Run(symbol, streamedBars); ok {
				ok := events.Add(event)
				if !ok {
					continue
				}
				// is a new event
				price := event.Price
				size := initial / price
				pos := MakePosition(event, size)
				added, err := Wallet.Add(pos)
				// check many things
				// 	log.Println("[BACKTEST] NewPosition added %s @%.8f %s\n", event.Symbol, event.Price, EventTypes[event.Type])
				_ = price
				_ = pos
				_ = added
				_ = err
			}
		}
	}

	// fmt.Printf("%s\n", Wallet.Print())
	var wins, total int
	total = len(Wallet.Closed)
	for _, po := range Wallet.Closed {
		if po.profit > 0 {
			wins++
		}
	}
	log.Printf("[BACKTEST] completed with %d Closed Events, wins=%d/%d ratio=%.1f%%\n", total, wins, total, 100*float64(wins)/float64(total))

	_ = Wallet
	return events, nil
}

// remove slice element at index(s) and returns new slice
func remove[T any](slice []T, n int) []T {
	return append(slice[:n], slice[n+1:]...)
}

// func (p *Portfolio) Print() []byte {

// 	var buf = bytes.NewBuffer([]byte("[BACKTEST] SUMMARY\r\n----------------------------"))
// 	for _, pair := range p.Pairs {
// 		buf.WriteString(fmt.Sprintf(`
// symbol:       %s
// pos. open:    %d
// pos. closed:  %d
// win ratio     %d/%d (%.2f%%)
// initial:      %.6f
// balance:      %.6f
// profit:       %.6f
// unreleased:   %.6f
// ----------------------------`, pair.symbol, len(pair.open), len(pair.closed), pair.CountWins(), len(pair.closed), 100*float64(pair.CountWins())/float64(len(pair.closed)), pair.initial, pair.balance, pair.balance-pair.initial, pair.unreleased))
// 	}

// 	return buf.Bytes()
// }
