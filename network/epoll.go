package network

// #include <pthread.h>

import (
	"errors"
	"fmt"
	"frbg/codec"
	"frbg/def"
	"frbg/register"
	"frbg/timer"
	"log"
	"net"
	_ "net/http/pprof"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gobwas/ws"
	reuseport "github.com/kavu/go_reuseport"
	"golang.org/x/sys/unix"
)

var wg sync.WaitGroup

type Handler interface {
	Attach(poll *Poll)                          // 绑定poll
	Init()                                      // Handler的初始化
	Route(conn *Conn, msg *codec.Message) error // 消息路由
	Close(conn *Conn)                           // 连接关闭的回调
	OnConnect(conn *Conn)                       // 连接成功的回调
	OnAccept(conn *Conn)                        // 新连接的回调
	Tick()                                      // 心跳
}

type PollConfig struct {
	MaxConn   int   // 最大连接数
	HeartBeat int64 // 心跳时间
	Etcd      bool
}

func NewPollConfig() *PollConfig {
	return &PollConfig{
		MaxConn:   10000,
		HeartBeat: 60,
		Etcd:      false,
	}
}

type Poll struct {
	epollFd    int
	eventFd    int
	listenFd   int
	listener   *net.TCPListener
	fdConns    map[int]*Conn
	cliConns   map[uint16]*Conn
	connNum    int64 // 改为 int64 便于原子操作
	pollConfig *PollConfig
	queue      *esqueue
	handle     Handler
	upgrader   *ws.Upgrader
	ServerConf *ServerConfig
	events     []unix.EpollEvent // 重用事件数组
	mu         sync.RWMutex      // 保护 fdconns 和 connNum
	ticker     *time.Ticker
	heartBeat  *time.Ticker
}

func NewPoll(sconf *ServerConfig, pconf *PollConfig, handle Handler) *Poll {
	epollFd, err := unix.EpollCreate1(0)
	must(err)
	eventFd, err := unix.Eventfd(0, unix.EFD_CLOEXEC)
	must(err)
	err = unix.EpollCtl(epollFd, unix.EPOLL_CTL_ADD, eventFd, &unix.EpollEvent{Events: unix.EPOLLIN, Fd: int32(eventFd)})
	must(err)
	poll := &Poll{
		fdConns:    make(map[int]*Conn),
		cliConns:   make(map[uint16]*Conn),
		epollFd:    epollFd,
		eventFd:    eventFd,
		pollConfig: pconf,
		handle:     handle,
		ServerConf: sconf,
		queue:      new(esqueue),
		events:     make([]unix.EpollEvent, 64), // 预分配事件数组
	}
	handle.Attach(poll)
	return poll
}

