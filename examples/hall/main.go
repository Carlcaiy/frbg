package main

import (
	"flag"
	"fmt"
	core "frbg/core"
	"frbg/def"
	"frbg/examples/hall/route"
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
	timer.Init(time.Second)
}

func main() {
	port := 6676
	sid := 1
	flag.IntVar(&port, "p", 6676, "-p 6676")
	flag.IntVar(&sid, "sid", 1, "-sid 1")
	flag.Parse()

	serverConfig := &core.ServerConfig{
		Addr:       fmt.Sprintf(":%d", port),
		ServerType: def.ST_Hall,
		ServerId:   uint8(sid),
	}

	pollConfig := &core.PollConfig{
		MaxConn: 10000,
		Etcd:    true,
	}

	poll := core.NewPoll(serverConfig, pollConfig, route.New())
	poll.Start()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	sig := <-ch
	if sig == syscall.SIGQUIT || sig == syscall.SIGTERM || sig == syscall.SIGINT {
		log.Println("signal kill")
		core.Signal(poll)
	}

	core.Wait(poll)
	log.Println("free")
}
