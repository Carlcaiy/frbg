package local

import (
	"fmt"
	"frbg/def"
	"frbg/network"
	"frbg/register"
	"frbg/timer"
	"log"
	"runtime"
)

type Handle func(*network.Message) error

type BaseLocal struct {
	*network.Poll
	m_users   map[uint32]interface{}
	serverMgr *network.ServerMgr
	m_route   map[uint16]Handle // 路由
	*timer.TaskCtl
}

func NewBase() *BaseLocal {
	return &BaseLocal{
		serverMgr: network.NewServerMgr(),
		m_route:   make(map[uint16]Handle),
		TaskCtl:   timer.NewTaskCtl(),
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

func (l *BaseLocal) SendModUid(uid uint32, buf []byte, serverType uint8) error {
	sids := register.Gets(serverType)
	if len(sids) == 0 {
		return fmt.Errorf("error not find server %d", serverType)
	}
	serverId := sids[int(uid)%len(sids)]
	return l.SendToSid(serverId, buf, serverType)
}

func (l *BaseLocal) SendToGame(gameId uint8, buf []byte) error {
	return l.SendToSid(gameId, buf, def.ST_Game)
}

func (l *BaseLocal) SendToGate(gateId uint8, buf []byte) error {
	return l.SendToSid(gateId, buf, def.ST_Gate)
}

func (l *BaseLocal) SendToSid(serverId uint8, buf []byte, serverType uint8) error {
	conn := l.Poll.GetClient(&network.ServerConfig{
		ServerId:   serverId,
		ServerType: serverType,
	})
	if conn == nil {
		return fmt.Errorf("error not find server %d", serverId)
	}
	return conn.Write(buf)
}

func (l *BaseLocal) GetUser(uid uint32) interface{} {
	if u, ok := l.m_users[uid]; ok {
		return u
	}
	log.Printf("get user:%d failed", uid)
	return nil
}

func (l *BaseLocal) SetUser(uid uint32, data interface{}) {
	l.m_users[uid] = data
	log.Printf("add user:%d", uid)
}

func (l *BaseLocal) DelUser(uid uint32) {
	delete(l.m_users, uid)
}

var buf = make([]byte, 1024)

func (l *BaseLocal) CatchEx() {
	if err := recover(); err != nil {
		runtime.Stack(buf, false)
		log.Println(string(buf))
	}
}
