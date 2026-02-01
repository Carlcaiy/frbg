package local

import (
	"fmt"
	"frbg/codec"
	core "frbg/core"
	"frbg/def"
	"frbg/examples/pb"
	"frbg/timer"
	"log"
	"runtime"

	"google.golang.org/protobuf/proto"
)

type Handle func(*Input)

type BaseLocal struct {
	queue chan *Input
	*core.Poll
	m_route map[uint16]Handle // 路由
	*timer.TaskCtl
}

func NewBase() *BaseLocal {
	return &BaseLocal{
		m_route: make(map[uint16]Handle),
		TaskCtl: timer.NewTaskCtl(),
		queue:   make(chan *Input, 20),
	}
}

func (l *BaseLocal) Attach(poll *core.Poll) {
	l.Poll = poll
}

func (l *BaseLocal) Start() {
	go func() {
		for {
			input := <-l.queue
			l.Route(input)
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
	l.queue <- NewInput(conn, msg)
}

func (l *BaseLocal) AddRoute(cmd uint16, h Handle) {
	if _, ok := l.m_route[cmd]; ok {
		log.Printf("repeated Add cmd:%d\n", cmd)
	}
	log.Printf("add route cmd:%d\n", cmd)
	l.m_route[cmd] = h
}

func (l *BaseLocal) Route(input *Input) {
	// 2. 检查消息是否有路由
	if handle, ok := l.m_route[input.Cmd]; ok {
		defer l.CatchEx(input.Cmd)
		handle(input)
	} else {
		log.Printf("call: not find cmd %d", input.Cmd)
	}
}

func (l *BaseLocal) Send(svid uint16, cmd uint16, req proto.Message) error {
	if svid/100 == def.ST_Gate {
		return fmt.Errorf("error gate server %d", svid)
	}
	conn := l.Poll.GetServer(svid)
	if conn == nil {
		return fmt.Errorf("error not find server %d", svid)
	}
	return conn.WriteBy(cmd, req)
}

func (l *BaseLocal) SendTo(svid uint16, uid uint32, cmd uint16, req proto.Message) error {
	if svid/100 != def.ST_Gate {
		return fmt.Errorf("error gate server %d", svid)
	}
	bs, err := proto.Marshal(req)
	if err != nil {
		return fmt.Errorf("error marshal req %d", svid)
	}
	conn := l.Poll.GetServer(svid)
	if conn == nil {
		return fmt.Errorf("error not find server %d", svid)
	}
	return conn.WriteBy(def.PacketOut, &pb.PacketOut{
		Uid:     []uint32{uid},
		Cmd:     uint32(cmd),
		Payload: bs,
	})
}

func (l *BaseLocal) RpcCall(svid uint16, cmd uint16, req proto.Message, rsp proto.Message) error {
	conn := l.Poll.GetServer(svid)
	if conn == nil {
		return fmt.Errorf("error not find server %d", svid)
	}

	return conn.RpcWrite(cmd, req, rsp, 10000)
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
