package network

import (
	"frbg/def"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var wg sync.WaitGroup
var mainpoll *Poll
var wspoll *Poll

type IClose interface {
	Close()
}

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	log.SetOutput(os.Stdout)
}

func Serve(pconf *PollConfig, handle Handler, sconf *ServerConfig) {
	mainpoll = NewPoll(sconf, pconf, handle)
	mainpoll.Init()
}

func WsServe(pconf *PollConfig, handle Handler, sconf *ServerConfig) {
	wspoll = NewPoll(sconf, pconf, handle)
	wspoll.Init()
}

func Wait() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	sig := <-ch
	if sig == syscall.SIGQUIT || sig == syscall.SIGTERM || sig == syscall.SIGINT {
		mainpoll.Trigger(def.ET_Close)
		if wspoll != nil {
			wspoll.Trigger(def.ET_Close)
		}
		log.Println("signal kill")
	}
	wg.Wait()
	log.Println("free")
	mainpoll.Close()
	if wspoll != nil {
		wspoll.Close()
	}
}

func NewClient(sconf *ServerConfig) *Conn {
	if mainpoll == nil {
		return nil
	}
	conn, err := mainpoll.AddConnector(sconf)
	if err != nil {
		return nil
	}
	return conn
}
