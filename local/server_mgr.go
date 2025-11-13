package local

import (
	"frbg/network"
	"sync"
)

type serverMgr struct {
	lock    sync.Mutex
	clients map[uint16]*network.Conn
}

var serveMap serverMgr

func GetServe(conf *network.ServerConfig) *network.Conn {
	serveMap.lock.Lock()
	defer serveMap.lock.Unlock()
	for svid, conn := range serveMap.clients {
		if svid == conf.Svid() {
			return conn
		}
	}
	return nil
}

func AddServe(conf *network.ServerConfig, conn *network.Conn) {
	serveMap.lock.Lock()
	defer serveMap.lock.Unlock()
	serveMap.clients[conf.Svid()] = conn
}

func DelServe(conf *network.ServerConfig) {
	serveMap.lock.Lock()
	defer serveMap.lock.Unlock()
	delete(serveMap.clients, conf.Svid())
}
