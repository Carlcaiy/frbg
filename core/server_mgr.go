package core

import (
	"log"
	"sync"
)

// serverMgr 内部服务器管理
var serverMgr = NewServerMgr()

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

func (s *ServerMgr) GetServe(svid uint16) *Conn {
	s.lock.Lock()
	for id, conn := range s.clients {
		if id == svid {
			s.lock.Unlock()
			return conn
		}
	}
	s.lock.Unlock()
	return nil
}

func (s *ServerMgr) AddServe(svid uint16, conn *Conn) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.clients[svid] = conn
	log.Printf("AddServe svid:%d conn:%v", svid, conn.RemoteAddr().String())
}

func (s *ServerMgr) DelServe(svid uint16) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.clients, svid)
}

func (s *ServerMgr) Range(f func(conn *Conn) error) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	for svid, conn := range s.clients {
		if err := f(conn); err != nil {
			delete(s.clients, svid)
			log.Printf("DelServe svid:%d conn:%v", svid, conn.RemoteAddr().String())
			return err
		}
	}
	return nil
}

func (s *ServerMgr) Clients() []*Conn {
	s.lock.Lock()
	defer s.lock.Unlock()
	clients := make([]*Conn, 0, len(s.clients))
	for _, conn := range s.clients {
		clients = append(clients, conn)
	}
	return clients
}
