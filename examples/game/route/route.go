package route

import (
	"frbg/examples/cmd"
	"frbg/examples/db"
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
	l.AddRoute(cmd.GameStart, l.startGame)
	l.AddRoute(cmd.Tap, l.tapGame)
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
			room.Reconnect(msg.UserID, pack.GateId)
		}
	}

	return nil
}

func (l *Local) startGame(c *network.Conn, msg *parser.Message) error {
	data := new(proto.StartGame)
	msg.Unpack(data)
	log.Println("startGame", data.String())
	room, ok := l.rooms[data.RoomId]
	if !ok {
		room = &Room{
			hall:   c,
			hallId: data.HallId,
			roomId: data.RoomId,
			tempId: data.TempId,
			l:      l,
		}
		l.rooms[data.RoomId] = room
	}
	room.Users = make([]*User, len(data.Uids))
	for i, u := range room.Users {
		room.Users[i] = &User{
			uid: data.Uids[i],
			tap: 0,
		}
		l.users[data.Uids[i]] = room
		db.SetGame(u.uid, l.ServerId)
	}
	room.Start()
	return nil
}

func (l *Local) tapGame(c *network.Conn, msg *parser.Message) error {
	data := new(proto.Tap)
	msg.Unpack(data)
	log.Println("tap game")
	if room, ok := l.rooms[data.RoomId]; ok {
		room.Tap(msg.UserID, data.Tap)
	}

	return nil
}
