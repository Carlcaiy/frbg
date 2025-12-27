package core

import (
	"log"
	"sync"
)

var connMgr = NewServerMgr()

type ConnMgr struct {
	lock    sync.Mutex
	conns   map[int]IConn
	servers map[uint16]IConn
}

func NewServerMgr() *ConnMgr {
	return &ConnMgr{
		lock:    sync.Mutex{},
		conns:   make(map[int]IConn),
		servers: make(map[uint16]IConn),
	}
}

func (s *ConnMgr) GetBySid(svid uint16) IConn {
	s.lock.Lock()
	for id, conn := range s.servers {
		if id == svid {
			s.lock.Unlock()
			return conn
		}
	}
	s.lock.Unlock()
	return nil
}

func (s *ConnMgr) GetByFd(fd int) IConn {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.conns[fd]
}

func (s *ConnMgr) AddServe(conn IConn) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.conns[conn.Fd()] = conn
	log.Printf("AddServe svid:%d conn:%v", conn.Svid(), conn.String())
}

func (s *ConnMgr) DelBySid(svid uint16) {
	s.lock.Lock()
	defer s.lock.Unlock()
	conn := s.servers[svid]
	if conn == nil {
		return
	}
	delete(s.conns, conn.Fd())
	delete(s.servers, svid)
}

func (s *ConnMgr) DelByFd(fd int) IConn {
	s.lock.Lock()
	defer s.lock.Unlock()
	conn := s.conns[fd]
	if conn == nil {
		return nil
	}
	delete(s.servers, conn.Svid())
	delete(s.conns, fd)
	return conn
}

func (s *ConnMgr) Range(f func(conn IConn) error) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	for svid, conn := range s.conns {
		if err := f(conn); err != nil {
			delete(s.conns, svid)
			log.Printf("DelServe svid:%d conn:%v", svid, conn.String())
			return err
		}
	}
	return nil
}

func (s *ConnMgr) Clients() []IConn {
	s.lock.Lock()
	defer s.lock.Unlock()
	clients := make([]IConn, 0, len(s.conns))
	for _, conn := range s.conns {
		clients = append(clients, conn)
	}
	return clients
}
