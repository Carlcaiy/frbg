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
	wsport := 8888
	port := 6666
	sid := 1
	flag.IntVar(&wsport, "wsp", 8080, "-wsp 8080")
	flag.IntVar(&port, "p", 6666, "-p 6666")
	flag.IntVar(&sid, "s", 1, "-s 1")
	flag.Parse()
	wserverConfig := &network.ServerConfig{
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
	network.WsServe(wserverConfig, serverConfig, pollConfig, route.New(serverConfig))
}
