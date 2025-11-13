package main

import (
	"flag"
	"fmt"
	"frbg/def"
	"frbg/examples/hall/route"
	"frbg/network"
	"frbg/timer"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func init() {
	timer.Init(time.Second)
}

func main() {
	port := 6676
	sid := 1
	flag.IntVar(&port, "p", 6676, "-p 6676")
	flag.IntVar(&sid, "sid", 1, "-sid 1")
	flag.Parse()

	serverConfig := &network.ServerConfig{
		Addr:       fmt.Sprintf(":%d", port),
		ServerType: def.ST_Hall,
		ServerId:   uint8(sid),
	}

	pollConfig := &network.PollConfig{
		MaxConn: 10000,
		Etcd:    true,
	}

	poll := network.NewPoll(serverConfig, pollConfig, route.New(serverConfig))
	poll.Start()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	sig := <-ch
	if sig == syscall.SIGQUIT || sig == syscall.SIGTERM || sig == syscall.SIGINT {
		log.Println("signal kill")
		network.Signal(poll)
	}

	network.Wait(poll)
	log.Println("free")
}
