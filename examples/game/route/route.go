package route

import (
	"frbg/def"
	"frbg/examples/pb"
	"frbg/local"
	"log"
)

type Local struct {
	*local.BaseLocal
	rooms map[uint32]*Room // 房间配置
	users map[uint32]*Room
}

func NewLocal() *Local {
	l := &Local{
		BaseLocal: local.NewBase(),
		rooms:     make(map[uint32]*Room),
		users:     make(map[uint32]*Room),
	}
	l.init()
	return l
}

func (l *Local) init() {
	l.Start()
	l.AddRoute(def.GameStatus, l.getGameStatus)
	l.AddRoute(def.StartGame, l.startGame)
	l.AddRoute(def.LeaveRoom, l.leaveRoom)
	l.AddRoute(def.OptGame, l.optGame)
	l.AddRoute(def.Reconnect, l.reconnect)
	l.AddRoute(def.Offline, l.offline)
	l.AddRoute(def.Continue, l.continueGame)
}

func (l *Local) getGameStatus(in *local.Input) {
	req := new(pb.GameStatusReq)
	if err := in.Unpack(req); err != nil {
		log.Printf("getGameStatus unpack error:%s", err.Error())
		return
	}
	log.Println("getGameStatus", req.String())
	rsp := new(pb.GameStatusRsp)
	room, ok := l.users[req.Uid]
	if ok && room.playing {
		rsp.RoomId = room.roomId
	}
	in.Rpc(rsp)
}

func (l *Local) offline(in *local.Input) {
	req := new(pb.Offline)
	if err := in.Unpack(req); err != nil {
		log.Printf("offline unpack error:%s", err.Error())
		return
	}
	log.Println("offline", req.String())
	if room, ok := l.users[req.Uid]; ok {
		room.Offline(req.Uid)
	}
}

func (l *Local) reconnect(in *local.Input) {
	req := new(pb.Reconnect)
	if err := in.Unpack(req); err != nil {
		log.Printf("reconnect unpack error:%s", err.Error())
		return
	}
	log.Println("reconnect", req.String())
	if req.RoomId > 0 {
		log.Printf("rooms:%v roomId:%d", l.rooms, req.RoomId)
		room, ok := l.rooms[req.RoomId]
		if ok {
			room.Reconnect(req.Uid, uint16(req.GateId))
		} else {
			log.Printf("room %d not found", req.RoomId)
		}
	}
}

func (l *Local) startGame(in *local.Input) {
	req, rsp := new(pb.StartGameReq), new(pb.StartGameRsp)
	if err := in.Unpack(req); err != nil {
		log.Printf("startGame unpack error:%s", err.Error())
		return
	}
	log.Println("startGame", req.String())
	room := l.rooms[req.RoomId]
	if room == nil {
		room = NewRoom(l, req.RoomId)
		l.rooms[req.RoomId] = room
	}
	log.Printf("rooms:%v", l.rooms)
	for uid, gateId := range req.Users {
		room.AddUser(uid, uint16(gateId))
		l.users[uid] = room
	}

	rsp.RoomId = room.roomId
	rsp.Multi = req.Multi
	rsp.Users = make(map[uint32]int32)
	for uid := range req.Users {
		rsp.Users[uid] = int32(room.GetUserByUID(uid).Seat())
	}
	for uid := range req.Users {
		room.GetUserByUID(uid).Send(def.StartGame, rsp)
	}

	room.MajFaPai()
}

func (l *Local) leaveRoom(in *local.Input) {
	req, rsp := new(pb.LeaveRoomReq), new(pb.LeaveRoomRsp)
	if err := in.Unpack(req); err != nil {
		log.Printf("leaveRoom unpack error:%s", err.Error())
		return
	}
	log.Println("leaveRoom", req.String())
	room, ok := l.rooms[req.RoomId]
	if !ok {
		return
	}
	if room.playing {
		return
	}
	room.DelUser(req.Uid)
	delete(l.users, req.Uid)
	in.WriteBy(in.Cmd, rsp)
}

func (l *Local) optGame(in *local.Input) {
	req := new(pb.MjOpt)
	if err := in.Unpack(req); err != nil {
		log.Printf("optGame unpack error:%s", err.Error())
		return
	}

	if room, ok := l.rooms[req.RoomId]; ok {
		room.MjOp(req.Uid, req)
	}
}

func (l *Local) continueGame(in *local.Input) {
	req := new(pb.Continue)
	if err := in.Unpack(req); err != nil {
		log.Printf("continueGame unpack error:%s", err.Error())
		return
	}
	log.Println("continueGame", req.String())
	room, ok := l.rooms[req.RoomId]
	if !ok {
		return
	}
	user := room.GetUserByUID(req.Uid)
	if user == nil {
		return
	}
	user.prepare = true
	for _, u := range room.Users {
		if !u.prepare {
			return
		}
	}
	room.Reset()
	room.MajFaPai()
}
