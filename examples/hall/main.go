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
	flag.IntVar(&sid, "sid", 1, "-sid 1")
	flag.Parse()

	serverConfig := &network.ServerConfig{
		Addr:       fmt.Sprintf(":%d", port),
		ServerType: def.ST_Hall,
		ServerId:   uint8(sid),
	}

	pollConfig := &network.PollConfig{
		HeartBeat: time.Millisecond * 100,
		MaxConn:   50000,
		Etcd:      true,
	}

	network.Serve(pollConfig, route.NewLocal(serverConfig), serverConfig)
	network.Wait()
}
