package core

// #include <pthread.h>

import (
	"errors"
	"fmt"
	"frbg/codec"
	"frbg/def"
	"frbg/third"
	"frbg/timer"
	"frbg/util"
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
	Attach(poll *Poll)                   // 绑定poll
	Push(conn IConn, msg *codec.Message) // 消息路由
	Close(conn IConn)                    // 连接关闭的回调
	OnConnect(conn IConn)                // 连接成功的回调
	OnAccept(conn IConn)                 // 新连接的回调
	Tick()                               // 心跳
}

type PollConfig struct {
	MaxConn   int   // 最大连接数
	HeartBeat int64 // 心跳时间
	Etcd      bool
}

var upgrader = &ws.Upgrader{
	ReadBufferSize:  1024 * 64,
	WriteBufferSize: 1024 * 64,
	OnHeader: func(key, value []byte) (err error) {
		log.Printf("non-websocket header: %q=%q", key, value)
		return
	},
	Protocol: func(b []byte) bool {
		log.Printf("protocol: %q", b)
		return true
	},
}

type RpcCallback func(*codec.Message, error)

func NewPollConfig() *PollConfig {
	return &PollConfig{
		MaxConn:   10000,
		HeartBeat: 60,
		Etcd:      false,
	}
}

type Poll struct {
	epollFd     int               // epoll fd
	eventFd     int               // event fd
	wsListenFd  int               // ws监听fd
	wsListener  *net.TCPListener  // ws监听
	tcpListenFd int               // tcp监听fd
	tcpListener *net.TCPListener  // tcp监听
	connNum     int64             // 改为 int64 便于原子操作
	pollConfig  *PollConfig       // 配置
	queue       *util.Esqueue     // 事件队列
	handle      Handler           // 处理
	ServerConf  *ServerConfig     // 服务配置
	events      []unix.EpollEvent // 重用事件数组
	ticker      *time.Ticker      // 定时器
	heartBeat   *time.Ticker      // 心跳定时器
}

func NewPoll(sconf *ServerConfig, pconf *PollConfig, handle Handler) *Poll {
	epollFd, err := unix.EpollCreate1(0)
	must(err)
	eventFd, err := unix.Eventfd(0, unix.EFD_CLOEXEC)
	must(err)
	err = unix.EpollCtl(epollFd, unix.EPOLL_CTL_ADD, eventFd, &unix.EpollEvent{Events: unix.EPOLLIN, Fd: int32(eventFd)})
	must(err)
	poll := &Poll{
		epollFd:    epollFd,
		eventFd:    eventFd,
		pollConfig: pconf,
		handle:     handle,
		ServerConf: sconf,
		queue:      new(util.Esqueue),
		events:     make([]unix.EpollEvent, 64), // 预分配事件数组
	}
	handle.Attach(poll)
	log.Printf("AddPoll epollFd:%d eventFd:%d", epollFd, eventFd)
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
		third.Put(conf.Svid(), conf.Addr)
	}

	// 监听tcp
	p.tcpListener, p.tcpListenFd = p.Listen(conf.Addr)
	if conf.ServerType == def.ST_Gate {
		p.wsListener, p.wsListenFd = p.Listen(fmt.Sprintf("%s:%d", conf.IP(), conf.Port()+1))
	}
	log.Printf("Start tcpListenFd:%d wsListenFd:%d", p.tcpListenFd, p.wsListenFd)

	// 添加定时事件
	if conf.ServerType != def.ST_Gate {
		timer.AddTrigger(func() {
			p.Trigger(def.ET_Timer)
		})
	}

	// 开始轮询
	wg.Add(1)
	go p.LoopRun()
	go p.CheckTimeout()
	go p.ConnTick()
}

