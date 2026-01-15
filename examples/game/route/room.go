package route

import (
	"frbg/def"
	"frbg/examples/pb"
	"frbg/mj"
	"log"
	"math/rand"
	"time"

	"google.golang.org/protobuf/proto"
)

type DeskMj struct {
	MjVal  byte
	Belong byte
}

type Room struct {
	l           *Local
	master      uint32   // 房主ID
	roomId      uint32   // 房间ID
	BookUids    []uint32 // 已预约用户
	Users       []*User  // 用户
	turn        int      // 庄家
	mj          []uint8  // 麻将
	usedMjIndex []uint8  // 已使用的麻将下标
	mjIndex     uint8    // 麻将索引
	touzi       []int32  // 骰子
	pizi        uint8    // 皮子
	laizi       uint8    // 赖子
	zhuang      int32    // 庄家
	waitOther   bool     // 等待其他玩家操作
	history     []*mj.MjOp
	playing     bool
	huangZhuang int // 黄庄剩余牌数
	endTime     time.Time
}

func NewRoom(l *Local, rid uint32) *Room {
	room := &Room{
		l:           l,
		mj:          make([]uint8, len(mj.BanBiShanMJ)),
		touzi:       make([]int32, 2),
		huangZhuang: 10,
		roomId:      rid,
	}
	copy(room.mj, mj.BanBiShanMJ)
	return room
}

func (r *Room) GetUserByUID(uid uint32) *User {
	for _, u := range r.Users {
		if u.uid == uid {
			return u
		}
	}
	return nil
}

func (r *Room) AddUser(uid uint32, gateId uint16) {
	r.Users = append(r.Users, &User{
		l:      r.l,
		uid:    uid,
		gateId: gateId,
		seat:   int(len(r.Users)),
	})
}

func (r *Room) DelUser(uid uint32) {
	for i, u := range r.Users {
		if u.uid == uid {
			count := len(r.Users) - 1
			r.Users[i], r.Users[count] = r.Users[count], r.Users[i]
			return
		}
	}
}

func (r *Room) Full() bool {
	return len(r.Users) == 4
}

func (r *Room) GetUser(seat int) *User {
	return r.Users[seat%len(r.Users)]
}

func (r *Room) SetPlayers(uids []uint32) {
	for i := range r.Users {
		r.Users[i].uid = uids[i]
		r.Users[i].Reset()
	}
}

