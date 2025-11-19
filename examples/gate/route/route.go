package route

import (
	"fmt"
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
	clients *Clients
}

func New() *Local {
	l := &Local{
		BaseLocal: local.NewBase(),
		clients:   NewClients(),
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

func (l *Local) Route(msg *network.Message) error {
	switch msg.DestType {
	// 优先调用与本服务
	case def.ST_WsGate, def.ST_Gate:
		return l.BaseLocal.Route(msg)
	// 大厅多个服务
	case def.ST_Hall:
		return l.Send(msg.Message)
	// 游戏单个服务
	case def.ST_Game:
		return l.Send(msg.Message)
	}

	return fmt.Errorf("without cmd %d route", msg.Cmd)
}

func (l *Local) login(msg *network.Message) error {
	req := new(proto.LoginReq)
	if err := msg.Unpack(req); err != nil {
		log.Printf("login unpack error:%s", err.Error())
		return err
	}
	log.Println(req)
	var info *User
	if req.Uid == 0 {
		uid, err := db.GenUserId()
		if err != nil {
			log.Printf("GenUserId err:%s", err.Error())
			return msg.Response(def.ST_User, cmd.Login, &proto.LoginRsp{
				Ret: 1,
			})
		}
		info = &User{
			Nick:   "Beautify",
			Sex:    0,
			IconId: 1,
			Uid:    uid,
			conn:   msg.GetClient(),
		}
		if err := db.SetUser(uid, info); err != nil {
			return err
		}
		l.clients.SetClient(uid, msg.GetClient())
	} else {
		if conn := l.clients.GetClient(req.Uid); conn != msg.GetClient() {
			if conn != nil {
				log.Println("给已经登录的连接推送挤号信息")
				conn.Send(def.ST_User, cmd.GateKick, &proto.GateKick{
					Type: proto.KickType_Squeeze,
				})
			}
			l.clients.SetClient(req.Uid, msg.GetClient())
			if err := db.SetGate(req.Uid, msg.DestId); err != nil {
				log.Printf("db.SetGate(%d, %d) error:%s", req.Uid, msg.DestId, err.Error())
				return nil
			}
		} else {
			log.Println("连接相同不做处理")
		}
		info = new(User)
		if err := db.GetUser(req.Uid, info); err != nil {
			log.Println(err)
			return err
		}
	}
	log.Printf("login uid:%d, gate:%d", info.Uid, msg.DestId)
	return msg.Response(0, msg.Cmd, &proto.LoginRsp{
		Ret:      0,
		Nick:     info.Nick,
		Uid:      info.Uid,
		IconId:   uint32(info.IconId),
		IsRegist: req.Uid == 0,
		GameId:   uint32(db.GetGame(req.Uid)),
	})
}

func (l *Local) multibc(msg *network.Message) error {
	req := new(proto.MultiBroadcast)
	if err := msg.Unpack(req); err != nil {
		return err
	}
	for _, uid := range req.Uids {
		if client := l.clients.GetClient(uid); client != nil {
			client.Write(msg.Message)
		}
	}
	return nil
}

// 离开网关
func (l *Local) logout(msg *network.Message) error {
	req := new(proto.LogoutReq)
	if err := msg.Unpack(req); err != nil {
		return err
	}
	client := l.clients.GetClient(req.Uid)
	if client != nil {
		msg.GetClient().Send(def.ST_User, msg.Cmd, &proto.CommonRsp{
			Code: proto.ErrorCode_Success,
		})
		l.clients.DelClient(req.Uid)
		return nil
	}
	return fmt.Errorf("reqLeaveGate not find user: %d", req.Uid)
}

func (l *Local) Close(conn *network.Conn) {
	l.clients.DelClient(conn.Uid)
}
