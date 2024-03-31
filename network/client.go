package network

import (
	"frbg/def"
	"os"
	"os/signal"
	"syscall"
)

func Client(sconf *ServerConfig, pconf *PollConfig, handle Handler) {

	poll := NewPoll(pconf)
	poll.handle = handle
	go poll.LoopRun()

	poll.AddConnector(sconf)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	sig := <-ch
	if sig == syscall.SIGQUIT || sig == syscall.SIGTERM || sig == syscall.SIGINT {
		poll.Trigger(def.ET_Close)
	}

	poll.Close()
}
