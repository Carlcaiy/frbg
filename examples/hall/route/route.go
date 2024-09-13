package route

import (
	"fmt"
	"frbg/def"
	"frbg/examples/cmd"
	"frbg/examples/db"
	"frbg/examples/hall/slots"
	"frbg/examples/proto"
	"frbg/local"
	"frbg/network"
	"frbg/parser"
	"log"
	"time"
)

type Local struct {
	templetes map[uint32]*RoomTemplete
	rooms     map[uint32]*RoomInstance
	*local.BaseLocal
}

func NewLocal(st *network.ServerConfig) *Local {
	return &Local{
		BaseLocal: local.NewBase(st),
		templetes: make(map[uint32]*RoomTemplete),
		rooms:     make(map[uint32]*RoomInstance),
	}
}

func (l *Local) Init() {
	l.load_room_templete()

	l.BaseLocal.Init()
	l.AddRoute(cmd.ReqRoomList, l.getRoomList)
	l.AddRoute(cmd.ReqEnterRoom, l.reqEnterRoom)
	l.AddRoute(cmd.ReqLeaveRoom, l.reqLeaveRoom)
	l.AddRoute(cmd.GameOver, l.gameOver)
	l.AddRoute(cmd.Offline, l.offline)
	l.AddRoute(cmd.SlotsEnter, l.reqEnterSlots)
	l.AddRoute(cmd.SlotsSpin, l.reqSlotsSpin)
	l.AddRoute(cmd.SlotsLeave, l.reqLeaveSlots)
}

func (l *Local) offline(c *network.Conn, msg *parser.Message) error {
	l.DelUser(msg.UserID)
	return nil
}

func (l *Local) load_room_templete() {
	for i := uint32(0); i < 5; i++ {
		l.templetes[i] = &RoomTemplete{
			TempId:    i,
			UserCount: 2,
			GameID:    1,
		}
	}
}

func (l *Local) gameOver(c *network.Conn, msg *parser.Message) error {
	data := new(proto.GameOver)
	msg.Unpack(data)

	var room *RoomInstance
	if r, ok := l.rooms[data.RoomId]; ok {
		room = r
	}

	// 先发消息后处理数据
	if room != nil {
		if room.sitCount == room.UserCount {
			for _, user := range room.users {
				bs, _ := parser.Pack(user.UserID(), def.ST_Gate, cmd.CountDown, &proto.Empty{})
				l.SendToGate(user.gateID, bs)
			}
			l.Start(room.tevent)
		}

		for _, u := range room.users {
			bs, _ := parser.Pack(u.userID, def.ST_Gate, cmd.GameOver, data)
			l.SendToGate(u.gateID, bs)
		}
	}

	return nil
}

func (l *Local) getRoomList(c *network.Conn, msg *parser.Message) error {
	log.Println("getRoomList")
	data := new(proto.ReqRoomList)
	if err := msg.Unpack(data); err != nil {
		return err
	}
	res := new(proto.ResRoomList)
	res.Rooms = make([]*proto.RoomInfo, len(l.templetes))
	for i, temp := range l.templetes {
		res.Rooms[i] = &proto.RoomInfo{
			ServerId: 0,
			RoomId:   temp.TempId,
			Tag:      1,
		}
	}
	if buf, err := parser.Pack(msg.UserID, def.ST_Gate, cmd.ResRoomList, res); err == nil {
		c.Write(buf)
	}
	return nil
}

