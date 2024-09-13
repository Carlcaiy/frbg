package local

import (
	"fmt"
	"frbg/def"
	"frbg/examples/cmd"
	"frbg/examples/proto"
	"frbg/network"
	"frbg/parser"
	"log"
	"net"
	"time"
)

type Handle func(*network.Conn, *parser.Message) error

type IUser interface {
	UserID() uint32
	GameID() uint32
	GateID() uint32
	net.Conn
}

type BaseLocal struct {
	*network.ServerConfig
	m_users   map[uint32]IUser
	m_servers map[uint8][]*network.Conn
	m_route   map[uint16]Handle // 路由
	m_hook    map[uint16]Handle // 钩子路由
	*Timer
}

func NewBase(sconf *network.ServerConfig) *BaseLocal {
	return &BaseLocal{
		ServerConfig: sconf,
		m_users:      make(map[uint32]IUser),
		m_servers:    make(map[uint8][]*network.Conn),
		m_route:      make(map[uint16]Handle),
		Timer:        NewTimer(1024),
		m_hook:       make(map[uint16]Handle),
	}
}

func (l *BaseLocal) Init() {
	log.Println("base.Init")
	l.AddRoute(cmd.HeartBeat, l.HeartBeat)
	l.AddRoute(cmd.Regist, l.Regist)
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
	for t, m_servers := range l.m_servers {
		for _, s := range m_servers {
			bs := parser.NewMessage(s.ServerId, t, cmd.HeartBeat, 1, &proto.HeartBeat{
				ServerType: uint32(t),
				ServerId:   s.ServerId,
			}).Pack()
			// log.Printf("send heart beat addr:%s type:%s id:%d\n", s.Addr, s.ServerType, s.ServerId)
			s.Write(bs)
		}
	}
}

func (l *BaseLocal) OnConnect(conn *network.Conn) {
	if sli, ok := l.m_servers[conn.ServerType]; ok {
		// 更新server
		for i := range sli {
			if sli[i].ServerId == conn.ServerId {
				log.Printf("AddConn origin:%+v new:%+v\n", sli[i], conn.ServerConfig)
				sli[i] = conn
				return
			}
		}
	}
	// 尾部加入server
	log.Printf("AddConn new:%+v\n", conn.ServerConfig)
	l.m_servers[conn.ServerType] = append(l.m_servers[conn.ServerType], conn)
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

	if sli, ok := l.m_servers[conn.ServerType]; ok {
		index := -1
		for i := range sli {
			// 找到对应的元素，移到最后
			if sli[i].ServerId == conn.ServerId {
				log.Printf("DelConn index:%d new:%+v\n", i, conn)
				sli[i] = nil
				index = i
			} else if index >= 0 {
				sli[i], sli[index] = sli[index], sli[i]
				index = i
			}
		}
		if index >= 0 {
			// 长度-1
			l.m_servers[conn.ServerType] = sli[:index]
		}
	}
}

func (l *BaseLocal) RangeUser(iter func(u IUser)) {
	for _, u := range l.m_users {
		iter(u)
	}
}

func (l *BaseLocal) Regist(conn *network.Conn, msg *parser.Message) error {
	data := new(proto.Regist)
	msg.Unpack(data)
	st := uint8(data.ServerType)
	if sli, ok := l.m_servers[st]; ok {
		for i := range sli {
			if sli[i].ServerConfig != nil && sli[i].ServerId == data.ServerId {
				return fmt.Errorf("re register config: %+v", sli[i])
			}
		}
	}
	conf := &network.ServerConfig{
		Addr:       conn.RemoteAddr().String(),
		ServerType: st,
		ServerId:   data.ServerId,
	}
	conn.ServerConfig = conf
	l.m_servers[st] = append(l.m_servers[st], conn)
	log.Printf("regist serverId:%d serverType:%s addr:%s\n", conf.ServerId, conf.ServerType, conf.Addr)
	return nil
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

func (l *BaseLocal) SendModUid(uid uint32, buf []byte, t uint8) error {
	if conns, ok := l.m_servers[t]; ok {
		if len(conns) > 0 {
			conn := conns[uid%uint32(len(conns))]
			conn.Write(buf)
			return nil
		} else {
			return fmt.Errorf("server %s size = 0", t)
		}
	} else {
		return fmt.Errorf("error not find server %s", t)
	}
}

func (l *BaseLocal) SendToSid(serverId uint32, buf []byte, t uint8) error {
	if servers, ok := l.m_servers[t]; ok {
		for i := range servers {
			if servers[i].ServerId == serverId {
				servers[i].Write(buf)
				return nil
			}
		}
	}
	return fmt.Errorf("SendToSid: serverType=%s serverId=%d not found", t, serverId)
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

func (l *BaseLocal) SendToGame(gameId uint32, buf []byte) error {
	if conns, ok := l.m_servers[def.ST_Game]; ok {
		for _, s := range conns {
			if s.ServerId == gameId {
				s.Write(buf)
				return nil
			}
		}
		return fmt.Errorf("game server %d not find", gameId)
	} else {
		return fmt.Errorf("error not find game server")
	}
}

func (l *BaseLocal) SendToGate(gateId uint32, buf []byte) error {
	if conns, ok := l.m_servers[def.ST_Gate]; ok {
		for _, s := range conns {
			if s.ServerId == gateId {
				s.Write(buf)
				return nil
			}
		}
		return fmt.Errorf("game server %d not find", gateId)
	} else {
		return fmt.Errorf("error not find gate server")
	}
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
