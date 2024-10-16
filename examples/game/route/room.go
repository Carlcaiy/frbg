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
	l       *Local
	hall    *network.Conn
	hallId  uint8
	roomId  uint32
	tempId  uint32
	Users   []*User // 用户
	turn    int     // 庄家
	mj      []uint8 // 麻将
	mjIndex int16   // 麻将索引
	last    int32
	touzi   []int32 // 骰子
	pizi    uint8   // 皮子
	laizi   uint8   // 赖子
	history []mj.MjOp
}

func NewRoom(l *Local) *Room {
	return &Room{
		l:     l,
		touzi: make([]int32, 2),
	}
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
				faPai = append(faPai, &proto.DeskMj{
					Index: int32(r.mjIndex),
					Uid:   u.uid,
					MjVal: int32(r.mj[r.mjIndex]),
				})
				r.mjIndex++
			}
		}
	}
	for i := 0; i < 5; i++ {
		u := r.Users[(r.turn+i)%4]
		faPai = append(faPai, &proto.DeskMj{
			Index: int32(r.mjIndex),
			Uid:   u.uid,
			MjVal: int32(r.mj[r.mjIndex]),
		})
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

	// 当前回合
	zhuang := r.Users[r.turn]
	bs, _ := parser.Pack(zhuang.uid, def.ST_User, cmd.GameFaPai, &proto.FaMj{
		Fapai:  faPai,
		Zhuang: zhuang.uid,
		Touzi:  r.touzi,
		Pizi: &proto.DeskMj{
			Index: piziIndex,
			MjVal: int32(r.pizi),
		},
		Laizi: int32(r.laizi),
	})
	r.l.SendToGate(zhuang.gateId, bs)
}

func (r *Room) MoPai() uint8 {
	pai := r.mj[r.mjIndex]
	r.mjIndex++
	return pai
}

func (r *Room) Waiting() bool {
	for _, u := range r.Users {
		if u.waiting {
			return true
		}
	}
	return false
}

func (r *Room) getOpUser() *User {
	var user *User
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
	// 验证当前出牌的玩家
	u := r.Users[r.turn]
	if uid != u.uid {
		log.Printf("tap err, uid:%d should uid:%d\n", uid, u.uid)
		return
	}

	op := opt.Op
	if u.waiting {
		u.wait_op = uint8(op)
		u.waiting = false
	}
	if r.Waiting() {
		return
	}
	user := r.getOpUser()
	op = int32(user.wait_op)

	pai := opt.Mj
	// 验证麻将
	if op == mj.DaPai && !u.DaMj(uint8(pai)) {
		log.Printf("Op:%d err, uid:%d pai:%d not found\n", op, uid, pai)
		return
	} else if op == mj.LChi && !u.LChiMj(uint8(pai)) {
		log.Printf("Op:%d err, uid:%d pai:%d not found\n", op, uid, pai)
		return
	} else if op == mj.MChi && !u.MChiMj(uint8(pai)) {
		log.Printf("Op:%d err, uid:%d pai:%d not found\n", op, uid, pai)
		return
	} else if op == mj.RChi && !u.RChiMj(uint8(pai)) {
		log.Printf("Op:%d err, uid:%d pai:%d not found\n", op, uid, pai)
		return
	} else if op == mj.Peng && !u.PengMj(uint8(pai)) {
		log.Printf("Op:%d err, uid:%d pai:%d not found\n", op, uid, pai)
		return
	} else if op == mj.MGang && !u.MGangMj(uint8(pai)) {
		log.Printf("Op:%d err, uid:%d pai:%d not found\n", op, uid, pai)
		return
	} else if op == mj.BGang && !u.BGangMj(uint8(pai)) {
		log.Printf("Op:%d err, uid:%d pai:%d not found\n", op, uid, pai)
		return
	} else if op == mj.AGang && !u.AGangMj(uint8(pai)) {
		log.Printf("Op:%d err, uid:%d pai:%d not found\n", op, uid, pai)
		return
	}

	// 广播操作
	noCanOp := true
	if op != mj.GuoPai {
		for _, u := range r.Users {
			// 如果是出牌操作，告知其他玩家可执行的操作
			if op == mj.DaPai || op == mj.BGang {
				canOp := u.CanOpOther(uint8(pai), uint8(op), r.laizi)
				if canOp > 0 {
					noCanOp = false
					opt.CanOp = canOp
				}
			}
			bs, _ := parser.Pack(u.uid, def.ST_User, cmd.Opt, opt)
			r.l.SendToGate(u.gateId, bs)
		}
	}

	// 出牌操作，没有人有操作，给下一家发牌，并告知可执行操作
	if (op == mj.DaPai && noCanOp) ||
		op == mj.GuoPai || op == mj.AGang || op == mj.MGang {
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
