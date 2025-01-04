package route

import (
	"frbg/def"
	"frbg/examples/cmd"
	"frbg/examples/proto"
	"frbg/mj"
	"frbg/network"
	"frbg/parser"
	"log"
	"math/rand"
)

type Room struct {
	l         *Local
	hall      *network.Conn
	hallId    uint8
	roomId    uint32
	tempId    uint32
	Users     []*User // 用户
	turn      int     // 庄家
	mj        []uint8 // 麻将
	mjIndex   int16   // 麻将索引
	touzi     []int32 // 骰子
	pizi      uint8   // 皮子
	laizi     uint8   // 赖子
	waitOther bool    // 等待其他玩家操作
	history   []*mj.MjOp
}

func NewRoom(l *Local) *Room {
	return &Room{
		l:     l,
		touzi: make([]int32, 2),
	}
}

func (r *Room) GetUserByUID(uid uint32) *User {
	for _, u := range r.Users {
		if u.uid == uid {
			return u
		}
	}
	return nil
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
		u.pai = 0
	}
	rand.Shuffle(len(r.mj), func(i, j int) {
		r.mj[i], r.mj[j] = r.mj[j], r.mj[i]
	})
	r.mjIndex = 0
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

	faPai := make([]*proto.DeskMj, 0, 4*3+4+1)
	// 每个玩家发3轮
	for t := 0; t < 3; t++ {
		// 4个玩家，从庄家开始
		for i := 0; i < 4; i++ {
			u := r.Users[(r.turn+i)%4]
			// 每个玩家发4个马建
			for j := 0; j < 4; j++ {
				mjVal := r.mj[r.mjIndex]
				faPai = append(faPai, &proto.DeskMj{
					Index: int32(r.mjIndex),
					Uid:   u.uid,
					MjVal: int32(mjVal),
				})
				r.mjIndex++
				u.MoMj(mjVal)
			}
		}
	}
	for i := 0; i < 5; i++ {
		u := r.Users[(r.turn+i)%4]
		mjVal := r.mj[r.mjIndex]
		faPai = append(faPai, &proto.DeskMj{
			Index: int32(r.mjIndex),
			Uid:   u.uid,
			MjVal: int32(r.mj[r.mjIndex]),
		})
		r.mjIndex++
		u.MoMj(mjVal)
	}

	// 确定赖子
	r.touzi = []int32{rand.Int31n(6) + 1, rand.Int31n(6) + 1}
	col := r.touzi[1]
	if r.touzi[0] < r.touzi[1] {
		col = r.touzi[0]
	}
	piziIndex := (9*3+8)*2 - col*2
	r.pizi = r.mj[piziIndex]
	r.laizi = mj.GetLaizi(r.pizi)

	// 组装信息
	zhuang := r.Users[r.turn]
	can_op := zhuang.CanOpSelf()

	piziMj := &proto.DeskMj{
		Index: piziIndex,
		MjVal: int32(r.pizi),
	}

	r.waitOther = false
	for _, u := range r.Users {
		handsMj := make([]*proto.DeskMj, len(faPai))
		for i := range faPai {
			handsMj[i] = &proto.DeskMj{
				Index: faPai[i].Index,
				Uid:   faPai[i].Uid,
			}
			if faPai[i].Uid == u.uid {
				handsMj[i].MjVal = faPai[i].MjVal
			}
		}
		canOp := int32(0)
		if u.uid == zhuang.uid {
			canOp = can_op
		}
		bs, _ := parser.Pack(zhuang.uid, def.ST_User, cmd.GameFaPai, &proto.FaMj{
			Fapai:  handsMj,
			Zhuang: zhuang.uid,
			Touzi:  r.touzi,
			Pizi:   piziMj,
			Laizi:  int32(r.laizi),
			CanOp:  canOp,
		})
		r.l.SendToGate(zhuang.gateId, bs)
	}
}