func (r *Room) Reset() {
	for _, u := range r.Users {
		u.Reset()
	}
	rand.Shuffle(len(r.mj), func(i, j int) {
		r.mj[i], r.mj[j] = r.mj[j], r.mj[i]
	})

	r.touzi[0] = rand.Int31n(6) + 1
	r.touzi[1] = rand.Int31n(6) + 1
	r.mjIndex = 0
	r.usedMjIndex = r.usedMjIndex[:0]
	r.history = r.history[:0]
	r.playing = true
	r.waitOther = false

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

func (r *Room) Reconnect(uid uint32, gateId uint16) {
	req := &pb.DeskSnapshot{
		RoomId:      r.roomId,
		Pizi:        int32(r.pizi),
		Touzi:       r.touzi,
		Zhuang:      uid,
		Laizi:       int32(r.laizi),
		UsedMjIndex: r.usedMjIndex,
		Info:        make([]*pb.PlayerInfo, len(r.Users)),
	}
	for i, u := range r.Users {
		req.Info[i] = &pb.PlayerInfo{
			Uid:   u.uid,
			Dachu: r.Users[i].mj_history,
			Cpgs:  make([][]byte, len(r.Users[i].mj_group)),
		}
		for j, g := range r.Users[i].mj_group {
			req.Info[i].Cpgs[j] = g.ToBytes()
		}
		if u.uid == uid {
			req.Info[i].Hands = r.Users[i].mj_hands
			req.Info[i].CanOp = u.CanOp()
		} else {
			req.Info[i].Hands = make([]byte, len(r.Users[i].mj_hands))
		}
	}
	user := r.GetUserByUID(uid)
	if user == nil {
		log.Printf("Reconnect error: not find uid:%d\n", uid)
		return
	}
	user.gateId = gateId
	user.offline = false
	log.Println("Reconnect", "uid:", uid, "sit", user.Seat(), "turn", r.turn)
	user.Send(def.Reconnect, req)
}

func (r *Room) MajFaPai() {
	r.Reset()
	log.Println("FaPai")

	faPai := make([]*pb.DeskMj, 0, 4*3+4+1)
	// 每个玩家发3轮
	for t := 0; t < 3; t++ {
		// 4个玩家，从庄家开始
		for i := 0; i < 4; i++ {
			u := r.Users[(r.turn+i)%4]
			// 每个玩家发4个麻將
			for j := 0; j < 4; j++ {
				mjVal := r.mj[r.mjIndex]
				faPai = append(faPai, &pb.DeskMj{
					Index: int32(r.mjIndex),
					Uid:   u.uid,
					MjVal: int32(mjVal),
				})
				r.usedMjIndex = append(r.usedMjIndex, r.mjIndex)
				r.mjIndex++
				u.MoMj(mjVal)
			}
		}
	}
	for i := 0; i < 5; i++ {
		u := r.Users[(r.turn+i)%4]
		mjVal := r.mj[r.mjIndex]
		faPai = append(faPai, &pb.DeskMj{
			Index: int32(r.mjIndex),
			Uid:   u.uid,
			MjVal: int32(r.mj[r.mjIndex]),
		})
		r.usedMjIndex = append(r.usedMjIndex, r.mjIndex)
		r.mjIndex++
		u.MoMj(mjVal)
	}

	// 确定赖子
	col := r.touzi[1]
	if r.touzi[0] < r.touzi[1] {
		col = r.touzi[0]
	}
	piziIndex := (9*3+8)*2 - col*2
	r.pizi = r.mj[piziIndex]
	r.laizi = mj.GetLaizi(r.pizi)

	// 组装信息
	zhuang := r.Users[r.turn]
	piziMj := &pb.DeskMj{
		Index: piziIndex,
		MjVal: int32(r.pizi),
	}

	r.waitOther = false
	for _, u := range r.Users {
		log.Printf("mj:%v", u.Mj())
		handsMj := make([]*pb.DeskMj, len(faPai))
		for i := range faPai {
			handsMj[i] = &pb.DeskMj{
				Index: faPai[i].Index,
				Uid:   faPai[i].Uid,
			}
			if faPai[i].Uid == u.uid {
				handsMj[i].MjVal = faPai[i].MjVal
			}
		}
		data := &pb.MjFaPai{
			Fapai:  handsMj,
			Zhuang: zhuang.uid,
			Touzi:  r.touzi,
			Pizi:   piziMj,
			Laizi:  int32(r.laizi),
		}
		if zhuang.uid == u.uid {
			data.CanOp = u.CanOpSelf()
		}
		u.Send(def.GameFaPai, data)
	}
}

func (r *Room) MoPai() uint8 {
	pai := r.mj[r.mjIndex]
	r.mjIndex++
	log.Printf("MoPai index:%d val:%d\n", r.mjIndex-1, pai)
	return pai
}

// 从当前的位置往前找，存在没有操作的玩家，并且玩家可执行的操作大于当前操作，则继续等待
func (r *Room) SkipWaiting(u *User) bool {
	for op := u.wait_op + 1; op <= mj.HuPai; op++ {
		for seat := r.turn + 1; seat != u.seat; seat = (seat + 1) % 4 {
			user := r.GetUser(seat)
			if mj.HasOp(user.can_ops_flag, op) {
				return false
			}
		}
	}
	return true
}

func (r *Room) Waiting() bool {
	for _, u := range r.Users {
		if u.waiting {
			log.Printf("Waiting uid:%d wait_op:%d\n", u.uid, u.wait_op)
			return true
		}
	}
	return false
}

func (r *Room) getOpUser(user *User) *User {
	if r.waitOther {
		maxOp, distance := uint8(0), int(4)
		for _, u := range r.Users {
			if u.wait_op >= maxOp {
				maxOp = u.wait_op
			}
		}
		for i, u := range r.Users {
			if u.wait_op == maxOp {
				if dis := r.getDistance(i); dis < distance {
					distance = dis
					user = u
				}
			}
		}
	}
	return user
}

func (r *Room) getDistance(idx int) int {
	dis := idx - r.turn
	if dis < 0 {
		dis += 4
	}
	return dis
}

func (r *Room) MjOp(uid uint32, opt *pb.MjOpt) {
	log.Printf("MjOp uid:%d opt:%v\n", uid, opt)

	// 获取当前玩家
	currUser := r.GetUserByUID(uid)
	if currUser == nil {
		log.Printf("tap err, uid:%d not in room:%d\n", uid, r.roomId)
		return
	}
	// 验证当前是否为等待玩家
	if !currUser.waiting {
		log.Printf("tap err, uid:%d not waiting\n", uid)
		return
	}
	// 是否为可执行操作
	if !currUser.IsCanOp(opt.Op) {
		log.Printf("capop err, uid:%d cant op:%d\n", uid, opt.Op)
		return
	}
	// 执行操作
	if !currUser.DealMj(uint8(opt.Op), uint8(opt.Mj)) {
		log.Printf("deal err, uid:%d opt:%d mj:%d\n", uid, opt.Op, opt.Mj)
		return
	}

	r.history = append(r.history, &mj.MjOp{
		Uid: opt.Uid,
		Op:  opt.Op,
		Mj:  opt.Mj,
	})

	// 当前操作
	currUser.wait_op = uint8(opt.Op)
	currUser.waiting = false

	// 等待完毕
	if r.Waiting() {
		return
	}

	if !r.waitOther {
		r.MjOpSelf(uid, opt)
	} else {
		r.waitOther = false
		r.MjOpOther(uid, opt)
	}
}

func (r *Room) MjOpOther(uid uint32, opt *pb.MjOpt) {
	log.Printf("MjOpOther uid:%d opt:%v\n", uid, opt)
	currUser := r.GetUserByUID(uid)
	// 获取最佳操作玩家
	finalUser := r.getOpUser(currUser)
	finalOp := int32(finalUser.wait_op)

	// 出牌操作，没有人有操作，给下一家发牌，并告知可执行操作
	switch finalOp {
	case mj.GuoPai:
		// 黄庄操作，没有其他玩家可操作，且黄庄牌数大于等于牌数，游戏结束
		if r.haiDiLao() {
			return
		}
		r.turn = (r.turn + 1) % len(r.Users)
		turnUser := r.Users[r.turn]
		moPai := r.MoPai()
		turnUser.MoMj(moPai)
		r.waitOther = false
		for _, u := range r.Users {
			nextOpt := &pb.MjOpt{
				Op:  mj.MoPai,
				Uid: turnUser.uid,
			}
			if u == turnUser {
				nextOpt.Mj = int32(moPai)
				nextOpt.CanOp = u.CanOpSelf()
			}
			u.Send(def.BcOpt, nextOpt)
		}
	case mj.MGang:
		// 杠牌操作，通知其他玩家当前玩家杠牌
		for _, u := range r.Users {
			u.Send(def.BcOpt, opt)
		}

		// 当前玩家摸牌
		r.turn = finalUser.seat
		turnUser := r.Users[r.turn]
		moPai := r.MoPai()
		turnUser.MoMj(moPai)
		r.waitOther = false
		for _, u := range r.Users {
			nextOpt := &pb.MjOpt{
				Op:  mj.MoPai,
				Uid: finalUser.uid,
			}
			if u == turnUser {
				nextOpt.Mj = int32(moPai)
				nextOpt.CanOp = u.CanOpSelf()
			}
			u.Send(def.BcOpt, nextOpt)
		}
	case mj.Peng, mj.LChi, mj.MChi, mj.RChi: // 吃碰操作
		r.turn = finalUser.seat
		turnUser := r.Users[r.turn]
		r.waitOther = false
		for _, u := range r.Users {
			nextOpt := &pb.MjOpt{
				Op:  finalOp,
				Uid: finalUser.uid,
				Mj:  opt.Mj,
			}
			if u == turnUser {
				nextOpt.CanOp = u.CanOpSelf()
			}
			u.Send(def.BcOpt, nextOpt)
		}
	case mj.HuPai: // 胡牌操作
		r.huPai(finalUser)
	}
}

func (r *Room) MjOpSelf(uid uint32, opt *pb.MjOpt) {
	log.Printf("MjOpSelf uid:%d opt:%v\n", uid, opt)
	// 如果是出牌操作，告知其他玩家可执行的操作
	switch opt.Op {
	case mj.ChuPai, mj.BGang:
		// 检查是否有玩家可操作
		noCanOp := true
		for _, u := range r.Users {
			opt.CanOp = 0
			if u.uid != uid {
				// 如果是出牌操作，告知其他玩家可执行的操作
				opt.CanOp = u.CanOpOther(uint8(opt.Mj), uint8(opt.Op), r.laizi)
				if opt.CanOp > 0 {
					// 其他玩家可操作，继续等待
					noCanOp = false
					r.waitOther = true
					log.Printf("uid:%d canop:%d\n", u.uid, u.CanOp())
				}
			}
			u.Send(def.BcOpt, opt)
		}
		log.Printf("noCanOp:%t", noCanOp)
		if noCanOp {
			// 没有可操作的玩家，给下一家发牌，并告知可执行操作
			if opt.Op == mj.ChuPai {
				r.turn = (r.turn + 1) % len(r.Users)
			}
			// 如果是补杠操作，给当前玩家发牌，并告知可执行操作
			turnUser := r.Users[r.turn]

			// 给当前玩家发牌
			moPai := r.MoPai()
			turnUser.MoMj(moPai)
			r.waitOther = false

			// 通知其他玩家当前玩家出牌
			for _, u := range r.Users {
				temp := &pb.MjOpt{
					Op:  mj.MoPai,
					Uid: turnUser.uid,
				}
				if u == turnUser {
					temp.Mj = int32(moPai)
					temp.CanOp = u.CanOpSelf()
				}
				u.Send(def.BcOpt, temp)
			}
		}
		return
	case mj.AGang:
		// 暗杆操作，通知其他玩家当前玩家暗杆
		for _, u := range r.Users {
			u.Send(def.BcOpt, opt)
		}

		// 给当前玩家发牌
		turnUser := r.Users[r.turn]
		moPai := r.MoPai()
		turnUser.MoMj(moPai)
		r.waitOther = false

		// 通知其他玩家当前玩家出牌
		for _, u := range r.Users {
			temp := &pb.MjOpt{
				Op:  mj.MoPai,
				Uid: turnUser.uid,
			}
			if u == turnUser {
				temp.Mj = int32(moPai)
				temp.CanOp = u.CanOpSelf()
			}
			u.Send(def.BcOpt, temp)
		}
	case mj.HuPai:
		turnUser := r.Users[r.turn]
		// 胡牌操作，通知其他玩家当前玩家胡牌
		if turnUser.Zimo() {
			for _, u := range r.Users {
				u.Send(def.BcOpt, opt)
			}
		} else {
			log.Printf("tap err, uid:%d not zimo\n", uid)
		}
	}
}

func (r *Room) getLatestOp(Op int32) *mj.MjOp {
	for i := len(r.history) - 1; i >= 0; i-- {
		if Op&r.history[i].Op > 0 {
			return r.history[i]
		}
	}
	return nil
}

func (r *Room) haiDiLao() bool {
	if r.huangZhuang+int(r.mjIndex) >= len(r.mj) {
		settle := &pb.GameOver{}
		for _, u := range r.Users {
			userSettle := pb.GameOverUser{
				Uid:   u.uid,
				Hands: u.Mj(),
			}
			settle.Users = append(settle.Users, &userSettle)
		}
		r.SendAll(def.GameOver, settle)
		return true
	}
	return false
}

func (r *Room) huPai(huUser *User) {
	log.Println("game over")
	pai := uint8(0)
	if r.waitOther {
		if opt := r.getLatestOp(mj.ChuPai | mj.BGang); opt != nil {
			pai = uint8(opt.Mj)
		}
	}
	settle := &pb.GameOver{}
	ht, hs := huUser.HuPai(pai, r.laizi)
	win := int64(0)
	for _, u := range r.Users {
		if u == huUser {
			continue
		}
		fan := u.FanShu()
		lose := int64(hs) * int64(fan)
		win += lose
		userSettle := pb.GameOverUser{
			Uid:   u.uid,
			Win:   -lose,
			Hands: u.Mj(),
		}
		settle.Users = append(settle.Users, &userSettle)
	}
	settle.Users = append(settle.Users, &pb.GameOverUser{
		Uid:    huUser.uid,
		Win:    win,
		Hands:  huUser.Mj(),
		HuType: ht,
	})
	r.SendAll(def.GameOver, settle)
	r.playing = false
}

func (r *Room) SendOther(uid uint32, cmd uint16, data proto.Message) {
	for _, u := range r.Users {
		if u.uid != uid {
			u.Send(cmd, data)
		}
	}
}

func (r *Room) SendAll(cmd uint16, data proto.Message) {
	for _, u := range r.Users {
		u.Send(cmd, data)
	}
}
