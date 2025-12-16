package local

import (
	"fmt"
	"frbg/codec"
	"frbg/network"
	"frbg/timer"
	"log"
	"runtime"

	"google.golang.org/protobuf/proto"
)

type Handle func(*Input) error

type BaseLocal struct {
	*network.Poll
	m_route map[uint16]Handle // 路由
	*timer.TaskCtl
}

func NewBase() *BaseLocal {
	return &BaseLocal{
		m_route: make(map[uint16]Handle),
		TaskCtl: timer.NewTaskCtl(),
	}
}

func (l *BaseLocal) Attach(poll *network.Poll) {
	l.Poll = poll
	serverType = poll.ServerConf.ServerType
}

func (l *BaseLocal) Init() {
	log.Println("base.Init")
}

// 连接成功的回调
func (l *BaseLocal) OnConnect(conn *network.Conn) {
}

func (l *BaseLocal) OnAccept(conn *network.Conn) {

}

func (l *BaseLocal) Close(conn *network.Conn) {
}

func (l *BaseLocal) Tick() {
	l.FrameCheck()
}

func (l *BaseLocal) AddRoute(cmd uint16, h Handle) {
	if _, ok := l.m_route[cmd]; ok {
		log.Printf("repeated Add cmd:%d\n", cmd)
	}
	log.Printf("add route cmd:%d\n", cmd)
	l.m_route[cmd] = h
}

func (l *BaseLocal) Route(conn *network.Conn, msg *codec.Message) error {
	// 1. 检查消息是否为空
	if msg == nil {
		return fmt.Errorf("msg is nil")
	}
	// 2. 检查消息是否有路由
	if handle, ok := l.m_route[msg.Cmd]; ok {
		defer l.CatchEx()
		return handle(NewInput(conn, msg))
	} else {
		return fmt.Errorf("call: not find cmd %d", msg.Cmd)
	}
}

func (l *BaseLocal) Send(svid uint16, msg *codec.Message) error {
	conn := l.Poll.GetServer(svid)
	if conn == nil {
		return fmt.Errorf("error not find server %d", svid)
	}
	return conn.Write(msg)
}

func (l *BaseLocal) RpcCall(svid uint16, cmd uint16, req proto.Message, rsp proto.Message) error {
	conn := l.Poll.GetServer(svid)
	if conn == nil {
		return fmt.Errorf("error not find server %d", svid)
	}

	msg, err := conn.RpcWrite(cmd, req, 1000)
	if err != nil {
		return err
	}
	return msg.Unpack(rsp)
}

func (l *BaseLocal) RpcCallAsync(svid uint16, cmd uint16, req proto.Message, f func(msg *codec.Message, err error)) error {
	conn := l.Poll.GetServer(svid)
	if conn == nil {
		return fmt.Errorf("error not find server %d", svid)
	}

	return conn.RpcWriteAsync(cmd, req, f)
}

var buf = make([]byte, 1024)

func (l *BaseLocal) CatchEx() {
	if err := recover(); err != nil {
		runtime.Stack(buf, false)
		log.Println(string(buf))
	}
}
