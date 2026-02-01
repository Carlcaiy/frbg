package route

import (
	"frbg/core"
	"frbg/def"
	"frbg/examples/db"
	"frbg/examples/hall/slots"
	"frbg/examples/pb"
	"frbg/local"
	"log"
	"sync/atomic"
)

var incRoomId atomic.Uint32
var route *Local

type User struct {
	Uid    uint32
	GateId uint16
	GameId uint32
	Multi  uint32
	RoomId uint32
}

type Local struct {
	*local.BaseLocal
	userGame map[uint32]*User // 用户状态 0:等待进入房间 >0:在房间内
}

func New() *Local {
	route = &Local{
		BaseLocal: local.NewBase(),
		userGame:  make(map[uint32]*User),
	}
	route.init()
	return route
}

func (l *Local) init() {
	l.Start()
	l.AddRoute(def.Offline, l.offline)
	l.AddRoute(def.GetGameList, l.getGameList)
	l.AddRoute(def.GetRoomList, l.getRoomList)
	l.AddRoute(def.EnterRoom, l.enterRoom)
	l.AddRoute(def.EnterSlots, l.enterSlots)
	l.AddRoute(def.SpinSlots, l.spinSlots)
	l.AddRoute(def.LeaveSlots, l.leaveSlots)
}

func (l *Local) offline(in *local.Input) {
}

func (l *Local) getGameList(in *local.Input) {
	log.Println("getGameList")
	req, rsp := new(pb.GetGameListReq), new(pb.GetGameListRsp)
	if err := in.Unpack(req); err != nil {
		log.Printf("getGameList msg.Unpack() err:%s", err.Error())
		return
	}
	log.Printf("getGameList req:%v", req.String())
	rsp.Games = db.GetGameList()
	if errSend := l.SendTo(core.Svid(def.ST_Gate, uint8(req.GateId)), req.Uid, in.Cmd, rsp); errSend != nil {
		log.Printf("SendTo() err:%v", errSend.Error())
	}
}

func (l *Local) getRoomList(in *local.Input) {
	log.Println("getRoomList")
	req, rsp := new(pb.GetRoomListReq), new(pb.GetRoomListRsp)
	if err := in.Unpack(req); err != nil {
		log.Printf("getRoomList msg.Unpack() err:%s", err.Error())
		return
	}
	log.Printf("getRoomList req:%v", req.String())
	rsp.Rooms = db.GetRoomList(req.GameId)
	// 发送给客户端
	if errSend := l.SendTo(core.Svid(def.ST_Gate, uint8(req.GateId)), req.Uid, in.Cmd, rsp); errSend != nil {
		log.Printf("Send() err:%s", errSend.Error())
	}
}

