package route

import (
	"fmt"
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
	l.AddRoute(def.MultiBC, l.multibc)
	l.AddRoute(def.Logout, l.logout)
	l.AddRoute(def.PacketIn, l.packetIn)
	l.AddRoute(def.PacketOut, l.packetOut)
}

func (l *Local) echo(in *local.Input) error {
	test := new(pb.Test)
	in.Unpack(test)
	return in.Response(0, in.Cmd, test)
}

func (l *Local) login(in *local.Input) error {
	req := new(pb.LoginReq)
	if err := in.Unpack(req); err != nil {
		log.Printf("login unpack error:%s", err.Error())
		return err
	}
	log.Printf("login req:%v", req)
	var info *User
	if req.Uid == 0 {
		uid, err := db.GenUserId()
		if err != nil {
			log.Printf("GenUserId err:%s", err.Error())
			return in.Response(req.Uid, def.Login, &pb.LoginRsp{
				Ret: 1,
			})
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
		l.clients.SetClient(uid, in.Client())
	} else {
		if conn := l.clients.GetClient(req.Uid); conn != in.Client() {
			if conn != nil {
				log.Println("给已经登录的连接推送挤号信息")
				conn.Write(codec.NewMessage(def.GateKick, &pb.GateKick{
					Type: pb.KickType_Squeeze,
				}))
			}
			l.clients.SetClient(req.Uid, in.Client())
			if err := db.SetGate(req.Uid, uint8(req.GateId)); err != nil {
				log.Printf("db.SetGate(%d, %d) error:%s", req.Uid, req.GateId, err.Error())
				return nil
			}
		} else {
			log.Println("连接相同不做处理")
		}
		info = new(User)
		if err := db.GetUser(req.Uid, info); err != nil {
			log.Printf("db.GetUser(%d) error:%s", req.Uid, err.Error())
			return err
		}
	}
	log.Printf("login uid:%d, gate:%d", info.Uid, req.GateId)
	return in.Response(req.Uid, in.Cmd, &pb.LoginRsp{
		Ret:      0,
		Nick:     info.Nick,
		Uid:      info.Uid,
		IconId:   uint32(info.IconId),
		IsRegist: req.Uid == 0,
		GameId:   uint32(db.GetGame(req.Uid)),
	})
}

func (l *Local) multibc(in *local.Input) error {
	req := new(pb.MultiBroadcast)
	if err := in.Unpack(req); err != nil {
		return err
	}
	for _, uid := range req.Uids {
		if client := l.clients.GetClient(uid); client != nil {
			client.Write(in.Message)
		}
	}
	return nil
}

// 离开网关
func (l *Local) logout(in *local.Input) error {
	req := new(pb.LogoutReq)
	if err := in.Unpack(req); err != nil {
		return err
	}
	client := l.clients.GetClient(req.Uid)
	if client != nil {
		client.Write(codec.NewMessage(in.Cmd, &pb.CommonRsp{
			Code: pb.ErrorCode_Success,
		}))
		l.clients.DelClient(req.Uid)
		return nil
	}
	return fmt.Errorf("reqLeaveGate not find user: %d", req.Uid)
}

func (l *Local) packetIn(in *local.Input) error {
	req := new(pb.PacketIn)
	if err := in.Unpack(req); err != nil {
		return err
	}
	cli := l.Poll.GetServer(uint16(req.Svid))
	if cli == nil {
		return fmt.Errorf("not find server %d", req.Svid)
	}
	log.Printf("packetIn cmd:%d, svid:%d", req.Cmd, req.Svid)
	data := in.Message
	data.Cmd = uint16(req.Cmd)
	data.Payload = req.Payload
	return cli.Write(data)
}

func (l *Local) packetOut(in *local.Input) error {
	req := new(pb.PacketOut)
	if err := in.Unpack(req); err != nil {
		return err
	}
	log.Printf("packetOut uid:%d, cmd:%d", req.Uid, req.Cmd)
	cli := l.clients.GetClient(req.Uid)
	if cli == nil {
		return fmt.Errorf("not find user %d", req.Uid)
	}
	in.Message.Cmd = uint16(req.Cmd)
	in.Message.Payload = req.Payload
	return cli.Write(in.Message)
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
