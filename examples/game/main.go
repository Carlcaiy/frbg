package main

import (
	core "frbg/core"
	"frbg/def"
	"frbg/examples/game/route"
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
	serverConfig := &core.ServerConfig{
		Addr:       ":6686",
		ServerType: def.ST_Game,
		ServerId:   def.SID_MahjongBanbisan,
	}
	pollConfig := &core.PollConfig{
		Etcd:    true,
		MaxConn: 10000,
	}
	poll := core.NewPoll(serverConfig, pollConfig, route.NewLocal())
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
