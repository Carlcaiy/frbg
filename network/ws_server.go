package network

import (
	"frbg/def"
	"frbg/parser"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/gobwas/ws"
)

func WsServe(wsconf *ServerConfig, sconf *ServerConfig, pconf *PollConfig, handle Handler) {
	handle.Init()

	go wsLoop(wsconf, handle)

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

func wsLoop(wsconf *ServerConfig, handle Handler) {
	ln, err := net.ListenTCP("tcp", &net.TCPAddr{IP: wsconf.IP(), Port: wsconf.Port()})
	if err != nil {
		log.Println(err)
		return
	}

	log.Printf("listen %s success", wsconf.Addr)
	u := ws.Upgrader{
		ReadBufferSize:  1024 * 64,
		WriteBufferSize: 1024 * 64,
		OnHeader: func(key, value []byte) (err error) {
			log.Printf("non-websocket header: %q=%q", key, value)
			return
		},
		Protocol: func(b []byte) bool {
			log.Println(string(b))
			return true
		},
	}

	for {
		tcpConn, err := ln.AcceptTCP()
		if err != nil {
			log.Fatal(err)
			return
		}
		_, err = u.Upgrade(tcpConn)
		if err != nil {
			log.Printf("upgrade error: %s", err)
			return
		}
		log.Printf("accept addr:%v", tcpConn.RemoteAddr())
		conn := &Conn{
			ServerConfig: wsconf,
			TCPConn:      tcpConn,
			Fd:           socketFD(tcpConn),
		}
		handle.OnAccept(conn)
		go func() {
			defer tcpConn.Close()
			for {
				select {
				case <-closech:
					log.Println("close")
					return
				default:
					if msg, err := parser.WsRead(conn); err != nil {
						log.Println(err)
						return
					} else {
						handle.Route(conn, msg)
					}
				}
			}
		}()
	}
}
