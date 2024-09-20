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

var route *Local

type Local struct {
	templetes map[uint32]*RoomTemplete
	rooms     map[uint32]*RoomInstance
	*local.BaseLocal
}

func New(st *network.ServerConfig) *Local {
	route = &Local{
		BaseLocal: local.NewBase(st),
		templetes: make(map[uint32]*RoomTemplete),
		rooms:     make(map[uint32]*RoomInstance),
	}
	route.init()
	return route
}

func (l *Local) init() {
	l.load_room_templete()

	l.BaseLocal.Init()
	l.AddRoute(cmd.GetGameList, l.getGameList)
	l.AddRoute(cmd.GetRoomList, l.getRoomList)
	l.AddRoute(cmd.EnterRoom, l.enterRoom)
	l.AddRoute(cmd.LeaveRoom, l.leaveRoom)
	l.AddRoute(cmd.GameOver, l.gameOver)
	l.AddRoute(cmd.Offline, l.offline)
	l.AddRoute(cmd.SlotsEnter, l.slotsEnter)
	l.AddRoute(cmd.SlotsSpin, l.slotsSpin)
	l.AddRoute(cmd.SlotsLeave, l.slotsLeave)
	l.AddRoute(cmd.Test, l.test)
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

func (l *Local) test(conn *network.Conn, msg *parser.Message) error {
	data := new(proto.Test)
	if err := msg.Unpack(data); err != nil {
		return err
	}

	b, _ := parser.Pack(msg.UserID, def.ST_User, msg.Cmd, &proto.Test{
		Uid:       data.Uid,
		StartTime: data.StartTime,
	})

	_, err := conn.Write(b)
	return err
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
	req, rsp := new(proto.GetRoomListReq), new(proto.GetRoomListRsp)
	if err := msg.Unpack(req); err != nil {
		return err
	}
	rsp.Rooms = make([]*proto.RoomInfo, len(l.templetes))
	for i, temp := range l.templetes {
		rsp.Rooms[i] = &proto.RoomInfo{
			ServerId: 0,
			RoomId:   temp.TempId,
			Tag:      1,
		}
	}
	if buf, err := parser.Pack(msg.UserID, def.ST_Gate, msg.Cmd, rsp); err == nil {
		c.Write(buf)
	}
	return nil
}

func (l *Local) getGameList(c *network.Conn, msg *parser.Message) error {
	log.Println("getGameList")
	req, rsp := new(proto.GetRoomListReq), new(proto.GetRoomListRsp)
	if err := msg.Unpack(req); err != nil {
		return err
	}
	rsp.Rooms = make([]*proto.RoomInfo, len(l.templetes))
	for i, temp := range l.templetes {
		rsp.Rooms[i] = &proto.RoomInfo{
			ServerId: 0,
			RoomId:   temp.TempId,
			Tag:      1,
		}
	}
	if buf, err := parser.Pack(msg.UserID, def.ST_Gate, msg.Cmd, rsp); err == nil {
		c.Write(buf)
	}
	return nil
}

func (l *Local) enterRoom(c *network.Conn, msg *parser.Message) error {
	req, rsp := new(proto.EnterRoomReq), new(proto.EnterRoomRsp)
	if err := msg.Unpack(req); err != nil {
		return err
	}
	log.Printf("enterRoom uid:%d tempId:%d\n", msg.UserID, req.TempleteId)

	user, ok := l.GetUser(msg.UserID).(*User)
	if !ok {
		return fmt.Errorf("not found user:%d", msg.UserID)
	}

	var room *RoomInstance

	// 寻找一个空的房间
	for _, r := range l.rooms {
		if r.TempId == req.TempleteId && r.status == 0 && r.sitCount < r.UserCount {
			room = r
		}
	}

	// 没找到房间，新建一个房间
	if room == nil {
		temp := l.templetes[req.TempleteId]
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
					HallId: uint32(l.ServerId),
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

		rsp.Uids = make([]uint32, 0, room.sitCount)
		for i := range room.users {
			if room.users[i] != nil {
				rsp.Uids = append(rsp.Uids, room.users[i].userID)
			}
		}

		bs, _ := parser.Pack(msg.UserID, def.ST_User, msg.Cmd, rsp)
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

func (l *Local) leaveRoom(c *network.Conn, msg *parser.Message) error {
	req := new(proto.LeaveRoomReq)
	if err := msg.Unpack(req); err != nil {
		return err
	}

	if room, ok := l.rooms[req.RoomId]; ok {
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

	return fmt.Errorf("leaveRoom error %s", req)
}

// 请求进入老虎机
func (l *Local) slotsEnter(c *network.Conn, msg *parser.Message) error {
	req := new(proto.EnterSlotsReq)
	if err := msg.Unpack(req); err != nil {
		return err
	}
	conf := slots.GetSlotsData(msg.UserID, req.SlotsId)
	rsp := proto.EnterSlotsRsp{
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
	return l.SendToGate(msg.GateID, bs)
}

// 老虎机请求摇奖
func (l *Local) slotsSpin(c *network.Conn, msg *parser.Message) error {
	req := new(proto.SlotsSpinReq)
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

	return l.SendToGate(msg.GateID, bs)
}

// 离开老虎机
func (l *Local) slotsLeave(c *network.Conn, msg *parser.Message) error {
	req := new(proto.LeaveSlotsReq)
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
