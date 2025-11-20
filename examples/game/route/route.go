package route

import (
	"frbg/examples/cmd"
	"frbg/examples/proto"
	"frbg/local"
	"frbg/network"
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
	l.AddRoute(cmd.EnterRoom, l.enterRoom)
	l.AddRoute(cmd.LeaveRoom, l.leaveRoom)
	l.AddRoute(cmd.OptGame, l.optGame)
	l.AddRoute(cmd.Reconnect, l.reconnect)
	l.AddRoute(cmd.Offline, l.offline)
}

func (l *Local) offline(msg *network.Message) error {
	req := new(proto.Offline)
	if err := msg.Unpack(req); err != nil {
		return err
	}
	log.Println("offline", req.String())
	if room, ok := l.users[req.Uid]; ok {
		room.Offline(req.Uid)
	}
	return nil
}

func (l *Local) reconnect(msg *network.Message) error {
	req := new(proto.Reconnect)
	if err := msg.Unpack(req); err != nil {
		return err
	}
	log.Println("reconnect", req.String())
	if req.RoomId > 0 {
		room, ok := l.rooms[req.RoomId]
		if ok {
			room.Reconnect(req.Uid, uint8(req.GateId))
		}
	}

	return nil
}

func (l *Local) enterRoom(msg *network.Message) error {
	req, rsp := new(proto.EnterRoomReq), new(proto.EnterRoomRsp)
	if err := msg.Unpack(req); err != nil {
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
	room.AddUser(req.Uid, uint8(req.GateId))
	msg.Response(msg.Cmd, rsp)
	if room.Full() {
		room.Start()
	}

	return nil
}

func (l *Local) leaveRoom(msg *network.Message) error {
	req, rsp := new(proto.LeaveRoomReq), new(proto.LeaveRoomRsp)
	if err := msg.Unpack(req); err != nil {
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
	msg.Response(msg.Cmd, rsp)
	return nil
}

func (l *Local) optGame(msg *network.Message) error {
	req := new(proto.MjOpt)
	if err := msg.Unpack(req); err != nil {
		return err
	}

	log.Println("tap game")
	if room, ok := l.rooms[req.RoomId]; ok {
		room.MjOp(req.Uid, req)
	}

	return nil
}
