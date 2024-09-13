package network

import (
	"frbg/def"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var wg sync.WaitGroup

// 控制所有使用select阻塞功能的组件结束阻塞
var closech = make(chan struct{})

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	log.SetOutput(os.Stdout)
}

func Serve(sconf *ServerConfig, pconf *PollConfig, handle Handler) {

	handle.Init()

	poll := NewPoll(pconf, handle)
	poll.AddListener(sconf)

	go poll.LoopRun()

	etcd = NewEtcd(sconf)
	etcd.Init()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	sig := <-ch
	if sig == syscall.SIGQUIT || sig == syscall.SIGTERM || sig == syscall.SIGINT {
		poll.Trigger(def.ET_Close)
		close(closech)
	}

	wg.Wait()
	poll.Close()
	etcd.Close()
}
