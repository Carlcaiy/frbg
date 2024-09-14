package local

import (
	"fmt"
	"frbg/def"
	"frbg/examples/cmd"
	"frbg/examples/proto"
	"frbg/network"
	"frbg/parser"
	"frbg/register"
	"log"
	"net"
	"time"
)

type Handle func(*network.Conn, *parser.Message) error

type IUser interface {
	UserID() uint32
	GameID() uint8
	GateID() uint8
	net.Conn
}

type BaseLocal struct {
	*network.ServerConfig
	m_users   map[uint32]IUser
	m_servers map[uint16]*network.Conn
	m_route   map[uint16]Handle // 路由
	m_hook    map[uint16]Handle // 钩子路由
	*Timer
}

func NewBase(sconf *network.ServerConfig) *BaseLocal {
	return &BaseLocal{
		ServerConfig: sconf,
		m_users:      make(map[uint32]IUser),
		m_servers:    make(map[uint16]*network.Conn),
		m_route:      make(map[uint16]Handle),
		Timer:        NewTimer(1024),
		m_hook:       make(map[uint16]Handle),
	}
}

func (l *BaseLocal) Init() {
	log.Println("base.Init")
	l.AddRoute(cmd.HeartBeat, l.HeartBeat)
	l.AddRoute(cmd.Test, l.TestRequest)
	l.StartTimer(time.Minute, l.TimerHeartBeat, true)
}

func (l *BaseLocal) StartTimer(dur time.Duration, f func(), loop bool) {
	e := &TimerEvent{
		duration:    dur,
		triggerTime: time.Now().Add(dur),
		Loop:        loop,
		event:       f,
	}
	l.Timer.Push(e)
}

func (l *BaseLocal) TimerHeartBeat() {
	for _, s := range l.m_servers {
		bs := parser.NewMessage(0, s.ServerType, cmd.HeartBeat, uint8(s.ServerId), &proto.HeartBeat{
			ServerType: uint32(s.ServerType),
			ServerId:   uint32(s.ServerId),
		}).Pack()
		// log.Printf("send heart beat addr:%s type:%s id:%d\n", s.Addr, s.ServerType, s.ServerId)
		s.Write(bs)
	}
}

// 连接成功的回调
func (l *BaseLocal) OnConnect(conn *network.Conn) {
	log.Printf("AddConn new:%+v\n", conn.ServerConfig)
	l.m_servers[conn.Svid()] = conn
}

func (l *BaseLocal) OnAccept(conn *network.Conn) {

}

func (l *BaseLocal) Close(conn *network.Conn) {
	log.Printf("DelConn:%+v\n", conn)
	// 如果是用户连接删除用户数据即可
	if conn.ServerConfig == nil {
		u, ok := conn.Context().(IUser)
		if ok {
			l.DelUser(u.UserID())
		}
		return
	}
	delete(l.m_servers, conn.Svid())
}

func (l *BaseLocal) RangeUser(iter func(u IUser)) {
	for _, u := range l.m_users {
		iter(u)
	}
}

func (l *BaseLocal) HeartBeat(conn *network.Conn, msg *parser.Message) error {
	data := new(proto.HeartBeat)
	if err := msg.Unpack(data); err != nil {
		return err
	}
	log.Println("HeartBeat", data.String())
	return nil
}

func (l *BaseLocal) TestRequest(conn *network.Conn, msg *parser.Message) error {
	data := new(proto.Test)
	if err := msg.Unpack(data); err != nil {
		return err
	}

	l.AddUser(&UserImplement{
		userId: data.Uid,
		Conn:   conn,
	})

	b, _ := msg.PackProto(&proto.Test{
		Uid:       data.Uid,
		StartTime: data.StartTime,
	})

	return l.SendToUser(data.Uid, b)
}

func (l *BaseLocal) Tick() {
	l.FrameCheck()
}

func (l *BaseLocal) AddHook(cmd uint16, h Handle) {
	if _, ok := l.m_hook[cmd]; !ok {
		l.m_hook[cmd] = h
	} else {
		log.Printf("err: hook.Add cmd:%d\n", cmd)
	}
}

