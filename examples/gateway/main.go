package main

import (
	"flag"
	"fmt"
	"frbg/def"
	"frbg/examples/gateway/route"
	"frbg/network"
	"time"
)

func main() {
	wsport := 8080
	port := 8081
	sid := 1
	flag.IntVar(&wsport, "p", 8080, "-p 6666")
	flag.IntVar(&port, "p", 8081, "-p 6666")
	flag.IntVar(&sid, "s", 1, "-s 1")
	flag.Parse()
	wsserverConfig := &network.ServerConfig{
		Addr:       fmt.Sprintf(":%d", wsport),
		ServerType: def.ST_WsGate,
		ServerId:   uint32(sid),
	}
	serverConfig := &network.ServerConfig{
		Addr:       fmt.Sprintf(":%d", port),
		ServerType: def.ST_Gate,
		ServerId:   uint32(sid),
	}
	pollConfig := &network.PollConfig{
		HeartBeat: time.Millisecond * 100,
		MaxConn:   50000,
	}
	network.Serve(pollConfig, route.New(serverConfig), serverConfig, wsserverConfig)
}
