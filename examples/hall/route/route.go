package route

import (
	"fmt"
	"frbg/codec"
	"frbg/def"
	"frbg/examples/cmd"
	"frbg/examples/db"
	"frbg/examples/hall/slots"
	"frbg/examples/proto"
	"frbg/local"
	"frbg/network"
	"log"
)

var route *Local

type Local struct {
	*local.BaseLocal
}

func New(st *network.ServerConfig) *Local {
	route = &Local{
		BaseLocal: local.NewBase(st),
	}
	route.init()
	return route
}

func (l *Local) init() {
	l.BaseLocal.Init()
	l.AddRoute(cmd.GetGameList, l.getGameList)
	l.AddRoute(cmd.GetRoomList, l.getRoomList)
	l.AddRoute(cmd.EnterSlots, l.enterSlots)
	l.AddRoute(cmd.SpinSlots, l.spinSlots)
	l.AddRoute(cmd.LeaveSlots, l.leaveSlots)
	l.AddRoute(cmd.Test, l.test)
}

func (l *Local) offline(c *network.Conn, msg *codec.Message) error {
	return nil
}

func (l *Local) test(conn *network.Conn, msg *codec.Message) error {
	data := new(proto.Test)
	if err := msg.Unpack(data); err != nil {
		return err
	}

	b, _ := codec.Pack(msg.UserID, def.ST_User, msg.Cmd, &proto.Test{
		Uid:       data.Uid,
		StartTime: data.StartTime,
	})

	_, err := conn.Write(b)
	return err
}

func (l *Local) getGameList(c *network.Conn, msg *codec.Message) error {
	log.Println("getGameList")
	req, rsp := new(proto.GetGameListReq), new(proto.GetGameListRsp)
	if err := msg.Unpack(req); err != nil {
		log.Printf("getGameList msg.Unpack() err:%s", err.Error())
		return err
	}
	rsp.Games = db.GetGameList()
	if buf, err := codec.Pack(msg.UserID, def.ST_User, msg.Cmd, rsp); err == nil {
		if errSend := l.SendToGate(msg.GateID, buf); errSend != nil {
			log.Printf("SendToGate(%d) err:%s", msg.GateID, errSend.Error())
		}
	}
	return nil
}

func (l *Local) getRoomList(c *network.Conn, msg *codec.Message) error {
	log.Println("getRoomList")
	req, rsp := new(proto.GetRoomListReq), new(proto.GetRoomListRsp)
	if err := msg.Unpack(req); err != nil {
		return err
	}
	rsp.Rooms = db.GetRoomList(req.GameId)
	if buf, err := codec.Pack(msg.UserID, def.ST_User, msg.Cmd, rsp); err == nil {
		l.SendToGate(msg.GateID, buf)
		// c.Write(buf)
	}
	return nil
}

// 请求进入老虎机
func (l *Local) enterSlots(c *network.Conn, msg *codec.Message) error {
	req := new(proto.EnterSlotsReq)
	if err := msg.Unpack(req); err != nil {
		return err
	}
	conf := slots.GetSlotsData(msg.UserID, req.GameId)
	if conf == nil {
		return nil
	}
	rsp := &proto.EnterSlotsRsp{
		GameId: req.GameId,
		Bet:    conf.BetConf.Bet,
		Level:  conf.BetConf.Level,
		Line:   conf.BetConf.Lines,
		Lines:  conf.RouteConf,
		Elems:  conf.ElemConf,
	}
	if buf, err := codec.Pack(msg.UserID, def.ST_User, msg.Cmd, rsp); err == nil {
		// c.Write(buf)
		l.SendToGate(msg.GateID, buf)
	}
	return nil
}

// 老虎机请求摇奖
func (l *Local) spinSlots(c *network.Conn, msg *codec.Message) error {
	req := new(proto.SlotsSpinReq)
	if err := msg.Unpack(req); err != nil {
		return err
	}

	slotsData := slots.GetSlotsData(msg.UserID, req.GameId)
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
	if buf, err := codec.Pack(msg.UserID, def.ST_User, msg.Cmd, rsp); err == nil {
		// c.Write(buf)
		l.SendToGate(msg.GateID, buf)
	}
	return nil
}

// 离开老虎机
func (l *Local) leaveSlots(c *network.Conn, msg *codec.Message) error {
	req := new(proto.LeaveSlotsReq)
	if err := msg.Unpack(req); err != nil {
		return err
	}
	slots.DelSlotsData(msg.UserID)
	return nil
}

func (l *Local) Close(conn *network.Conn) {
	l.BaseLocal.Close(conn)
	if conn.ServerConfig == nil {
		return
	}
}
