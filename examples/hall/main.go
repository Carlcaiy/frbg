package main

import (
	"flag"
	"fmt"
	"frbg/def"
	"frbg/examples/hall/route"
	"frbg/network"
	"time"
)

func main() {
	port := 6676
	sid := 1
	flag.IntVar(&port, "p", 6676, "-p 6676")
	flag.IntVar(&sid, "s", 1, "-s 1")
	flag.Parse()

	serverConfig := &network.ServerConfig{
		Addr:       fmt.Sprintf(":%d", port),
		ServerType: def.ST_Hall,
		ServerId:   uint32(sid),
		Subs:       []def.ServerType{def.ST_Game},
	}

	pollConfig := &network.PollConfig{
		HeartBeat: time.Millisecond * 100,
		MaxConn:   50000,
	}

	network.Serve(serverConfig, pollConfig, route.NewLocal(serverConfig))
}
