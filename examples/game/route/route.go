package route

import (
	"frbg/examples/cmd"
	"frbg/examples/proto"
	"frbg/local"
	"frbg/network"
	"frbg/parser"
	"log"
)

type Local struct {
	*local.BaseLocal
	rooms map[uint32]*Room // 房间配置
	users map[uint32]*Room
}

func NewLocal(st *network.ServerConfig) *Local {
	return &Local{
		BaseLocal: local.NewBase(st),
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

func (l *Local) offline(c *network.Conn, msg *parser.Message) error {
	if room, ok := l.users[msg.UserID]; ok {
		room.Offline(msg.UserID)
	}
	return nil
}

func (l *Local) reconnect(c *network.Conn, msg *parser.Message) error {
	pack := new(proto.Reconnect)
	if err := msg.Unpack(pack); err != nil {
		return err
	}
	log.Println("reconnect", pack.String())
	if pack.RoomId > 0 {
		room, ok := l.rooms[pack.RoomId]
		if ok {
			room.Reconnect(msg.UserID, uint8(pack.GateId))
		}
	}

	return nil
}

func (l *Local) enterRoom(c *network.Conn, msg *parser.Message) error {
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
		room = NewRoom(l)
		l.rooms[room.roomId] = room
	}
	l.users[msg.UserID] = room
	room.AddUser(msg.UserID, msg.GateID)
	bs, _ := msg.PackCmd(msg.Cmd, rsp)
	c.Write(bs)
	if room.Full() {
		room.Start()
	}

	return nil
}

func (l *Local) leaveRoom(c *network.Conn, msg *parser.Message) error {
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
	room.DelUser(msg.UserID)
	bs, _ := msg.PackProto(rsp)
	c.Write(bs)
	return nil
}

func (l *Local) optGame(c *network.Conn, msg *parser.Message) error {
	data := new(proto.MjOpt)
	if err := msg.Unpack(data); err != nil {
		return err
	}

	log.Println("tap game")
	if room, ok := l.rooms[data.RoomId]; ok {
		room.MjOp(msg.UserID, data)
	}

	return nil
}
