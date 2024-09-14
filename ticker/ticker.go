package ticker

import (
	"log"
	"time"
)

var ticker *time.Ticker
var events []func()

func Init(t time.Duration) {
	ticker = time.NewTicker(t)
	go func() {
		log.Printf("start ticker(%v) coroutine", t)
		for range ticker.C {
			for _, e := range events {
				e()
			}
		}
	}()
}

func AddEvent(e func()) {
	events = append(events, e)
}