func (l *Local) reqEnterRoom(c *network.Conn, msg *parser.Message) error {
	data := new(proto.ReqEnterRoom)
	if err := msg.Unpack(data); err != nil {
		return err
	}
	log.Printf("reqEnterRoom uid:%d tempId:%d\n", msg.UserID, data.TempleteId)

	user, ok := l.GetUser(msg.UserID).(*User)
	if !ok {
		return fmt.Errorf("not found user:%d", msg.UserID)
	}

	var room *RoomInstance

	// 寻找一个空的房间
	for _, r := range l.rooms {
		if r.TempId == data.TempleteId && r.status == 0 && r.sitCount < r.UserCount {
			room = r
		}
	}

	// 没找到房间，新建一个房间
	if room == nil {
		temp := l.templetes[data.TempleteId]
		room = &RoomInstance{
			RoomTemplete: temp,
			status:       0,
			users:        make([]*User, temp.UserCount),
			conn:         c,
			roomID:       uint32(len(l.rooms)) + 1000,
			tevent: local.NewDelayEvent(time.Second*3, func() {
				if room.sitCount < room.UserCount {
					return
				}
				room.status = 1
				greq := &proto.StartGame{
					TempId: room.TempId,
					RoomId: room.roomID,
					HallId: l.ServerId,
					Uids:   make([]uint32, room.sitCount),
				}
				for i := range greq.Uids {
					greq.Uids[i] = room.users[i].userID
					log.Printf("i:%d uid:%d gateid:%d\n", i, room.users[i].userID, room.users[i].gateID)
				}
				bs, _ := parser.Pack(msg.UserID, def.ST_Game, cmd.GameStart, greq)
				if err := l.SendModUid(room.roomID, bs, def.ST_Game); err == nil {
					log.Printf("配桌成功")
				} else {
					log.Printf("配桌失败")
				}
			}),
		}
		l.rooms[room.roomID] = room
	}

	if room != nil {
		user.roomID = room.roomID

		for i := range room.users {
			if room.users[i] == nil {
				room.users[i] = user
				room.sitCount++
				break
			}
		}

		db.SetRoom(user.userID, room.roomID)

		res := &proto.ResEnterRoom{}
		res.Uids = make([]uint32, 0, room.sitCount)
		for i := range room.users {
			if room.users[i] != nil {
				res.Uids = append(res.Uids, room.users[i].userID)
			}
		}

		bs, _ := parser.Pack(msg.UserID, def.ST_User, cmd.ResEnterRoom, res)
		c.Write(bs)

		if room.sitCount == room.UserCount {
			for _, user := range room.users {
				bs, _ := parser.Pack(user.UserID(), def.ST_Gate, cmd.CountDown, &proto.Empty{})
				l.SendToGate(user.gateID, bs)
			}

			l.Start(room.tevent)
		}
	}

	return nil
}

func (l *Local) reqLeaveRoom(c *network.Conn, msg *parser.Message) error {
	data := new(proto.ReqLeaveRoom)
	msg.Unpack(data)

	if room, ok := l.rooms[data.RoomId]; ok {
		if room.status == 0 {
			for i, u := range room.users {
				if u.UserID() == msg.UserID {
					room.users[i] = nil
					room.sitCount -= 1
					l.Stop(room.tevent)
					db.SetRoom(u.UserID(), 0)
					return nil
				}
			}
		} else {
			return fmt.Errorf("游戏中不能离开")
		}
	}

	return fmt.Errorf("reqLeaveRoom error %s", data)
}

// 请求进入老虎机
func (l *Local) reqEnterSlots(c *network.Conn, msg *parser.Message) error {
	req := new(proto.ReqEnterSlots)
	if err := msg.Unpack(req); err != nil {
		return err
	}
	conf := slots.GetSlotsData(msg.UserID, req.SlotsId)
	rsp := proto.ResEnterSlots{
		GameId: req.SlotsId,
		Bet:    conf.BetConf.Bet,
		Level:  conf.BetConf.Level,
		Line:   conf.BetConf.Lines,
		Lines:  conf.RouteConf,
		Elems:  conf.ElemConf,
	}
	bs, err := msg.PackProto(&rsp)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	return l.SendToGate(msg.UserID, bs)
}

// 老虎机请求摇奖
func (l *Local) reqSlotsSpin(c *network.Conn, msg *parser.Message) error {
	req := new(proto.ReqSlotsSpin)
	if err := msg.Unpack(req); err != nil {
		return err
	}

	slotsData := slots.GetSlotsData(msg.UserID, req.GameId)
	rsp, err := slotsData.Spin(int64(req.Bet) * int64(req.Level))
	if err != nil {
		return err
	}

	bs, err := msg.PackProto(rsp)
	if err != nil {
		return err
	}

	return l.SendToGate(msg.UserID, bs)
}

// 离开老虎机
func (l *Local) reqLeaveSlots(c *network.Conn, msg *parser.Message) error {
	req := new(proto.ReqLeaveSlots)
	if err := msg.Unpack(req); err != nil {
		return err
	}
	slots.DelSlotsData(msg.UserID)
	return nil
}

func (l *Local) Close(conn *network.Conn) {
	l.BaseLocal.Close(conn)
	// 大厅服，清理所有相关桌子
	if conn.ServerType == def.ST_Game {
		for _, room := range l.rooms {
			if room.GameID == conn.ServerId {
				room.status = 0
				room.sitCount = 0
				for i := range room.users {
					room.users[i] = nil
				}
			}
		}
	}
}
