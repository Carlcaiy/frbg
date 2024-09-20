package route

import (
	"fmt"
	"frbg/def"
	"frbg/examples/cmd"
	"frbg/examples/db"
	"frbg/examples/proto"
	"frbg/local"
	"frbg/network"
	"frbg/parser"
	"log"
)

type Local struct {
	*local.BaseLocal
}

func New(st *network.ServerConfig) *Local {
	l := &Local{
		BaseLocal: local.NewBase(st),
	}
	l.Init()
	return l
}

func (l *Local) Init() {
	l.BaseLocal.Init()
	l.AddRoute(cmd.Login, l.login)
	l.AddRoute(cmd.MultiBroadcast, l.multiBroadcast)
	l.AddRoute(cmd.Logout, l.logout)
	l.AddHook(cmd.SyncData, l.sync)
}

func (l *Local) login(c *network.Conn, msg *parser.Message) error {
	req := new(proto.LoginReq)
	if err := msg.Unpack(req); err != nil {
		return err
	}
	user := new(User)
	err := db.GetUser(msg.UserID, user)
	if err == nil {
		if c != user.Conn {
			buf, _ := parser.Pack(msg.UserID, def.ST_User, cmd.GateKick, &proto.GateKick{
				Type: proto.KickType_Squeeze,
			})
			user.Write(buf)
			user.Conn = c
			db.SetGate(msg.UserID, l.ServerId)
		} else {
			buf, _ := parser.Pack(msg.UserID, def.ST_User, cmd.Login, &proto.LoginRsp{
				Ret: 1,
			})
			c.Write(buf)
			db.SetGate(msg.UserID, l.ServerId)
		}
	} else {
		u := &User{
			uid:  msg.UserID,
			Conn: c,
		}
		c.SetContext(u)
		l.AddUser(u)
		db.SetGate(msg.UserID, l.ServerId)
	}

	if gid := db.GetGame(msg.UserID); gid > 0 {
		buf, _ := parser.Pack(msg.UserID, def.ST_Game, cmd.Reconnect, &proto.Reconnect{})
		l.SendToGame(gid, buf)
	}
	return nil
}

func (l *Local) multiBroadcast(c *network.Conn, msg *parser.Message) error {
	req := new(proto.MultiBroadcast)
	if err := msg.Unpack(req); err != nil {
		return err
	}
	for _, uid := range req.Uids {
		if user := l.GetUser(uid); user != nil {
			user.Write(req.Data)
		}
	}
	return nil
}

// 记录游戏服务ID
func (l *Local) sync(c *network.Conn, msg *parser.Message) error {
	req := new(proto.SyncData)
	if err := msg.Unpack(req); err != nil {
		return err
	}
	log.Printf("sync userId:%d gameId:%d roomId:%d\n", msg.UserID, req.GameId, req.RoomId)
	if user, ok := l.GetUser(msg.UserID).(*User); ok && user != nil {
		user.gameId = uint8(req.GameId)
	}
	return nil
}

// 离开网关
func (l *Local) logout(c *network.Conn, msg *parser.Message) error {
	u, ok := c.Context().(*User)
	if ok {
		b, _ := parser.Pack(u.UserID(), def.ST_User, msg.Cmd, &proto.CommonRsp{
			Code: proto.ErrorCode_Success,
		})
		l.SendToUser(u.UserID(), b)
		l.DelUser(u.UserID())
		db.SetGate(msg.UserID, 0)
		return nil
	}
	return fmt.Errorf("reqLeaveGate not find user: %d", c.Fd)
}

func (l *Local) Close(conn *network.Conn) {
	l.BaseLocal.Close(conn)
	// 1、游戏服，踢出游戏内所有玩家
	if conn.ServerConfig == nil {
		u, ok := conn.Context().(*User)
		if ok {
			if u.gameId > 0 {
				bs, _ := parser.Pack(u.UserID(), def.ST_Game, cmd.Offline, &proto.Offline{})
				l.SendToSid(u.gameId, bs, def.ST_Game)
			} else if u.hallId > 0 {
				bs, _ := parser.Pack(u.UserID(), def.ST_Hall, cmd.Offline, &proto.Offline{})
				l.SendToSid(u.hallId, bs, def.ST_Hall)
			}
		}
	} else if conn.ServerType == def.ST_Game {
		l.RangeUser(func(u local.IUser) {
			if u.GameID() == conn.ServerId {
				b, _ := parser.Pack(u.UserID(), def.ST_User, cmd.GateKick, &proto.GateKick{
					Type: proto.KickType_GameNotFound,
				})
				l.SendToUser(u.UserID(), b)
			}
		})
	} else if conn.ServerType == def.ST_User {
		u, ok := conn.Context().(*User)
		if ok {
			if u.gameId > 0 {
				b, _ := parser.Pack(u.UserID(), def.ST_Game, cmd.Offline, &proto.Offline{})
				l.SendToSid(u.gameId, b, def.ST_Game)
			} else if u.hallId > 0 {
				b, _ := parser.Pack(u.UserID(), def.ST_Hall, cmd.Offline, &proto.Offline{})
				l.SendToSid(u.hallId, b, def.ST_Hall)
			}
		}
	}
}
