package route

import (
	"fmt"
	"frbg/codec"
	"frbg/def"
	"frbg/examples/cmd"
	"frbg/examples/db"
	"frbg/examples/proto"
	"frbg/local"
	"frbg/network"
	"log"
)

type Local struct {
	*local.BaseLocal
	m_client map[uint32]*network.Conn
}

func New(st *network.ServerConfig) *Local {
	l := &Local{
		BaseLocal: local.NewBase(st),
		m_client:  make(map[uint32]*network.Conn),
	}
	l.Init()
	return l
}

func (l *Local) Init() {
	l.BaseLocal.Init()
	l.AddRoute(cmd.Login, l.login)
	l.AddRoute(cmd.MultiBC, l.multibc)
	l.AddRoute(cmd.Logout, l.logout)
}

func (l *Local) Route(conn *network.Conn, msg *codec.Message) error {
	log.Println(msg, l.ServerType)
	if msg.UserID == 0 && msg.Cmd != cmd.HeartBeat && msg.Cmd != cmd.Login {
		return fmt.Errorf("msg wrong")
	}

	switch msg.DestST {
	// 优先调用与本服务
	case def.ST_WsGate, def.ST_Gate:
		return l.BaseLocal.Route(conn, msg)
	case def.ST_User:
		return l.SendToUser(msg.UserID, msg.Bytes())
	case def.ST_Hall:
		return l.SendToHall(msg.UserID, msg.Bytes())
	case def.ST_Game:
		return l.SendToGame(msg.GameID, msg.Bytes())
	}
	return fmt.Errorf("without cmd %d route", msg.Cmd)
}

func (l *Local) login(c *network.Conn, msg *codec.Message) error {
	req := new(proto.LoginReq)
	if err := msg.Unpack(req); err != nil {
		return err
	}
	log.Println(req)
	var info *User
	if msg.UserID == 0 {
		uid, err := db.GenUserId()
		if err != nil {
			bs, _ := codec.Pack(msg.UserID, def.ST_User, cmd.Login, &proto.LoginRsp{
				Ret: 1,
			})
			codec.WsWrite(c, bs)
			return err
		}
		info = &User{
			Nick:   "Beautify",
			Sex:    0,
			IconId: 1,
			Uid:    uid,
		}
		if err := db.SetUser(uid, info); err != nil {
			return err
		}
		l.SetConn(uid, c)
	} else {
		if conn := l.GetConn(msg.UserID); conn != c {
			if conn != nil {
				log.Println("给已经登录的连接推送挤号信息")
				buf, _ := codec.Pack(msg.UserID, def.ST_User, cmd.GateKick, &proto.GateKick{
					Type: proto.KickType_Squeeze,
				})
				codec.WsWrite(conn, buf)
			}
			l.SetConn(msg.UserID, c)
			if err := db.SetGate(msg.UserID, l.ServerId); err != nil {
				log.Printf("db.SetGate(%d, %d) error:%s", msg.UserID, l.ServerId, err.Error())
				return nil
			}
		} else {
			log.Println("连接相同不做处理")
		}
		info = new(User)
		if err := db.GetUser(msg.UserID, info); err != nil {
			log.Println(err)
			return err
		}
	}
	bs, _ := codec.Pack(msg.UserID, def.ST_User, msg.Cmd, &proto.LoginRsp{
		Ret:      0,
		Nick:     info.Nick,
		Uid:      info.Uid,
		IconId:   uint32(info.IconId),
		IsRegist: msg.UserID == 0,
		GameId:   uint32(db.GetGame(msg.UserID)),
	})

	// if gid := db.GetGame(msg.UserID); gid > 0 {
	// 	buf, _ := codec.Pack(msg.UserID, def.ST_Game, cmd.Reconnect, &proto.Reconnect{})
	// 	l.SendToGame(gid, buf)
	// }
	return codec.WsWrite(c, bs)
}

func (l *Local) multibc(c *network.Conn, msg *codec.Message) error {
	req := new(proto.MultiBroadcast)
	if err := msg.Unpack(req); err != nil {
		return err
	}
	for _, uid := range req.Uids {
		if user := l.GetConn(uid); user != nil {
			user.Write(req.Data)
		}
	}
	return nil
}

// 离开网关
func (l *Local) logout(c *network.Conn, msg *codec.Message) error {
	u, ok := c.Context().(*User)
	if ok {
		b, _ := codec.Pack(u.UserID(), def.ST_User, msg.Cmd, &proto.CommonRsp{
			Code: proto.ErrorCode_Success,
		})
		l.SendToUser(u.UserID(), b)
		l.DelConn(u.UserID())
		db.SetGate(msg.UserID, 0)
		return nil
	}
	return fmt.Errorf("reqLeaveGate not find user: %d", c.Fd)
}

func (l *Local) Close(conn *network.Conn) {
	l.BaseLocal.Close(conn)
	if conn.Uid > 0 {
		l.DelConn(conn.Uid)
	}
	if conn.ServerConfig == nil {
		return
	}
	if conn.ServerType == def.ST_Game {
		// todo 给在此游戏服内的玩家推送消息
	} else if conn.ServerType == def.ST_User {
		u, ok := conn.Context().(*User)
		if ok {
			if u.GameId > 0 {
				b, _ := codec.Pack(u.UserID(), def.ST_Game, cmd.Offline, &proto.Offline{})
				l.SendToSid(u.GameId, b, def.ST_Game)
			}
		}
	}
}

func (l *Local) GetConn(uid uint32) *network.Conn {
	if u, ok := l.m_client[uid]; ok {
		return u
	}
	log.Printf("get conn:%d failed", uid)
	return nil
}

func (l *Local) SetConn(uid uint32, conn *network.Conn) {
	l.m_client[uid] = conn
	conn.Uid = uid
	log.Printf("add conn:%d", uid)
}

func (l *Local) DelConn(uid uint32) {
	delete(l.m_client, uid)
}

// attention: gateway use this function, other server should be careful
func (l *Local) SendToUser(uid uint32, buf []byte) error {
	if u, ok := l.m_client[uid]; ok {
		return codec.WsWrite(u, buf)
	} else {
		return fmt.Errorf("user:%d not find", uid)
	}
}
