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
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	log.SetOutput(os.Stdout)
	timer.Init(time.Millisecond * 10)
}

func main() {
	port := 8081
	sid := 1
	flag.IntVar(&port, "p", 8080, "-p 6666")
	flag.IntVar(&sid, "sid", 1, "-sid 1")
	flag.Parse()
	serverConfig := &network.ServerConfig{
		Addr:       fmt.Sprintf(":%d", port),
		ServerType: def.ST_WsGate,
		ServerId:   uint8(sid),
	}
	pollConfig := &network.PollConfig{
		MaxConn: 10000,
		Etcd:    true,
	}

	poll := network.NewPoll(serverConfig, pollConfig, route.New())
	poll.Start()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	sig := <-ch
	if sig == syscall.SIGQUIT || sig == syscall.SIGTERM || sig == syscall.SIGINT {
		log.Println("signal kill")
		poll.Close()
	}

	log.Println("free")
}