// 如果只是配桌，同时活跃的用户不会很多，全部写在内存也不会占用多少
// 考虑到服务器重启，只需要在服务器间同步配桌数据
// 如果出现服务器挂掉的情况，需要重新配桌，这个数据又不重要，所以不做处理
// 如果用户已经在房间内，直接返回
func (l *Local) enterRoom(in *local.Input) {
	req, rsp := new(pb.EnterRoomReq), new(pb.EnterRoomRsp)
	if err := in.Unpack(req); err != nil {
		log.Printf("enterRoom msg.Unpack() err:%s", err.Error())
		return
	}
	log.Println("enterRoom", req.String())
	svid := core.Svid(def.ST_Game, uint8(req.GameId))

	// 如果用户已经在房间内，直接返回
	if game := l.userGame[req.Uid]; game != nil && game.RoomId > 0 {
		reconnect := pb.Reconnect{
			Uid:    req.Uid,
			GateId: uint32(req.GateId),
			RoomId: game.RoomId,
		}
		l.Send(svid, def.Reconnect, &reconnect)
		return
	}

	// 查询用户状态
	greq, grsp := &pb.GameStatusReq{
		Uid: req.Uid,
	}, new(pb.GameStatusRsp)
	if err := l.RpcCall(svid, def.GameStatus, greq, grsp); err != nil {
		log.Printf("GameStatus uid:%d err:%s", req.Uid, err.Error())
		return
	}
	log.Printf("GameStatus uid:%d rsp:%v", req.Uid, grsp.String())

	// 如果用户已经在房间内，直接返回
	if grsp.RoomId != 0 {
		rsp.RoomId = grsp.RoomId
		if game := l.userGame[req.Uid]; game == nil {
			game = &User{
				Uid:    req.Uid,
				GateId: uint16(req.GateId),
				GameId: req.GameId,
				RoomId: req.RoomId,
			}
			l.userGame[req.Uid] = game
		} else {
			game.RoomId = grsp.RoomId
		}
		reconnect := pb.Reconnect{
			Uid:    req.Uid,
			GateId: uint32(req.GateId),
		}
		l.Send(svid, def.Reconnect, &reconnect)
		return
	}

	l.userGame[req.Uid] = &User{
		Uid:    req.Uid,
		GateId: uint16(req.GateId),
		GameId: req.GameId,
		Multi:  req.Multi,
	}

	matchUid := []uint32{req.Uid}
	matchUser := map[uint32]uint32{req.Uid: req.GateId}
	for uid, user := range l.userGame {
		if uid != req.Uid && user.GameId == req.GameId && user.Multi == req.Multi && user.RoomId == 0 {
			matchUid = append(matchUid, uid)
			matchUser[user.Uid] = uint32(user.GateId)
			if len(matchUid) == 4 {
				break
			}
		}
	}

	// 4人配桌
	if len(matchUid) < 4 {
		log.Printf("enterRoom matchUid:%v", matchUid)
		return
	}

	// 配桌信息
	rsp.RoomId = incRoomId.Add(1)
	for _, uid := range matchUid {
		l.userGame[uid].RoomId = rsp.RoomId
	}

	// 通知游戏预约房间
	log.Printf("StartGame svid:%d startGameReq:%s", svid, &pb.StartGameReq{
		RoomId: rsp.RoomId,
		Users:  matchUser,
	})
	l.Send(svid, def.StartGame, &pb.StartGameReq{
		RoomId: rsp.RoomId,
		Users:  matchUser,
	})
}

// 请求进入老虎机
func (l *Local) enterSlots(in *local.Input) {
	req := new(pb.EnterSlotsReq)
	if err := in.Unpack(req); err != nil {
		log.Printf("enterSlots msg.Unpack() err:%s", err.Error())
		return
	}
	conf := slots.GetSlotsData(req.Uid, req.GameId)
	if conf == nil {
		return
	}
	rsp := &pb.EnterSlotsRsp{
		GameId: req.GameId,
		Bet:    conf.BetConf.Bet,
		Level:  conf.BetConf.Level,
		Line:   conf.BetConf.Lines,
		Lines:  conf.RouteConf,
		Elems:  conf.ElemConf,
	}
	svid := core.Svid(def.ST_Gate, uint8(req.GateId))
	if errSend := l.SendTo(svid, req.Uid, def.EnterSlots, rsp); errSend != nil {
		log.Printf("Response() err:%s", errSend.Error())
	}
}

// 老虎机请求摇奖
func (l *Local) spinSlots(in *local.Input) {
	req := new(pb.SlotsSpinReq)
	if err := in.Unpack(req); err != nil {
		log.Printf("spinSlots msg.Unpack() err:%s", err.Error())
		return
	}

	slotsData := slots.GetSlotsData(req.Uid, req.GameId)
	if slotsData == nil {
		log.Printf("spinSlots sltos %d not find", req.GameId)
		return
	}
	if !slotsData.BetConf.Valid(req.Bet, req.Level) {
		log.Printf("spinSlots sltos %d: bet:%d level:%d invalid", req.GameId, req.Bet, req.Level)
		return
	}

	rsp, err := slotsData.Spin(int64(req.Bet) * int64(req.Level))
	if err != nil {
		log.Printf("spinSlots sltos %d Spin() err:%s", req.GameId, err.Error())
		return
	}
	svid := core.Svid(def.ST_Gate, uint8(req.GateId))
	l.SendTo(svid, req.Uid, def.LeaveSlots, rsp)
}

// 离开老虎机
func (l *Local) leaveSlots(in *local.Input) {
	req := new(pb.LeaveSlotsReq)
	if err := in.Unpack(req); err != nil {
		log.Printf("leaveSlots msg.Unpack() err:%s", err.Error())
		return
	}
	slots.DelSlotsData(req.Uid)
}

func (l *Local) Close(conn core.IConn) {
	l.BaseLocal.Close(conn)
}
