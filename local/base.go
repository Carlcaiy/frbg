package local

import (
	"fmt"
	"frbg/codec"
	core "frbg/core"
	"frbg/timer"
	"frbg/util"
	"log"
	"runtime"
	"time"

	"google.golang.org/protobuf/proto"
)

type Handle func(*Input) error

type BaseLocal struct {
	queue *util.ArrayQueue
	*core.Poll
	m_route map[uint16]Handle // 路由
	*timer.TaskCtl
}

func NewBase() *BaseLocal {
	return &BaseLocal{
		m_route: make(map[uint16]Handle),
		TaskCtl: timer.NewTaskCtl(),
		queue:   util.NewArrayQueue(128),
	}
}

func (l *BaseLocal) Attach(poll *core.Poll) {
	l.Poll = poll
	serverType = poll.ServerConf.ServerType
}

func (l *BaseLocal) Start() {
	go func() {
		for {
			if l.queue.IsEmpty() {
				time.Sleep(time.Millisecond * 100)
				continue
			}
			input, err := l.queue.Dequeue()
			if err != nil {
				log.Printf("Dequeue error:%s", err.Error())
				continue
			}
			if err := l.Route(input.(*Input)); err != nil {
				log.Printf("Route error:%s", err.Error())
			}
		}
	}()
}

// 连接成功的回调
func (l *BaseLocal) OnConnect(conn core.IConn) {
}

func (l *BaseLocal) OnAccept(conn core.IConn) {

}

func (l *BaseLocal) Close(conn core.IConn) {
}

func (l *BaseLocal) Tick() {
	l.FrameCheck()
}

func (l *BaseLocal) Push(conn core.IConn, msg *codec.Message) {
	l.queue.Enqueue(NewInput(conn, msg))
}

func (l *BaseLocal) AddRoute(cmd uint16, h Handle) {
	if _, ok := l.m_route[cmd]; ok {
		log.Printf("repeated Add cmd:%d\n", cmd)
	}
	log.Printf("add route cmd:%d\n", cmd)
	l.m_route[cmd] = h
}

func (l *BaseLocal) Route(input *Input) error {
	// 2. 检查消息是否有路由
	if handle, ok := l.m_route[input.Cmd]; ok {
		defer l.CatchEx(input.Cmd)
		return handle(input)
	} else {
		return fmt.Errorf("call: not find cmd %d", input.Cmd)
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

	msg, err := conn.RpcWrite(cmd, req, 10000)
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

func (l *BaseLocal) CatchEx(cmd uint16) {
	if err := recover(); err != nil {
		runtime.Stack(buf, false)
		log.Printf("catch exception cmd:%d, stack:%s", cmd, string(buf))
	}
}
