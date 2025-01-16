package main

import (
	"frbg/def"
	"frbg/examples/game/route"
	"frbg/network"
)

func main() {
	serverConfig := &network.ServerConfig{
		Addr:       ":6686",
		ServerType: def.ST_Game,
		ServerId:   1,
	}
	pollConfig := &network.PollConfig{
		MaxConn: 50000,
	}
	network.Serve(pollConfig, route.NewLocal(serverConfig), serverConfig)
	network.Wait()
}
