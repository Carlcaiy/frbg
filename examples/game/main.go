package main

import (
	"frbg/def"
	"frbg/examples/game/route"
	"frbg/network"
	"log"
	"os"
	"os/signal"
	"syscall"
)

// 初始化日志
func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	log.SetOutput(os.Stdout)
}

// 创建配置
func main() {
	serverConfig := &network.ServerConfig{
		Addr:       ":6686",
		ServerType: def.ST_Game,
		ServerId:   1,
	}
	pollConfig := &network.PollConfig{
		MaxConn: 10000,
	}
	poll := network.NewPoll(serverConfig, pollConfig, route.NewLocal())
	poll.Start()
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	sig := <-ch
	if sig == syscall.SIGQUIT || sig == syscall.SIGTERM || sig == syscall.SIGINT {
		log.Println("signal kill")
		network.Signal(poll)
	}
	network.Wait(poll)
	log.Println("free")
}
