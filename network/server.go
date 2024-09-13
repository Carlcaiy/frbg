package network

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var wg sync.WaitGroup

// 控制所有使用select阻塞功能的组件结束阻塞
var closech = make(chan struct{})

type IClose interface {
	Close()
}

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	log.SetOutput(os.Stdout)
}

func Serve(pconf *PollConfig, handle Handler, sconfs ...*ServerConfig) {

	handle.Init()

	closer := []IClose{}
	for i := range sconfs {
		poll := NewPoll(sconfs[i], pconf, handle)
		poll.Init()
		etcd = NewEtcd(sconfs[i], handle)
		etcd.Init()
		closer = append(closer, poll, etcd)
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	sig := <-ch
	if sig == syscall.SIGQUIT || sig == syscall.SIGTERM || sig == syscall.SIGINT {
		for _, c := range closer {
			c.Close()
		}
	}
	wg.Wait()
}
