package route

import (
	"frbg/codec"
	"frbg/core"
	"frbg/def"
	"frbg/examples/db"
	"frbg/examples/pb"
	"frbg/local"
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
	l.Start()
	l.AddRoute(def.Echo, l.echo)
	l.AddRoute(def.Login, l.login)
	l.AddRoute(def.Logout, l.logout)
	l.AddRoute(def.PacketIn, l.packetIn)
	l.AddRoute(def.PacketOut, l.packetOut)
}

func (l *Local) echo(in *local.Input) {
	test := new(pb.Test)
	in.Unpack(test)
	if errSend := in.WriteBy(in.Cmd, test); errSend != nil {
		log.Printf("echo Response() err:%s", errSend.Error())
	}
}

func (l *Local) login(in *local.Input) {
	req := new(pb.LoginReq)
	if err := in.Unpack(req); err != nil {
		log.Printf("login unpack error:%s", err.Error())
		return
	}
	log.Printf("login req:%v", req)
	var info *User
	if req.Uid == 0 {
		uid, err := db.GenUserId()
		if err != nil {
			log.Printf("GenUserId err:%s", err.Error())
			in.WriteBy(def.Login, &pb.LoginRsp{
				Ret: 1,
			})
			return
		}
		info = &User{
			Nick:   "Beautify",
			Sex:    0,
			IconId: 1,
			Uid:    uid,
		}
		if err := db.SetUser(uid, info); err != nil {
			log.Printf("SetUser err:%s", err.Error())
			in.WriteBy(def.Login, &pb.LoginRsp{
				Ret: 1,
			})
			return
		}
		l.clients.SetClient(uid, in.IConn)
	} else {
		if conn := l.clients.GetClient(req.Uid); conn != in.IConn {
			if conn != nil {
				log.Printf("给已经登录的连接推送挤号信息 uid:%d", req.Uid)
				conn.WriteBy(def.GateKick, &pb.GateKick{
					Type: pb.KickType_Squeeze,
				})
			}
			l.clients.SetClient(req.Uid, in.IConn)
			if err := db.SetGate(req.Uid, uint8(req.GateId)); err != nil {
				log.Printf("db.SetGate(%d, %d) error:%s", req.Uid, req.GateId, err.Error())
				return
			}
		} else {
			log.Println("连接相同不做处理")
		}
		info = new(User)
		if err := db.GetUser(req.Uid, info); err != nil {
			log.Printf("db.GetUser(%d) error:%s", req.Uid, err.Error())
			return
		}
	}
	log.Printf("login uid:%d, gate:%d", info.Uid, req.GateId)
	in.WriteBy(in.Cmd, &pb.LoginRsp{
		Ret:      0,
		Nick:     info.Nick,
		Uid:      info.Uid,
		IconId:   uint32(info.IconId),
		IsRegist: req.Uid == 0,
		GameId:   uint32(db.GetGame(req.Uid)),
	})
}

// 离开网关
func (l *Local) logout(in *local.Input) {
	req := new(pb.LogoutReq)
	if err := in.Unpack(req); err != nil {
		log.Printf("logout unpack error:%s", err.Error())
		return
	}
	in.WriteBy(in.Cmd, &pb.CommonRsp{
		Code: pb.ErrorCode_Success,
	})
	l.clients.DelClient(req.Uid)
	log.Printf("logout uid:%d", req.Uid)
}

func (l *Local) packetIn(in *local.Input) {
	req := new(pb.PacketIn)
	if err := in.Unpack(req); err != nil {
		log.Printf("packetIn unpack error:%s", err.Error())
		return
	}
	cli := l.Poll.GetServer(uint16(req.Svid))
	if cli == nil {
		log.Printf("not find server %d", req.Svid)
		return
	}
	// log.Printf("packetIn cmd:%d, svid:%d", req.Cmd, req.Svid)
	cli.Write(codec.NewMessageWithPayload(uint16(req.Cmd), req.Payload))
}

func (l *Local) packetOut(in *local.Input) {
	req := new(pb.PacketOut)
	if err := in.Unpack(req); err != nil {
		log.Printf("packetOut unpack error:%s", err.Error())
		return
	}
	// log.Printf("packetOut uid:%d, cmd:%d", req.Uid, req.Cmd)
	for _, uid := range req.Uid {
		cli := l.clients.GetClient(uid)
		if cli == nil {
			log.Printf("not find user %d", uid)
			continue
		}
		cli.Write(codec.NewMessageWithPayload(uint16(req.Cmd), req.Payload))
	}
}

func (l *Local) Close(conn core.IConn) {
	uid, ok := conn.Context().(uint32)
	if !ok {
		log.Printf("Close conn.Context() not uint32")
		return
	}
	log.Printf("Close uid:%d", uid)
	l.clients.DelClient(uid)
}