func (p *Poll) Start() {

	conf := p.ServerConf
	if conf == nil || conf.Addr == "" {
		log.Println("error addr", conf.Addr)
		return
	}

	// 初始化配置
	if p.pollConfig.HeartBeat == 0 {
		p.pollConfig.HeartBeat = 60
	}
	// 如果没有设置最大连接数，默认100
	if p.pollConfig.MaxConn == 0 {
		p.pollConfig.MaxConn = 100
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
	wg.Add(1)
	go p.LoopRun()
	go p.ConnCheck()
	go p.ConnTick()
}

// 优化关闭函数
func (p *Poll) Close() error {
	log.Println("poll close")
	var errs []error

	// 停止定时器
	p.ticker.Stop()
	p.heartBeat.Stop()

	// 注销etcd服务
	if p.pollConfig.Etcd {
		if err := register.Del(p.ServerConf.Svid()); err != nil {
			errs = append(errs, fmt.Errorf("etcd del error: %w", err))
		}
	}

	// 关闭连接connfd
	p.mu.Lock()
	for _, c := range p.fdConns {
		if err := c.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close conn %d error: %w", c.Fd, err))
		}
	}
	p.mu.Unlock()

	// 关闭epoll监听fd
	if p.epollFd > 0 {
		if err := unix.Close(p.epollFd); err != nil {
			errs = append(errs, fmt.Errorf("close epoll fd error: %w", err))
		}
	}
	// 关闭eventFd
	if p.eventFd > 0 {
		if err := unix.Close(p.eventFd); err != nil {
			errs = append(errs, fmt.Errorf("close event fd error: %w", err))
		}
	}
	// 关闭listenFd
	if p.listenFd > 0 {
		if err := unix.Close(p.listenFd); err != nil {
			errs = append(errs, fmt.Errorf("close listen fd error: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}
	return nil
}

// 优化事件循环
func (p *Poll) LoopRun() {
	defer wg.Done()

	log.Printf("start looprun coroutine,st:%d sid:%d addr:%s",
		p.ServerConf.ServerType, p.ServerConf.ServerId, p.ServerConf.Addr)

	for {
		n, err := unix.EpollWait(p.epollFd, p.events, 100)
		if err != nil && err != unix.EINTR {
			log.Printf("epoll wait error: %v", err)
			return
		}

		// 处理队列事件
		if err := p.processQueueEvents(); err != nil {
			log.Printf("process queue events error: %v", err)
			return
		}

		// 处理网络事件
		for i := 0; i < n; i++ {
			if err := p.processNetworkEvent(&p.events[i]); err != nil {
				log.Printf("process network event error: %v", err)
			}
		}
	}
}

// 拆分队列事件处理
func (p *Poll) processQueueEvents() error {
	return p.queue.ForEach(func(note interface{}) error {
		switch t := note.(type) {
		case def.EventType:
			switch t {
			case def.ET_Timer:
				p.handle.Tick()
			case def.ET_Close:
				return errors.New("signal close")
			case def.ET_Error:
				return errors.New("error")
			default:
				return fmt.Errorf("unknown event type %v", t)
			}
		default:
			return fmt.Errorf("unknown queue event type %v", t)
		}
		return nil
	})
}

// 拆分网络事件处理
func (p *Poll) processNetworkEvent(event *unix.EpollEvent) error {
	fd := int(event.Fd)

	switch fd {
	case p.eventFd:
		return p.processEventFd()
	case p.listenFd:
		return p.processAccept()
	default:
		return p.processClientData(fd)
	}
}

// 处理eventfd事件
func (p *Poll) processEventFd() error {
	// 创建 8 字节缓冲区
	data := make([]byte, 8)

	// 读取 eventfd 中的数据，清除事件状态
	if _, err := unix.Read(p.eventFd, data); err != nil {
		return fmt.Errorf("read event fd error: %w", err)
	}

	return nil
}

// 处理新连接
func (p *Poll) processAccept() error {
	conn, err := p.listener.AcceptTCP()
	if err != nil {
		if !isTemporaryError(err) {
			log.Printf("AcceptTCP error: %v", err)
		}
		return nil // 临时错误不返回错误，继续循环
	}

	if p.upgrader != nil {
		if _, err = p.upgrader.Upgrade(conn); err != nil {
			log.Printf("websocket upgrade error: %s", err)
			conn.Close()
			return nil
		}
	}

	p.Add(conn)
	return nil
}

// 处理客户端数据
func (p *Poll) processClientData(fd int) error {
	// 1. 线程安全地查找连接对象
	p.mu.RLock()
	conn, ok := p.fdConns[fd]
	p.mu.RUnlock()

	if !ok {
		log.Printf("connection not found for fd: %d", fd)
		return nil
	}

	// 3. 读取并解析网络消息
	msg, err := conn.Read()
	if err != nil {
		log.Printf("codec.Read error: msg:%s err:%s", msg.String(), err.Error())
		p.Del(fd) // 读取失败则关闭连接
		return nil
	}

	// 4. 处理心跳包
	if msg.IsHeartBeat() {
		conn.ActiveTime = time.Now().Unix()
		return nil
	}

	// 5. 路由消息到业务处理器
	if p.handle != nil {
		if err := p.handle.Route(conn, msg); err != nil {
			log.Printf("route error: fd:%d err:%v", fd, err)
			p.Del(fd) // 路由失败则关闭连接
			return nil
		}
	} else {
		log.Printf("handler is nil, cannot process message from fd:%d", fd)
	}
	return nil
}

// 添加辅助函数判断临时错误
func isTemporaryError(err error) bool {
	if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
		return true
	}
	return false
}

// 优化Add函数
func (p *Poll) Add(conn *net.TCPConn) error {
	// 1. 检查连接数限制
	if p.getConnNum() >= p.pollConfig.MaxConn {
		return fmt.Errorf("connection limit exceeded: %d", p.pollConfig.MaxConn)
	}

	// 2. 获取socket文件描述符
	fd := socketFD(conn)
	if fd == -1 {
		return errors.New("failed to get socket fd")
	}

	// 3. 设置非阻塞模式（关键优化点）
	if err := syscall.SetNonblock(fd, true); err != nil {
		return err
	}

	// 4. 添加到epoll监听
	event := &unix.EpollEvent{Events: unix.EPOLLIN, Fd: int32(fd)}
	if err := unix.EpollCtl(p.epollFd, syscall.EPOLL_CTL_ADD, fd, event); err != nil {
		return fmt.Errorf("epoll ctl add error: %w", err)
	}

	// 5. 创建连接对象并存储
	c := &Conn{
		poll:       p,
		conn:       conn,
		Fd:         fd,
		ActiveTime: time.Now().Unix(),
	}

	if p.ServerConf.ServerType == def.ST_WsGate {
		c.Protocol = def.ProtocolWs
	} else {
		c.Protocol = def.ProtocolTcp
	}

	// 6. 线程安全地更新连接映射
	p.mu.Lock()
	p.fdConns[fd] = c
	p.incrConnNum()
	p.mu.Unlock()

	// 7. 记录日志并触发回调
	log.Printf("Add fd:%d addr:%s conn_num=%d",
		fd, conn.RemoteAddr().String(), p.getConnNum())

	p.handle.OnAccept(c)
	return nil
}

// 优化Del函数
func (p *Poll) Del(fd int) error {
	if err := unix.EpollCtl(p.epollFd, syscall.EPOLL_CTL_DEL, fd, nil); err != nil {
		return fmt.Errorf("epoll ctl del error: %w", err)
	}

	p.mu.Lock()
	conn, ok := p.fdConns[fd]
	if !ok {
		p.mu.Unlock()
		return fmt.Errorf("connection not found for fd: %d", fd)
	}
	delete(p.fdConns, fd)
	p.decrConnNum()
	p.mu.Unlock()

	log.Printf("Del fd:%d addr:%s conn_num=%d",
		fd, conn.RemoteAddr(), p.getConnNum())

	p.handle.Close(conn)
	return conn.Close()
}

// 快速修复版本 - 减少锁持有时间
func (p *Poll) ConnCheck() {
	p.ticker = time.NewTicker(time.Second)
	for range p.ticker.C {
		// 获取当前时间避免重复计算
		now := time.Now().Unix()
		timeoutDuration := p.pollConfig.HeartBeat

		// 只收集需要删除的FD，不立即删除
		p.mu.RLock()
		timeoutFds := make([]int, 0, 64)
		for fd, conn := range p.fdConns {
			if now-conn.ActiveTime > timeoutDuration {
				log.Printf("ConnCheck timeout fd:%d active_time:%d timeout_duration:%d now:%d", fd, conn.ActiveTime, timeoutDuration, now)
				timeoutFds = append(timeoutFds, fd)
			}
		}
		p.mu.RUnlock()

		// 在锁外删除，避免阻塞
		for _, fd := range timeoutFds {
			log.Printf("ConnCheck timeout fd:%d", fd)
			p.Del(fd)
		}
	}
}

func (p *Poll) ConnTick() {
	p.heartBeat = time.NewTicker(time.Second)

	for range p.heartBeat.C {
		// 获取读锁，复制连接列表
		p.mu.RLock()
		clients := make([]*Conn, 0, len(p.cliConns))
		for _, conn := range p.cliConns {
			clients = append(clients, conn)
		}
		p.mu.RUnlock()

		// 发送心跳
		for _, cli := range clients {
			msg := codec.AcquireMessage()
			msg.SetFlags(codec.FlagsHeartBeat)
			if err := cli.Write(msg); err != nil {
				log.Printf("ConnTick Write error: fd:%d err:%s", cli.Fd, err.Error())
				// 使用统一的连接清理机制
				p.Del(cli.Fd)
			}
		}
	}
}

func (p *Poll) Connect(conf *ServerConfig) (*Conn, error) {
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
		poll:       p,
		conn:       conn,
		Fd:         fd,
		ActiveTime: time.Now().Unix(),
	}

	// 根据服务类型设置协议
	if conf.ServerType == def.ST_WsGate {
		ptr.Protocol = def.ProtocolWs
	} else {
		ptr.Protocol = def.ProtocolTcp
	}

	// 客戶端连接不受fdConns管理
	p.fdConns[fd] = ptr
	p.handle.OnConnect(ptr)
	return ptr, nil
}

func (p *Poll) Trigger(tri interface{}) {
	p.queue.Add(tri)
	unix.Write(p.eventFd, []byte{1, 1, 1, 1, 1, 1, 1, 1})
}

// 添加连接数原子操作
func (p *Poll) getConnNum() int {
	return int(atomic.LoadInt64(&p.connNum))
}

func (p *Poll) incrConnNum() {
	atomic.AddInt64(&p.connNum, 1)
}

func (p *Poll) decrConnNum() {
	atomic.AddInt64(&p.connNum, -1)
}

// 使用SyscallConn替代反射获取文件描述符
func socketFD(conn *net.TCPConn) int {
	syscallConn, err := conn.SyscallConn()
	if err != nil {
		log.Printf("Failed to get syscall conn: %v", err)
		return -1
	}

	var fd int
	err = syscallConn.Control(func(c uintptr) {
		fd = int(c)
	})
	if err != nil {
		log.Printf("Failed to get file descriptor: %v", err)
		return -1
	}

	return fd
}

// 使用SyscallConn替代反射获取监听器文件描述符
func listenFD(conn *net.TCPListener) int {

	syscallConn, err := conn.SyscallConn()
	if err != nil {
		log.Printf("Failed to get syscall conn: %v", err)
		return -1
	}

	var fd int
	err = syscallConn.Control(func(c uintptr) {
		fd = int(c)
	})
	if err != nil {
		log.Printf("Failed to get file descriptor: %v", err)
		return -1
	}

	return fd
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func (poll *Poll) GetServer(svid uint16) *Conn {
	var conn *Conn
	poll.mu.RLock()
	conn, ok := poll.cliConns[svid]
	if ok {
		poll.mu.RUnlock()
		return conn
	}
	poll.mu.RUnlock()

	addr := register.Get(svid)
	if addr == "" {
		return nil
	}

	conf := &ServerConfig{
		ServerType: ServerType(svid),
		ServerId:   ServerId(svid),
		Addr:       addr,
	}

	if conn, err := poll.Connect(conf); err == nil {
		poll.mu.Lock()
		poll.cliConns[conf.Svid()] = conn
		poll.mu.Unlock()
		return conn
	} else {
		log.Printf("Connect error: %v", err)
	}
	return nil
}
