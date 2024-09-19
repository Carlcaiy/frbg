package route

import (
	"fmt"
	"frbg/def"
	"frbg/examples/cmd"
	"frbg/examples/proto"
	"frbg/network"
	"frbg/parser"
	"log"
	"math/rand"
)

type User struct {
	uid     uint32
	gateId  uint8 // 网关ID
	tap     int32
	offline bool
}

type RoomTemplete struct {
	Id        int32
	UserCount int32
}

type Room struct {
	hall      *network.Conn
	hallId    uint8
	roomId    uint32
	tempId    uint32
	l         *Local
	Users     []*User
	turn      int
	guess_num int32
}

func (r *Room) Reset() {
	for _, u := range r.Users {
		u.tap = 0
	}
	r.guess_num = rand.Int31n(100) + 1
}

func (r *Room) GetUser(uid uint32) *User {
	for _, u := range r.Users {
		if u.uid == uid {
			return u
		}
	}
	return nil
}

func (r *Room) Offline(uid uint32) {
	if u := r.GetUser(uid); u != nil {
		log.Printf("user :%d offline\n", uid)
		u.offline = true
	}
}

func (r *Room) Reconnect(uid uint32, gateId uint8) {
	for i, u := range r.Users {
		if u.uid == uid {
			u.gateId = gateId
			u.offline = false

			log.Println(u.uid, u.gateId)
			bs, _ := parser.Pack(u.uid, def.ST_User, cmd.SyncData, &proto.SyncData{
				Data:   "Reconnect game success",
				RoomId: r.roomId,
			})
			r.l.SendToGate(u.gateId, bs)

			log.Println("Reconnect", "uid:", uid, "sit", i, "turn", r.turn)
			if i == r.turn {
				bs, _ := parser.Pack(uid, def.ST_User, cmd.Round, &proto.Empty{})
				r.l.SendToGate(u.gateId, bs)
			}
			return
		}
	}
	log.Printf("Reconnect error: not find uid:%d\n", uid)
}

func (r *Room) Start() {
	r.Reset()
	log.Println("Start", "turn:", r.turn)

	// 同步所有数据
	for i, u := range r.Users {
		log.Printf("uid:%d seat:%d gateId:%d\n", u.uid, i, u.gateId)
		bs, _ := parser.Pack(u.uid, def.ST_User, cmd.SyncData, &proto.SyncData{
			Data:   "game start",
			RoomId: r.roomId,
			GameId: uint32(r.l.ServerId),
		})
		r.l.SendToGate(u.gateId, bs)
	}

	// 当前回合
	u := r.Users[r.turn]
	bs, _ := parser.Pack(u.uid, def.ST_User, cmd.Round, &proto.Empty{})
	r.l.SendToGate(u.gateId, bs)
}

func (r *Room) Tap(uid uint32, tap int32) {
	u := r.Users[r.turn]
	if uid != u.uid {
		log.Printf("tap err, uid:%d should uid:%d\n", uid, u.uid)
		return
	}

	tips := ""
	if tap < r.guess_num {
		tips = "小了"
	} else if tap > r.guess_num {
		tips = "大了"
	} else {
		tips = fmt.Sprintf("答对了%d", r.guess_num)
	}
	for _, u := range r.Users {
		bs, _ := parser.Pack(u.uid, def.ST_User, cmd.Tap, &proto.Tap{
			Uid:    uid,
			RoomId: r.roomId,
			Tap:    tap,
			Tips:   tips,
		})
		r.l.SendToGate(u.gateId, bs)
	}

	if tap == r.guess_num {
		bs, _ := parser.Pack(uid, def.ST_Hall, cmd.GameOver, &proto.GameOver{
			TempId: r.tempId,
			RoomId: r.roomId,
			Data:   "game over",
		})
		r.l.SendToSid(r.hallId, bs, def.ST_Hall)
		r.gameOver()
		return
	}

	r.turn = (r.turn + 1) % len(r.Users)
	u = r.Users[r.turn]
	bs, _ := parser.Pack(u.uid, def.ST_User, cmd.Round, &proto.Empty{})
	r.l.SendToGate(u.gateId, bs)
}

func (r *Room) gameOver() {
	log.Println("game over")
}

func (r *Room) SendOne(bs []byte) {
	r.hall.Write(bs)
}

func (r *Room) SendOther(uid uint32, bs []byte) {
	multi := &proto.MultiBroadcast{
		Data: bs,
	}
	for _, u := range r.Users {
		if u.uid != uid {
			multi.Uids = append(multi.Uids, u.uid)
		}
	}
	buf, _ := parser.Pack(0, def.ST_User, cmd.MultiBroadcast, multi)
	r.hall.Write(buf)
}

func (r *Room) SendAll(bs []byte) {
	multi := &proto.MultiBroadcast{
		Data: bs,
	}
	for _, u := range r.Users {
		multi.Uids = append(multi.Uids, u.uid)
	}
	buf, _ := parser.Pack(0, def.ST_User, cmd.MultiBroadcast, multi)
	r.hall.Write(buf)
}