func (p *Poll) Listen(addr string) (*net.TCPListener, int) {
	// 监听tcp
	listener, err := reuseport.Listen("tcp", addr)
	if err != nil {
		log.Printf("listen error: %s", err)
		must(err)
	}
	log.Printf("listen addr:%s success", addr)
	tcpListener, listenFd, err := GetListenerFd(listener)
	if err != nil {
		log.Printf("get listener fd error: %s", err)
		must(err)
	}
	log.Printf("AddListener fd:%d conf:%+v\n", listenFd, addr)
	err = unix.EpollCtl(p.epollFd, syscall.EPOLL_CTL_ADD, listenFd, &unix.EpollEvent{Events: unix.EPOLLIN, Fd: int32(listenFd)})
	if err != nil {
		log.Printf("add epoll error: %s", err)
		must(err)
	}
	return tcpListener, listenFd
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
		if err := third.Del(p.ServerConf.Svid()); err != nil {
			errs = append(errs, fmt.Errorf("etcd del error: %w", err))
		}
	}

	// 关闭连接connfd
	connMgr.Range(func(conn IConn) error {
		return conn.Close()
	})

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
	if p.tcpListenFd > 0 {
		if err := unix.Close(p.tcpListenFd); err != nil {
			errs = append(errs, fmt.Errorf("close listen fd error: %w", err))
		}
	}
	if p.wsListenFd > 0 {
		if err := unix.Close(p.wsListenFd); err != nil {
			errs = append(errs, fmt.Errorf("close ws listen fd error: %w", err))
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
				log.Printf("process register event error: %v", err)
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
				// log.Printf("timer tick")
				// p.handle.Tick()
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

	if fd == p.eventFd {
		return p.processEventFd()
	}

	if fd == p.tcpListenFd {
		return p.processAcceptTcp()
	}

	if fd == p.wsListenFd {
		return p.processAcceptWebSocket()
	}

	return p.processClientData(fd)
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
func (p *Poll) processAcceptTcp() error {
	// 1. 接受新连接
	conn, err := p.tcpListener.AcceptTCP()
	if err != nil {
		log.Printf("AcceptTCP error: %v", err)
		return nil // 临时错误不返回错误，继续循环
	}

	// 2. 获取socket文件描述符
	fd := socketFD(conn)
	if fd == -1 {
		log.Printf("failed to get socket fd:%d", fd)
		return errors.New("failed to get socket fd")
	}

	// 3. 设置非阻塞模式（关键优化点）
	if err := syscall.SetNonblock(fd, true); err != nil {
		log.Printf("failed to set nonblock:%d", fd)
		return err
	}

	// 4. 添加到epoll监听
	event := &unix.EpollEvent{Events: unix.EPOLLIN, Fd: int32(fd)}
	if err := unix.EpollCtl(p.epollFd, syscall.EPOLL_CTL_ADD, fd, event); err != nil {
		log.Printf("failed to epoll ctl add:%d", fd)
		return fmt.Errorf("epoll ctl add error: %w", err)
	}

	// 5. 创建连接对象并存储
	c := &Conn{
		poll:       p,
		conn:       conn,
		fd:         fd,
		activeTime: time.Now().Unix(),
	}

	// 6. 线程安全地更新连接映射
	connMgr.AddConn(c)

	// 7. 记录日志
	log.Printf("Add fd:%d addr:%s conn_num=%d", fd, conn.RemoteAddr().String(), p.getConnNum())
	return nil
}

// 处理新连接
func (p *Poll) processAcceptWebSocket() error {
	// 1. 接受新连接
	conn, err := p.wsListener.AcceptTCP()
	if err != nil {
		log.Printf("AcceptTCP error: %v", err)
		return nil // 临时错误不返回错误，继续循环
	}

	// 1. 检查连接数限制
	if p.getConnNum() >= p.pollConfig.MaxConn {
		conn.Close()
		return fmt.Errorf("connection limit exceeded: %d", p.pollConfig.MaxConn)
	}

	// 2.处理websocket升级
	if _, err = upgrader.Upgrade(conn); err != nil {
		log.Printf("websocket upgrade error: %s fd", err)
		conn.Close()
		return nil
	}

	// 2. 获取socket文件描述符
	fd := socketFD(conn)
	if fd == -1 {
		conn.Close()
		return fmt.Errorf("failed to get socket fd:%d", fd)
	}

	// 3. 设置非阻塞模式（关键优化点）
	if err := syscall.SetNonblock(fd, true); err != nil {
		conn.Close()
		return err
	}

	// 4. 添加到epoll监听
	event := &unix.EpollEvent{Events: unix.EPOLLIN, Fd: int32(fd)}
	if err := unix.EpollCtl(p.epollFd, syscall.EPOLL_CTL_ADD, fd, event); err != nil {
		conn.Close()
		return fmt.Errorf("epoll ctl add %d error: %w", fd, err)
	}

	// 5. 创建连接对象并存储
	c := &WsConn{
		Conn: Conn{
			poll:       p,
			conn:       conn,
			fd:         fd,
			activeTime: time.Now().Unix(),
		},
	}

	// 6. 线程安全地更新连接映射
	connMgr.AddConn(c)

	// 7. 记录日志并触发回调
	log.Printf("Add fd:%d addr:%s conn_num=%d", fd, conn.RemoteAddr().String(), p.getConnNum())
	return nil
}

// 处理客户端数据
func (p *Poll) processClientData(fd int) error {
	// 1. 线程安全地查找连接对象
	conn := connMgr.GetByFd(fd)
	if conn == nil {
		log.Printf("connection not found for fd: %d", fd)
		return nil
	}

	// 3. 读取并解析网络消息
	msg, err := conn.Read()
	if err != nil {
		log.Printf("processClientData fd:%d read error: %v", fd, err)
		p.Del(fd) // 读取失败则关闭连接
		return nil
	}

	// 4. 处理心跳包
	if msg.IsHeartBeat() {
		conn.SetActiveTime(time.Now().Unix())
		return nil
	}

	log.Printf("processClientData fd:%d msg:%v", fd, msg)
	// 5. 处理RPC响应
	if rpcMgr.HandleRpcResponse(msg) {
		return nil
	}

	// 6. 路由消息到业务处理器
	if p.handle != nil {
		p.handle.Push(conn, msg)
	}
	return nil
}

// 优化Del函数
func (p *Poll) Del(fd int) error {
	if err := unix.EpollCtl(p.epollFd, syscall.EPOLL_CTL_DEL, fd, nil); err != nil {
		return fmt.Errorf("epoll ctl del error: %w", err)
	}

	p.decrConnNum()

	conn := connMgr.DelByFd(fd)

	log.Printf("Del fd:%d addr:%s conn_num=%d", fd, conn.String(), p.getConnNum())
	p.handle.Close(conn)
	return conn.Close()
}

// 快速修复版本 - 减少锁持有时间
func (p *Poll) CheckTimeout() {
	p.ticker = time.NewTicker(time.Second)
	for range p.ticker.C {
		// 获取当前时间避免重复计算
		now := time.Now().Unix()
		timeoutDuration := p.pollConfig.HeartBeat
		timeoutFds := make([]int, 0, 64)

		// 只收集需要删除的FD，不立即删除
		connMgr.Range(func(conn IConn) error {
			if now-conn.ActiveTime() > timeoutDuration {
				log.Printf("tcpConns timeout fd:%d active_time:%d timeout_duration:%d now:%d", conn.Fd(), conn.ActiveTime(), timeoutDuration, now)
				timeoutFds = append(timeoutFds, conn.Fd())
			}
			return nil
		})

		// 在锁外删除，避免阻塞
		for _, fd := range timeoutFds {
			log.Printf("tcpConns timeout fd:%d", fd)
			p.Del(fd)
		}
	}
}

func (p *Poll) ConnTick() {
	p.heartBeat = time.NewTicker(time.Second)
	for range p.heartBeat.C {
		// 发送心跳
		connMgr.HeartBeat()
	}
}

func (p *Poll) Connect(conf *ServerConfig) (*Conn, error) {
	tcpConn, err := net.DialTCP("tcp", nil, &net.TCPAddr{IP: conf.IP(), Port: conf.Port()})
	if err != nil {
		log.Printf("Connect error: %v", err)
		return nil, err
	}
	fd := socketFD(tcpConn)
	log.Printf("AddConnector fd:%d conf:%+v\n", fd, conf)
	if err := unix.EpollCtl(p.epollFd, syscall.EPOLL_CTL_ADD, fd, &unix.EpollEvent{Events: unix.EPOLLIN, Fd: int32(fd)}); err != nil {
		tcpConn.Close()
		return nil, err
	}
	conn := &Conn{
		poll:       p,
		conn:       tcpConn,
		fd:         fd,
		activeTime: time.Now().Unix(),
		svid:       conf.Svid(),
	}

	log.Printf("Connect fd:%d addr:%s", fd, tcpConn.RemoteAddr().String())

	connMgr.AddConn(conn)
	p.incrConnNum()
	p.handle.OnConnect(conn)
	return conn, nil
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
// GetListenerFd 从reuseport.Listen返回的listener中提取fd
func GetListenerFd(listener net.Listener) (*net.TCPListener, int, error) {
	// 步骤1：将net.Listener转为*net.TCPListener（reuseport.Listen返回的是TCPListener）
	tcpListener, ok := listener.(*net.TCPListener)
	if !ok {
		return nil, -1, fmt.Errorf("listener不是*net.TCPListener类型")
	}

	// 步骤2：获取listener的底层os.File对象
	file, err := tcpListener.File()
	if err != nil {
		return nil, -1, fmt.Errorf("获取listener file失败: %w", err)
	}
	// 注意：不要关闭file！关闭会导致listener失效

	// 步骤3：提取fd（Fd()返回uintptr，需转为int）
	fd := int(file.Fd())
	if fd < 0 {
		return nil, -1, fmt.Errorf("获取的fd无效: %d", fd)
	}

	return tcpListener, fd, nil
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func (poll *Poll) GetServer(svid uint16) IConn {
	conn := connMgr.GetBySid(svid)
	if conn != nil {
		return conn
	}

	addr := third.Get(svid)
	if addr == "" {
		return nil
	}

	conf := &ServerConfig{
		ServerType: ServerType(svid),
		ServerId:   ServerId(svid),
		Addr:       addr,
	}

	if conn, err := poll.Connect(conf); err == nil {
		return conn
	} else {
		log.Printf("Connect error: %v", err)
	}
	return nil
}
