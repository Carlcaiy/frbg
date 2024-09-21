package network

// #include <pthread.h>

import (
	"errors"
	"fmt"
	"frbg/def"
	"frbg/parser"
	"frbg/register"
	"frbg/timer"
	"log"
	"net"
	_ "net/http/pprof"
	"reflect"
	"syscall"

	"github.com/gobwas/ws"
	reuseport "github.com/kavu/go_reuseport"
	"golang.org/x/sys/unix"
)

type Handler interface {
	Init()                                       // Handler的初始化
	Route(conn *Conn, msg *parser.Message) error // 消息路由
	Close(conn *Conn)                            // 连接关闭的回调
	OnConnect(conn *Conn)                        // 连接成功的回调
	OnAccept(conn *Conn)                         // 新连接的回调
	Tick()                                       // 心跳
}

type PollConfig struct {
	MaxConn int // 最大连接数
	Etcd    bool
}

type Conn struct {
	*ServerConfig             // 信息
	*net.TCPConn              // 连接
	Fd            int         // 文件描述符
	ActiveTime    int64       // 活跃时间
	Uid           uint32      // 玩家
	ctx           interface{} // 该链接附带信息
}

func (c *Conn) Context() interface{} {
	return c.ctx
}

func (c *Conn) SetContext(d interface{}) {
	c.ctx = d
}

type Poll struct {
	epollFd    int
	eventFd    int
	listenFd   int
	listener   *net.TCPListener
	fdconns    map[int]*Conn
	conn_num   int
	pollConfig *PollConfig
	queue      *esqueue
	handle     Handler
	upgrader   *ws.Upgrader
	serverConf *ServerConfig
}

func NewPoll(sconf *ServerConfig, pconf *PollConfig, handle Handler) *Poll {
	epollFd, err := unix.EpollCreate1(0)
	must(err)
	eventFd, err := unix.Eventfd(0, unix.EFD_CLOEXEC) // unix.EFD_NONBLOCK|unix.EFD_CLOEXEC
	must(err)
	err = unix.EpollCtl(epollFd, unix.EPOLL_CTL_ADD, eventFd, &unix.EpollEvent{Events: unix.EPOLLIN, Fd: int32(eventFd)})
	must(err)
	return &Poll{
		fdconns:    make(map[int]*Conn),
		epollFd:    epollFd,
		eventFd:    eventFd,
		pollConfig: pconf,
		handle:     handle,
		serverConf: sconf,
		queue:      new(esqueue),
	}
}

func (p *Poll) Init() {
	conf := p.serverConf
	if conf == nil || conf.Addr == "" {
		log.Println("error addr", conf.Addr)
		return
	}

	// 注册etcd
	if p.pollConfig.Etcd {
		register.Put(conf.Svid(), conf.Addr)
	}

	// 是否为websocket
	if conf.ServerType == def.ST_WsGate {
		p.upgrader = &ws.Upgrader{
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
	}

	// 监听tcp
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: conf.IP(), Port: conf.Port()})
	reuseport.Listen("tcp", conf.Addr)
	must(err)
	p.listenFd = listenFD(listener)
	p.listener = listener
	log.Printf("AddListener fd:%d conf:%+v\n", p.listenFd, conf)
	unix.EpollCtl(p.epollFd, syscall.EPOLL_CTL_ADD, p.listenFd, &unix.EpollEvent{Events: unix.EPOLLIN, Fd: int32(p.listenFd)})

	// 添加定时事件
	if conf.ServerType != def.ST_WsGate {
		timer.AddTrigger(func() {
			p.Trigger(def.ET_Timer)
		})
	}

	// 开始轮询
	go p.LoopRun()
}

func (p *Poll) Close() {
	log.Println("poll close")

	if p.pollConfig.Etcd {
		register.Del(p.serverConf.Svid())
	}

	// 关闭连接connfd
	for _, c := range p.fdconns {
		c.Close()
	}
	// 关闭epoll监听fd
	if p.epollFd > 0 {
		unix.Close(p.epollFd)
	}
	// 关闭eventFd
	if p.eventFd > 0 {
		unix.Close(p.eventFd)
	}
	// 关闭listenFd
	if p.listenFd > 0 {
		unix.Close(p.listenFd)
	}
}

