package route

import (
	"fmt"
	"frbg/def"
	"frbg/examples/cmd"
	"frbg/examples/proto"
	"frbg/mj"
	"frbg/network"
	"frbg/parser"
	"log"
	"math/rand"
)

type User struct {
	uid        uint32
	gateId     uint8 // 网关ID
	tap        int32
	offline    bool
	can_op     uint8
	mj_hands   []uint8    // 麻将
	mj_history []uint8    // 出牌
	mj_group   []mj.Group // 麻将组
}

func (u *User) Reset() {
	u.mj_hands = u.mj_hands[:0]
	u.mj_history = u.mj_history[:0]
	u.mj_group = u.mj_group[:0]
}

func (u *User) remove_mj(val uint8, num int) {
	tail := len(u.mj_hands)
	for i, v := range u.mj_hands {
		if v == val {
			u.mj_hands[i], u.mj_hands[tail] = u.mj_hands[tail], u.mj_hands[i]
			tail--
			num--
			if num == 0 {
				break
			}
		}
	}
	u.mj_hands = u.mj_hands[:tail-num]
}

func (u *User) DaMj(val uint8) {
	u.remove_mj(val, 1)
}

func (u *User) MoMj(val uint8) {
	u.mj_hands = append(u.mj_hands, val)
}

func (u *User) LChi(val uint8) {
	val1, val2 := val+1, val+2
	u.remove_mj(val1, 1)
	u.remove_mj(val2, 1)
	u.mj_group = append(u.mj_group, mj.Group{Op: mj.LChi, Val: val})
}

func (u *User) MChi(val uint8) {
	val1, val2 := val-1, val+2
	u.remove_mj(val1, 1)
	u.remove_mj(val2, 1)
	u.mj_group = append(u.mj_group, mj.Group{Op: mj.MChi, Val: val})
}

func (u *User) RChi(val uint8) {
	val1, val2 := val-1, val-2
	u.remove_mj(val1, 1)
	u.remove_mj(val2, 1)
	u.mj_group = append(u.mj_group, mj.Group{Op: mj.RChi, Val: val})
}

func (u *User) PengMj(val uint8) bool {
	cnt := 0
	for _, v := range u.mj_hands {
		if v == val {
			cnt++
		}
	}
	if cnt < 2 {
		return false
	}

	u.remove_mj(val, 2)
	u.mj_group = append(u.mj_group, mj.Group{Op: mj.Peng, Val: val})
	return true
}

func (u *User) MGangMj(val uint8) bool {
	cnt := 0
	for _, v := range u.mj_hands {
		if v == val {
			cnt++
		}
	}
	if cnt < 3 {
		return false
	}

	u.remove_mj(val, 3)
	u.mj_group = append(u.mj_group, mj.Group{Op: mj.MGang, Val: val})
	return true
}

func (u *User) BGangMj(val uint8) bool {
	for _, v := range u.mj_group {
		if v.Op == mj.Peng && v.Val == val {
			v.Op = mj.BGang
		}
	}
	u.remove_mj(val, 1)
	return true
}

func (u *User) AGangMj(val uint8) bool {
	cnt := 0
	for _, v := range u.mj_hands {
		if v == val {
			cnt++
		}
	}
	if cnt < 4 {
		return false
	}

	u.remove_mj(val, 4)
	u.mj_group = append(u.mj_group, mj.Group{Op: mj.AGang, Val: val})
	return true
}

func (u *User) DianPao(val uint8) bool {
	st := mj.New(append(u.mj_hands, val))
	return st.CanHu()
}

func (u *User) Zimo() bool {
	st := mj.New(u.mj_hands)
	return st.CanHu()
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

func (r *Room) SetPlayers(uids []uint32) {
	for i := range r.Users {
		r.Users[i].uid = uids[i]
		r.Users[i].Reset()
	}
}

func (r *Room) Reset() {
	for _, u := range r.Users {
		u.tap = 0
	}
	r.guess_num = rand.Int31n(100) + 1
}

func (r *Room) GetConn(uid uint32) *User {
	for _, u := range r.Users {
		if u.uid == uid {
			return u
		}
	}
	return nil
}

func (r *Room) Offline(uid uint32) {
	if u := r.GetConn(uid); u != nil {
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
	buf, _ := parser.Pack(0, def.ST_User, cmd.MultiBC, multi)
	r.hall.Write(buf)
}

func (r *Room) SendAll(bs []byte) {
	multi := &proto.MultiBroadcast{
		Data: bs,
	}
	for _, u := range r.Users {
		multi.Uids = append(multi.Uids, u.uid)
	}
	buf, _ := parser.Pack(0, def.ST_User, cmd.MultiBC, multi)
	r.hall.Write(buf)
}
