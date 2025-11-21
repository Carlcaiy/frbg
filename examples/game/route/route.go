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
	return &Local{
		BaseLocal: local.NewBase(),
		rooms:     make(map[uint32]*Room),
		users:     make(map[uint32]*Room),
	}
}

func (l *Local) Init() {
	l.BaseLocal.Init()
	l.AddRoute(def.EnterRoom, l.enterRoom)
	l.AddRoute(def.LeaveRoom, l.leaveRoom)
	l.AddRoute(def.OptGame, l.optGame)
	l.AddRoute(def.Reconnect, l.reconnect)
	l.AddRoute(def.Offline, l.offline)
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

func (l *Local) enterRoom(in *local.Input) error {
	req, rsp := new(pb.EnterRoomReq), new(pb.EnterRoomRsp)
	if err := in.Unpack(req); err != nil {
		return err
	}
	log.Println("enterRoom", req.String())
	var room *Room
	for _, v := range l.rooms {
		if v.playing {
			continue
		}
		if len(v.Users) == 4 {
			continue
		}
		if room == nil {
			room = v
		}
		if len(room.Users) < len(v.Users) {
			room = v
		}
		if len(room.Users) == 3 {
			break
		}
	}
	if room == nil {
		room = NewRoom(l, 0)
		l.rooms[room.roomId] = room
	}
	l.users[req.Uid] = room
	room.AddUser(req.Uid, uint16(req.GateId))
	in.Response(req.Uid, in.Cmd, rsp)
	if room.Full() {
		room.Start()
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