func (p *Poll) LoopRun() {
	wg.Add(1)
	defer func() {
		wg.Done()
	}()

	log.Printf("start looprun coroutine,st:%d sid:%d addr:%s", p.serverConf.ServerType, p.serverConf.ServerId, p.serverConf.Addr)
	events := make([]unix.EpollEvent, 64)
	for {
		n, err := unix.EpollWait(p.epollFd, events, 100)
		if err != nil && err != unix.EINTR {
			return
		}

		if err := p.queue.ForEach(func(note interface{}) error {
			switch t := note.(type) {
			case def.EventType:
				if t == def.ET_Timer {
					p.handle.Tick()
				} else if t == def.ET_Close {
					return errors.New("signal close")
				} else if t == def.ET_Error {
					return errors.New("error")
				}
			default:
				return fmt.Errorf("unknow type %v", t)
			}
			return nil
		}); err != nil {
			return
		}

		for i := 0; i < n; i++ {
			// log.Println("i:", i, "Fd:", events[i].Fd, "Events", events[i].Events, "pad", events[i].Pad, "listenFd", p.epollFd)
			fd := int(events[i].Fd)
			if p.eventFd == fd {
				data := make([]byte, 8)
				unix.Read(fd, data)
			} else if p.listenFd == fd {
				conn, err := p.listener.AcceptTCP()
				if err != nil {
					log.Println("AcceptTCP", err)
					continue
				}

				if p.upgrader != nil {
					_, err = p.upgrader.Upgrade(conn)
					if err != nil {
						log.Printf("upgrade error: %s", err)
						return
					}
				}
				p.Add(conn)
			} else {
				conn, ok := p.fdconns[fd]
				if !ok {
					log.Println("Get conn", err)
					continue
				}

				msg, err := parser.Read(conn, p.serverConf.ServerType)
				if err != nil {
					log.Printf("parser.Read type:%d err:%s", p.serverConf.ServerType, err.Error())
					p.Del(fd)
					continue
				}
				// log.Printf("Route uid:%d cmd:%d dst:%s\n", msg.UserID, msg.Cmd, msg.Dest())
				if p.handle != nil {
					if err := p.handle.Route(conn, msg); err != nil {
						log.Println("route err:", err)
						p.Del(fd)
						continue
					}
				} else {
					log.Println("handle nil, can't deal message")
				}
			}
		}
	}
}

func (p *Poll) AddConnector(conf *ServerConfig) (*Conn, error) {
	conn, err := net.DialTCP("tcp", nil, &net.TCPAddr{IP: conf.IP(), Port: conf.Port()})
	if err != nil {
		log.Println(err)
		return nil, err
	}
	fd := socketFD(conn)
	log.Printf("AddConnector fd:%d conf:%+v\n", fd, conf)
	if err := unix.EpollCtl(p.epollFd, syscall.EPOLL_CTL_ADD, fd, &unix.EpollEvent{Events: unix.EPOLLIN, Fd: int32(fd)}); err != nil {
		conn.Close()
		return nil, err
	}
	ptr := &Conn{
		TCPConn:      conn,
		ServerConfig: conf,
		Fd:           fd,
	}
	p.fdconns[fd] = ptr
	p.handle.OnConnect(ptr)
	return ptr, nil
}

func (p *Poll) Trigger(tri interface{}) {
	p.queue.Add(tri)
	unix.Write(p.eventFd, []byte{1, 1, 1, 1, 1, 1, 1, 1})
}

func (p *Poll) Del(fd int) {
	err := unix.EpollCtl(p.epollFd, syscall.EPOLL_CTL_DEL, fd, nil)
	if err != nil {
		log.Println("Del", err)
		return
	}
	conn := p.fdconns[fd]
	log.Printf("Del fd:%d addr:%v conn_num=%d\n", fd, conn.RemoteAddr(), p.conn_num)
	p.handle.Close(conn)
	delete(p.fdconns, fd)
	p.conn_num--
	conn.Close()
}

func (p *Poll) Add(conn *net.TCPConn) {
	if p.conn_num >= p.pollConfig.MaxConn {
		log.Println("conn num too much.", p.pollConfig.MaxConn)
		return
	}
	fd := socketFD(conn)
	err := unix.EpollCtl(p.epollFd, syscall.EPOLL_CTL_ADD, fd, &unix.EpollEvent{Events: unix.EPOLLIN, Fd: int32(fd)})
	if err != nil {
		log.Println("Add", err)
		return
	}
	c := &Conn{
		TCPConn: conn,
		Fd:      fd,
	}
	p.fdconns[fd] = c
	p.conn_num++
	log.Printf("Add fd:%d addr:%v conn_num=%d net=%v\n", fd, conn.RemoteAddr().String(), p.conn_num, conn.RemoteAddr().Network())
	p.handle.OnAccept(c)
}

func socketFD(conn *net.TCPConn) int {
	tcpConn := reflect.Indirect(reflect.ValueOf(conn)).FieldByName("conn")
	fdVal := tcpConn.FieldByName("fd")
	pfdVal := reflect.Indirect(fdVal).FieldByName("pfd")
	return int(pfdVal.FieldByName("Sysfd").Int())
}

func listenFD(conn net.Listener) int {
	fdVal := reflect.Indirect(reflect.ValueOf(conn)).FieldByName("fd")
	pfdVal := reflect.Indirect(fdVal).FieldByName("pfd")
	return int(pfdVal.FieldByName("Sysfd").Int())
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
