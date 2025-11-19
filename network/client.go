package network

import (
	"frbg/def"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// 创建一个客户端，能发送和接收信息
func Client(sconf *ServerConfig, pconf *PollConfig, handle Handler) {

	poll := NewPoll(sconf, pconf, handle)
	poll.Start()

	poll.Connect(sconf)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	sig := <-ch
	if sig == syscall.SIGQUIT || sig == syscall.SIGTERM || sig == syscall.SIGINT {
		poll.Trigger(def.ET_Close)
	}

	poll.Close()
}

type ServerMgr struct {
	lock    sync.Mutex
	clients map[uint16]*Conn
}

func NewServerMgr() *ServerMgr {
	return &ServerMgr{
		lock:    sync.Mutex{},
		clients: make(map[uint16]*Conn),
	}
}

func (s *ServerMgr) GetServe(conf *ServerConfig) *Conn {
	s.lock.Lock()
	for svid, conn := range s.clients {
		if svid == conf.Svid() {
			s.lock.Unlock()
			return conn
		}
	}
	s.lock.Unlock()
	return nil
}

func (s *ServerMgr) AddServe(conf *ServerConfig, conn *Conn) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.clients[conf.Svid()] = conn
}

func (s *ServerMgr) DelServe(conf *ServerConfig) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.clients, conf.Svid())
}
