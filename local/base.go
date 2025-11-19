package local

import (
	"fmt"
	"frbg/codec"
	"frbg/network"
	"frbg/timer"
	"log"
	"runtime"
)

type Handle func(*network.Message) error

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

func (l *BaseLocal) Route(msg *network.Message) error {
	if handle, ok := l.m_route[msg.Cmd]; ok {
		defer l.CatchEx()
		return handle(msg)
	} else {
		return fmt.Errorf("call: not find cmd %d", msg.Cmd)
	}
}

func (l *BaseLocal) Send(msg *codec.Message) error {
	conn := l.Poll.GetClient(&network.ServerConfig{
		ServerId:   msg.DestId,
		ServerType: msg.DestType,
	})
	if conn == nil {
		return fmt.Errorf("error not find server %d", msg.DestId)
	}
	return conn.Write(msg)
}

var buf = make([]byte, 1024)

func (l *BaseLocal) CatchEx() {
	if err := recover(); err != nil {
		runtime.Stack(buf, false)
		log.Println(string(buf))
	}
}
