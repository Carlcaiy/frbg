package route

import (
	"fmt"
	"frbg/codec"
	"frbg/def"
	"frbg/examples/db"
	"frbg/examples/hall/slots"
	"frbg/examples/pb"
	"frbg/local"
	"frbg/network"
	"log"
)

var route *Local

type Local struct {
	*local.BaseLocal
}

func New() *Local {
	route = &Local{
		BaseLocal: local.NewBase(),
	}
	route.init()
	return route
}

func (l *Local) init() {
	l.BaseLocal.Init()
	l.AddRoute(def.Offline, l.offline)
	l.AddRoute(def.GetGameList, l.getGameList)
	l.AddRoute(def.GetRoomList, l.getRoomList)
	l.AddRoute(def.EnterSlots, l.enterSlots)
	l.AddRoute(def.SpinSlots, l.spinSlots)
	l.AddRoute(def.LeaveSlots, l.leaveSlots)
}

func (l *Local) offline(in *local.Input) error {
	return nil
}

func (l *Local) getGameList(in *local.Input) error {
	log.Println("getGameList")
	req, rsp := new(pb.GetGameListReq), new(pb.GetGameListRsp)
	if err := in.Unpack(req); err != nil {
		log.Printf("getGameList msg.Unpack() err:%s", err.Error())
		return err
	}
	rsp.Games = db.GetGameList()
	codec.NewMessage(in.Cmd, rsp)

	return in.Response(req.Uid, in.Cmd, rsp)
}

func (l *Local) getRoomList(in *local.Input) error {
	log.Println("getRoomList")
	req, rsp := new(pb.GetRoomListReq), new(pb.GetRoomListRsp)
	if err := in.Unpack(req); err != nil {
		return err
	}
	rsp.Rooms = db.GetRoomList(req.GameId)
	if errSend := in.Response(req.Uid, in.Cmd, rsp); errSend != nil {
		log.Printf("Send() err:%s", errSend.Error())
	}
	return nil
}

// 请求进入老虎机
func (l *Local) enterSlots(in *local.Input) error {
	req := new(pb.EnterSlotsReq)
	if err := in.Unpack(req); err != nil {
		return err
	}
	conf := slots.GetSlotsData(req.Uid, req.GameId)
	if conf == nil {
		return nil
	}
	rsp := &pb.EnterSlotsRsp{
		GameId: req.GameId,
		Bet:    conf.BetConf.Bet,
		Level:  conf.BetConf.Level,
		Line:   conf.BetConf.Lines,
		Lines:  conf.RouteConf,
		Elems:  conf.ElemConf,
	}
	if errSend := in.Response(req.Uid, in.Cmd, rsp); errSend != nil {
		log.Printf("Response() err:%s", errSend.Error())
	}
	return nil
}

// 老虎机请求摇奖
func (l *Local) spinSlots(in *local.Input) error {
	req := new(pb.SlotsSpinReq)
	if err := in.Unpack(req); err != nil {
		return err
	}

	slotsData := slots.GetSlotsData(req.Uid, req.GameId)
	if slotsData == nil {
		return fmt.Errorf("sltos %d not find", req.GameId)
	}
	if !slotsData.BetConf.Valid(req.Bet, req.Level) {
		return fmt.Errorf("sltos %d: bet:%d level:%d invalid", req.GameId, req.Bet, req.Level)
	}

	rsp, err := slotsData.Spin(int64(req.Bet) * int64(req.Level))
	if err != nil {
		return err
	}
	if errSend := in.Response(req.Uid, in.Cmd, rsp); errSend != nil {
		log.Printf("Response() err:%s", errSend.Error())
	}
	return nil
}

// 离开老虎机
func (l *Local) leaveSlots(in *local.Input) error {
	req := new(pb.LeaveSlotsReq)
	if err := in.Unpack(req); err != nil {
		return err
	}
	slots.DelSlotsData(req.Uid)
	return nil
}

func (l *Local) Close(conn *network.Conn) {
	l.BaseLocal.Close(conn)
}
