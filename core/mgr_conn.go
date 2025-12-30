package core

import (
	"frbg/codec"
	"log"
	"sync"
)

var connMgr = NewServerMgr()

type ConnMgr struct {
	lock    sync.RWMutex
	conns   map[int]IConn
	servers map[uint16]IConn
}

func NewServerMgr() *ConnMgr {
	return &ConnMgr{
		conns:   make(map[int]IConn),
		servers: make(map[uint16]IConn),
	}
}

func (s *ConnMgr) GetBySid(svid uint16) IConn {
	s.lock.RLock()
	for id, conn := range s.servers {
		if id == svid {
			s.lock.RUnlock()
			return conn
		}
	}
	s.lock.RUnlock()
	return nil
}

func (s *ConnMgr) GetByFd(fd int) IConn {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.conns[fd]
}

func (s *ConnMgr) AddConn(conn IConn) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.conns[conn.Fd()] = conn
	if conn.Svid() > 0 {
		s.servers[conn.Svid()] = conn
	}
	log.Printf("AddConn svid:%d fd:%d conn:%v", conn.Svid(), conn.Fd(), conn.String())
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
	conn, ok := s.conns[fd]
	if !ok {
		return nil
	}
	if conn.Svid() > 0 {
		delete(s.servers, conn.Svid())
	}
	delete(s.conns, fd)
	return conn
}

func (s *ConnMgr) Range(f func(conn IConn) error) error {
	s.lock.RLock()
	defer s.lock.RUnlock()
	for _, conn := range s.conns {
		if err := f(conn); err != nil {
			return err
		}
	}
	return nil
}

func (s *ConnMgr) HeartBeat() {
	s.lock.RLock()
	defer s.lock.RUnlock()
	for _, conn := range s.servers {
		msg := codec.AcquireMessage()
		msg.SetFlags(codec.FlagsHeartBeat)
		if err := conn.Write(msg); err != nil {
			log.Printf("HeartBeat error: %v", err)
			delete(s.conns, conn.Fd())
			delete(s.servers, conn.Svid())
			conn.Close()
		}
		codec.ReleaseMessage(msg)
	}
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
