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
	return &Local{
		BaseLocal: local.NewBase(st),
	}
}

func (l *Local) Init() {
	l.BaseLocal.Init()
	l.AddRoute(cmd.ReqGateLogin, l.reqGateLogin)
	l.AddRoute(cmd.GateMulti, l.multi)
	l.AddRoute(cmd.ReqGateLeave, l.reqLeaveGate)
	l.AddHook(cmd.SyncData, l.sync)
}

func (l *Local) reqGateLogin(c *network.Conn, msg *parser.Message) error {
	data := new(proto.ReqGateLogin)
	msg.UnPack(data)
	user, ok := l.GetUser(msg.UserID()).(*User)
	if ok {
		if c != user.Conn {
			buf, _ := parser.Pack(msg.UserID(), def.ST_User, cmd.GateKick, &proto.GateKick{
				Type: proto.KickType_Squeeze,
			})
			user.Write(buf)
			user.Conn = c
			db.SetGate(msg.UserID(), l.ServerId)
		} else {
			buf, _ := parser.Pack(msg.UserID(), def.ST_User, cmd.ResGateLogin, &proto.ResGateLogin{
				Ret: 1,
			})
			c.Write(buf)
			db.SetGate(msg.UserID(), l.ServerId)
		}
	} else {
		u := &User{
			uid:  msg.UserID(),
			Conn: c,
		}
		c.SetContext(u)
		l.AddUser(u)
		db.SetGate(msg.UserID(), l.ServerId)
	}

	if gid := db.GetGame(msg.UserID()); gid > 0 {
		buf, _ := parser.Pack(msg.UserID(), def.ST_Game, cmd.Reconnect, &proto.Reconnect{})
		l.SendToGame(gid, buf)
	}
	return nil
}

func (l *Local) multi(c *network.Conn, msg *parser.Message) error {
	pack := new(proto.MultiMsg)
	msg.UnPack(pack)
	for _, uid := range pack.Uids {
		if user := l.GetUser(uid); user != nil {
			user.Write(pack.Data)
		}
	}
	return nil
}

// 记录游戏服务ID
func (l *Local) sync(c *network.Conn, msg *parser.Message) error {
	pack := new(proto.SyncData)
	msg.UnPack(pack)
	log.Printf("sync userId:%d gameId:%d roomId:%d\n", msg.UserID(), pack.GameId, pack.RoomId)
	if user, ok := l.GetUser(msg.UserID()).(*User); ok && user != nil {
		user.gameId = pack.GameId
	}
	return nil
}

// 离开网关
func (l *Local) reqLeaveGate(c *network.Conn, msg *parser.Message) error {
	u, ok := c.Context().(*User)
	if ok {
		b, _ := parser.Pack(u.UserID(), def.ST_User, cmd.ResGateLeave, &proto.Empty{})
		l.SendToUser(u.UserID(), b)
		l.DelUser(u.UserID())
		db.SetGate(msg.UserID(), 0)
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
