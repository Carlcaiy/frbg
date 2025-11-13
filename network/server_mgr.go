package network

import (
	"sync"
)

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
	defer s.lock.Unlock()
	for svid, conn := range s.clients {
		if svid == conf.Svid() {
			return conn
		}
	}
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
