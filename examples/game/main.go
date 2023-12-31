package main

import (
	"frbg/examples/game/route"
	"frbg/network"
	"time"
)

func main() {
	serverConfig := &network.ServerConfig{
		Addr:       ":6686",
		ServerType: network.ST_Game,
		ServerId:   1,
		Subs:       []network.ServerType{network.ST_Gate},
	}
	pollConfig := &network.PollConfig{
		HeartBeat: time.Millisecond * 100,
		MaxConn:   50000,
	}
	network.Serve(serverConfig, pollConfig, route.NewLocal(serverConfig))
}
