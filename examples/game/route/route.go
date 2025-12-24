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
	l.AddRoute(def.SyncStatus, l.syncStatus)
	l.AddRoute(def.StartGame, l.startGame)
	l.AddRoute(def.LeaveRoom, l.leaveRoom)
	l.AddRoute(def.OptGame, l.optGame)
	l.AddRoute(def.Reconnect, l.reconnect)
	l.AddRoute(def.Offline, l.offline)
}

func (l *Local) getGameStatus(in *local.Input) error {
	req := new(pb.GameStatusReq)
	if err := in.Unpack(req); err != nil {
		return err
	}
	log.Println("getGameStatus", req.String())
	rsp := new(pb.GameStatusRsp)
	room, ok := l.users[req.Uid]
	if ok && room.playing {
		rsp.RoomId = room.roomId
	}
	return in.Rpc(rsp)
}

func (l *Local) offline(in *local.Input) error {
	req := new(pb.Offline)
	if err := in.Unpack(req); err != nil {
		return err
	}
	log.Println("offline", req.String())
	if room, ok := l.users[req.Uid]; ok {
		room.Offline(req.Uid)
	}
	return nil
}

func (l *Local) reconnect(in *local.Input) error {
	req := new(pb.Reconnect)
	if err := in.Unpack(req); err != nil {
		return err
	}
	log.Println("reconnect", req.String())
	if req.RoomId > 0 {
		room, ok := l.rooms[req.RoomId]
		if ok {
			room.Reconnect(req.Uid, uint16(req.GateId))
		}
	}

	return nil
}

func (l *Local) startGame(in *local.Input) error {
	req, rsp := new(pb.StartGameReq), new(pb.StartGameRsp)
	if err := in.Unpack(req); err != nil {
		return err
	}
	log.Println("startGame", req.String())
	room := l.rooms[req.RoomId]
	if room == nil {
		room = NewRoom(l, req.RoomId)
		l.rooms[room.roomId] = room
	}
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
	room.wait.WaitAll(def.StartGame)
	for uid := range req.Users {
		room.GetUserByUID(uid).Send(def.StartGame, rsp)
	}

	return nil
}

func (l *Local) syncStatus(in *local.Input) error {
	req := new(pb.SyncStatus)
	if err := in.Unpack(req); err != nil {
		return err
	}
	log.Println("syncStatus", req.String())
	room, ok := l.rooms[req.RoomId]
	if !ok {
		return nil
	}
	user := room.GetUserByUID(req.Uid)
	if user == nil {
		return nil
	}
	if ok := room.wait.Done(req.Cmd, int32(user.Seat())); !ok {
		return nil
	}
	if !room.wait.IsFull() {
		return nil
	}
	if req.Cmd == def.StartGame {
		room.Start()
	} else if req.Cmd == def.GameFaPai {
		room.NotifyChuPai()
	}
	return nil
}

func (l *Local) leaveRoom(in *local.Input) error {
	req, rsp := new(pb.LeaveRoomReq), new(pb.LeaveRoomRsp)
	if err := in.Unpack(req); err != nil {
		return err
	}
	log.Println("leaveRoom", req.String())
	room, ok := l.rooms[req.RoomId]
	if !ok {
		return nil
	}
	if room.playing {
		return nil
	}
	room.DelUser(req.Uid)
	in.Response(req.Uid, in.Cmd, rsp)
	return nil
}

func (l *Local) optGame(in *local.Input) error {
	req := new(pb.MjOpt)
	if err := in.Unpack(req); err != nil {
		return err
	}

	log.Println("tap game")
	if room, ok := l.rooms[req.RoomId]; ok {
		room.MjOp(req.Uid, req)
	}

	return nil
}
