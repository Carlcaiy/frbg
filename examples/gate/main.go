package main

import (
	"flag"
	"fmt"
	"frbg/def"
	"frbg/examples/gate/route"
	"frbg/network"
	"frbg/timer"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func init() {
	timer.Init(time.Millisecond * 10)
}

func main() {
	wsport := 8080
	port := 8081
	sid := 1
	flag.IntVar(&wsport, "wp", 8080, "-wp 6666")
	flag.IntVar(&port, "p", 8081, "-p 6666")
	flag.IntVar(&sid, "sid", 1, "-sid 1")
	flag.Parse()
	wsserverConfig := &network.ServerConfig{
		Addr:       fmt.Sprintf(":%d", wsport),
		ServerType: def.ST_WsGate,
		ServerId:   uint8(sid),
	}
	serverConfig := &network.ServerConfig{
		Addr:       fmt.Sprintf(":%d", port),
		ServerType: def.ST_Gate,
		ServerId:   uint8(sid),
	}
	pollConfig := &network.PollConfig{
		MaxConn: 10000,
		Etcd:    true,
	}
	poll := network.NewPoll(serverConfig, pollConfig, route.New(serverConfig))
	poll.Start()

	wsPoll := network.NewPoll(wsserverConfig, pollConfig, route.New(wsserverConfig))
	wsPoll.Start()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	sig := <-ch
	if sig == syscall.SIGQUIT || sig == syscall.SIGTERM || sig == syscall.SIGINT {
		log.Println("signal kill")
		network.Signal(wsPoll, poll)
	}

	network.Wait(wsPoll, poll)
	log.Println("free")
}