func (r *Room) MoPai() uint8 {
	pai := r.mj[r.mjIndex]
	r.mjIndex++
	return pai
}

// 从当前的位置往前找，存在没有操作的玩家，并且玩家可执行的操作大于当前操作，则继续等待
func (r *Room) SkipWaiting(u *User) bool {
	for op := u.wait_op + 1; op <= mj.HuPai; op++ {
		for seat := r.turn + 1; seat != u.seat; seat = (seat + 1) % 4 {
			user := r.GetUser(seat)
			if user.can_ops_flag&mj.OpBit(op) > 0 {
				return false
			}
		}
	}
	return true
}

func (r *Room) Waiting() bool {
	for _, u := range r.Users {
		if u.waiting {
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

func (r *Room) MjOp(uid uint32, opt *proto.MjOpt) {
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
	if !currUser.CanOp(opt.Op) {
		log.Printf("tap err, uid:%d cant op:%d\n", uid, opt.Op)
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

	// 获取最佳操作玩家
	finalUser := r.getOpUser(currUser)
	finalOp := int32(finalUser.wait_op)
	pai := opt.Mj

	// 广播操作
	noCanOp := true
	if finalOp != mj.GuoPai {
		for _, u := range r.Users {
			// 如果是出牌操作，告知其他玩家可执行的操作
			if finalOp == mj.DaPai || finalOp == mj.BGang {
				canOp := u.CanOpOther(uint8(pai), uint8(finalOp), r.laizi)
				if canOp > 0 {
					noCanOp = false
					opt.CanOp = canOp
				}
			}
			bs, _ := parser.Pack(u.uid, def.ST_User, cmd.Opt, opt)
			r.l.SendToGate(u.gateId, bs)
		}
	}
	// 如果有其他人可以操作，等待其他玩家操作
	r.waitOther = !noCanOp

	// 出牌操作，没有人有操作，给下一家发牌，并告知可执行操作
	if (finalOp == mj.DaPai && noCanOp) ||
		finalOp == mj.GuoPai || finalOp == mj.AGang || finalOp == mj.MGang {
		r.turn = (r.turn + 1) % len(r.Users)
		turnUser := r.Users[r.turn]
		moPai := r.MoPai()
		turnUser.MoMj(moPai)

		for _, u := range r.Users {
			opt := &proto.MjOpt{
				Op:  mj.MoPai,
				Uid: uid,
			}
			if u == turnUser {
				opt.Mj = int32(moPai)
				opt.CanOp = u.CanOpSelf()
			}
			bs, _ := parser.Pack(u.uid, def.ST_User, cmd.GameOpt, opt)
			r.l.SendToGate(u.gateId, bs)
		}
	}

	// 胡牌操作
	if finalOp == mj.HuPai {
		r.gameOver(finalUser)
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

func (r *Room) gameOver(huUser *User) {
	log.Println("game over")
	pai := uint8(0)
	if r.waitOther {
		if opt := r.getLatestOp(mj.DaPai | mj.BGang); opt != nil {
			pai = uint8(opt.Mj)
		}
	}
	settle := &proto.GameOver{}
	ht, hs := huUser.HuPai(pai, r.laizi)
	win := int64(0)
	for _, u := range r.Users {
		if u == huUser {
			continue
		}
		fan := u.FanShu()
		lose := int64(hs) * int64(fan)
		win += lose
		userSettle := proto.GameOverUser{
			Uid:   u.uid,
			Win:   -lose,
			Hands: u.Mj(),
		}
		settle.Users = append(settle.Users, &userSettle)
	}
	settle.Users = append(settle.Users, &proto.GameOverUser{
		Uid:    huUser.uid,
		Win:    win,
		Hands:  huUser.Mj(),
		HuType: ht,
	})
	buf, _ := parser.Pack(0, def.ST_User, cmd.GameOver, settle)
	r.SendAll(buf)
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
