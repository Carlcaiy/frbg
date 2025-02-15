package main

import (
	"flag"
	"fmt"
	"frbg/def"
	"frbg/examples/gate/route"
	"frbg/network"
	"frbg/timer"
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
		MaxConn: 50000,
		Etcd:    true,
	}
	router := route.New(serverConfig)
	network.Serve(pollConfig, router, serverConfig)
	network.WsServe(pollConfig, router, wsserverConfig)
	network.Wait()
}
