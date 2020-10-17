package history

import (
	"fmt"
	"testing"
	"time"
)

func Test(t *testing.T) {
	var wait chan bool = make(chan bool)
	// u := &Up{}
	// u.tick = c
	h := &Ticker{}

	go h.Handler()

	go Subscribe("C1", h)
	go Subscribe("C2", h)

	<-wait
}

type Ticker struct {
	C []chan string
}

func Subscribe(name string, Ticker *Ticker) {
	var c chan string = make(chan string, 1)
	Ticker.C = append(Ticker.C, c)

	defer fmt.Println("EXIT", name)

	var i int
	for {
		i++
		c <- fmt.Sprintf("%s(%d)", name, i)

		time.Sleep(time.Second)
		if name == "C1" && i > 3 {
			close(c)
			return
		}
	}
}

var v = time.Ticker{}

func (t *Ticker) Handler() {

	for {
		for i, c := range t.C {
			select {
			case msg, ok := <-c:
				if msg == "" {
					fmt.Printf("NIL == t.C[%d]\n", i)
				}
				if !ok {
					fmt.Printf("DEL t.C[%d] %s\n", i, msg)
					t.C = append(t.C[:i], t.C[i+1:]...)
					continue
				}
				fmt.Printf("<-%s ; %v\t len(%d)\n", msg, ok, len(t.C))
			default:
			}
		}
	}
}