func (l *BaseLocal) AddRoute(cmd uint16, h Handle) {
	if _, ok := l.m_route[cmd]; !ok {
		l.m_route[cmd] = h
	} else {
		log.Fatalf("err: handler.Add cmd:%d\n", cmd)
	}
}

// 鉴权
func (l *BaseLocal) Auth(conn *network.Conn, msg *parser.Message) error {
	if msg.UserID == 0 && msg.Cmd != cmd.Login && msg.Cmd != cmd.HeartBeat {
		return fmt.Errorf("msg wrong")
	}
	return nil
}

func (l *BaseLocal) Route(conn *network.Conn, msg *parser.Message) error {

	log.Println(msg)
	if err := l.Auth(conn, msg); err != nil {
		return err
	}

	u := l.GetUser(msg.UserID)

	// 优先调用钩子
	if hook, ok := l.m_hook[msg.Cmd]; ok {
		hook(conn, msg)
	}

	switch msg.DestST {
	// 优先调用与本服务
	case l.ServerType:
		if handle, ok := l.m_route[msg.Cmd]; ok {
			return handle(conn, msg)
		} else {
			return fmt.Errorf("call: not find cmd %d", msg.Cmd)
		}
	case def.ST_User:
		return l.SendToUser(msg.UserID, msg.Bytes())
	case def.ST_Hall:
		return l.SendToHall(msg.UserID, msg.Bytes())
	case def.ST_Game:
		return l.SendToGame(u.GameID(), msg.Bytes())
	case def.ST_Gate:
		return l.SendToGate(u.GateID(), msg.Bytes())
	}
	return fmt.Errorf("call: not find cmd %d", msg.Cmd)
}

func (l *BaseLocal) SendModUid(uid uint32, buf []byte, serverType uint8) error {
	sids := register.Gets(serverType)
	if len(sids) == 0 {
		return fmt.Errorf("error not find server %d", serverType)
	}
	serverId := sids[int(uid)%len(sids)]
	return l.SendToSid(serverId, buf, serverType)
}

// attention: gateway use this function, other server should be careful
func (l *BaseLocal) SendToUser(uid uint32, buf []byte) error {
	if l.ServerType != def.ST_Gate {
		return fmt.Errorf("only gate can do this, st：%d", l.ServerType)
	}

	if u, ok := l.m_users[uid]; ok {
		return parser.WsWrite(u, buf)
	} else {
		return fmt.Errorf("user:%d not find", uid)
	}
}

func (l *BaseLocal) SendToGame(gameId uint8, buf []byte) error {
	return l.SendToSid(gameId, buf, def.ST_Game)
}

func (l *BaseLocal) SendToGate(gateId uint8, buf []byte) error {
	return l.SendToSid(gateId, buf, def.ST_Gate)
}

func (l *BaseLocal) SendToSid(serverId uint8, buf []byte, serverType uint8) error {
	if svr := l.GetServer(serverType, serverId); svr != nil {
		svr.Write(buf)
		return nil
	}
	return fmt.Errorf("SendToSid: serverType=%d serverId=%d not found", serverType, serverId)
}

func (l *BaseLocal) GetServer(serverType uint8, serverId uint8) *network.Conn {
	key := uint16(serverType)*100 + uint16(serverId)
	if conns, ok := l.m_servers[key]; ok {
		return conns
	}
	addr := register.Get(key)
	if addr == "" {
		return nil
	}
	return network.NewClient(&network.ServerConfig{
		ServerType: serverType,
		ServerId:   serverId,
		Addr:       addr,
	})
}

func (l *BaseLocal) SendToHall(uid uint32, buf []byte) error {
	return l.SendModUid(uid, buf, def.ST_Hall)
}

func (l *BaseLocal) GetUser(uid uint32) IUser {
	if u, ok := l.m_users[uid]; ok {
		return u
	}
	return nil
}

func (l *BaseLocal) AddUser(user IUser) {
	l.m_users[user.UserID()] = user
}

func (l *BaseLocal) DelUser(uid uint32) {
	delete(l.m_users, uid)
}
