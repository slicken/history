/*
	backtest.go is work in progress..
*/

package history

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"time"
)

var (
	size    = 1000.
	initial = 1000.
)

type Portfolio struct {
	Pairs map[string]*Pair
	// initial float64
	// balance float64
}

type Pair struct {
	symbol     string
	initial    float64
	balance    float64
	unreleased float64
	open       map[time.Time]*Position
	closed     map[time.Time]*Position
}

type Position struct {
	id         time.Time
	symbol     string
	isBuy      bool
	openPrice  float64
	closePrice float64
	size       float64
}

// Add a position to portfolio
func (p *Portfolio) Add(pos *Position) error {
	if pos.symbol == "" {
		return errors.New("symbol is missing")
	}
	if pos.openPrice == 0 {
		return errors.New("price is missing")
	}
	if pos.size == 0 {
		return errors.New("size is missing")
	}
	if p.Pairs == nil {
		p.Pairs = make(map[string]*Pair)
	}

	var pair *Pair
	if _pair := p.Pairs[pos.symbol]; _pair != nil {
		pair = _pair
	} else {
		pair = &Pair{
			symbol:  pos.symbol,
			initial: initial,
			balance: initial,
			open:    make(map[time.Time]*Position),
			closed:  make(map[time.Time]*Position),
		}
	}
	p.Pairs[pos.symbol] = pair
	p.Pairs[pos.symbol].open[pos.id] = pos

	return nil
}

func (p *Portfolio) Close(symbol string, price float64) error {
	if _, ok := p.Pairs[symbol]; !ok {
		return errors.New("not exist")
	}
	if len(p.Pairs[symbol].open) == 0 {
		return errors.New("no open")
	}

	// first in first out close
	var pos *Position
	for _, v := range p.Pairs[symbol].open {
		pos = v
		break
	}

	// calc profits -->
	profit := (price - pos.openPrice) * pos.size
	if pos.isBuy {
		p.Pairs[pos.symbol].balance += profit
	} else {
		p.Pairs[pos.symbol].balance -= profit
	}

	pos.closePrice = price
	p.Pairs[pos.symbol].closed[pos.id] = pos
	delete(p.Pairs[pos.symbol].open, pos.id)

	return nil
}

// Pass checks if this event should be passed
func (p *Portfolio) Pass(e Event) bool {
	if _, ok := p.Pairs[e.Symbol]; !ok {
		return false
	}

	if len(p.Pairs[e.Symbol].open) == 0 {
		return true
	}

	return false
}

func (p Pair) CountWins() int {
	var win int
	for _, v := range p.closed {
		profit := (v.closePrice - v.openPrice) * v.size
		if profit > 0 {
			win++
		}
	}

	return win
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

				// fix code ----------------------->

				price := streamedBars.LastBar().C()

				pos := &Position{
					id:        event.Time,
					symbol:    symbol,
					isBuy:     event.IsBuy(),
					openPrice: event.Price,
					size:      size / price,
				}

				// check for EventType to preforme action
				if event.Type == MARKET_BUY || event.Type == MARKET_SELL {
					if err := Wallet.Add(pos); err != nil {
						log.Println("PortfolioAdd:", err.Error())
					}
				}
				// close postions
				if event.Type == CLOSE_BUY || event.Type == CLOSE_SELL {
					Wallet.Close(symbol, price)
				}

				// calc unreleased of open positions
				pair, ok := Wallet.Pairs[pos.symbol]
				if !ok {
					continue
				}
				pair.Update(pos.symbol, price)
			}
		}
	}

	fmt.Printf("%s\n", Wallet.Print())
	var wins, total int
	for _, v := range Wallet.Pairs {
		total += len(v.closed)
		wins += v.CountWins()
	}
	log.Printf("[BACKTEST] completed with %d Events, wins=%d/%d ratio=%.1f%%\n", len(events), wins, total, 100*float64(wins)/float64(total))
	return events, nil
}

func (p *Pair) Update(symbol string, price float64) {
	if symbol != p.symbol {
		return
	}

	p.unreleased = 0.
	for _, pos := range p.open {
		profit := (price - pos.openPrice) * pos.size

		if pos.isBuy {
			p.unreleased += profit
		} else {
			p.unreleased -= profit
		}
	}
}

func (p *Portfolio) Print() []byte {

	var buf = bytes.NewBuffer([]byte("[BACKTEST] SUMMARY\r\n----------------------------"))
	for _, pair := range p.Pairs {
		buf.WriteString(fmt.Sprintf(`
symbol:       %s
pos. open:    %d
pos. closed:  %d
win ratio     %d/%d (%.2f%%)
initial:      %.6f
balance:      %.6f
profit:       %.6f
unreleased:   %.6f
----------------------------`, pair.symbol, len(pair.open), len(pair.closed), pair.CountWins(), len(pair.closed), 100*float64(pair.CountWins())/float64(len(pair.closed)), pair.initial, pair.balance, pair.balance-pair.initial, pair.unreleased))
	}

	return buf.Bytes()
}
