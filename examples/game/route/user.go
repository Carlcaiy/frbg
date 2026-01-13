package route

import (
	"frbg/codec"
	"frbg/core"
	"frbg/def"
	"frbg/examples/pb"
	"frbg/mj"
	"log"

	"google.golang.org/protobuf/proto"
)

type User struct {
	l             *Local
	uid           uint32
	gateId        uint16 // 网关ID
	pai           int32
	offline       bool
	can_ops_group []*mj.Group
	waiting       bool       // 是否等待操作
	wait_op       uint8      // 等待期间收集到的操作
	mj_hands      []uint8    // 麻将
	mj_history    []uint8    // 出牌
	mj_group      []mj.Group // 麻将组
	seat          int
	can_ops_flag  int32
	hu_type       int32
	last_op       uint8 // 上一次操作
	prepare       bool  // 准备状态
}

func (u *User) Seat() int {
	return u.seat
}

func (u *User) Send(cmd uint16, data proto.Message) {
	payload, err := proto.Marshal(data)
	if err != nil {
		log.Printf("Send proto.Marshal() err:%s", err.Error())
		return
	}
	msg := codec.NewMessage(def.PacketOut, &pb.PacketOut{
		Uid:     u.uid,
		Cmd:     uint32(cmd),
		Payload: payload,
	})
	svid := core.Svid(def.ST_Gate, uint8(u.gateId))
	log.Printf("Send uid:%d cmd:%d svid:%d", u.uid, cmd, svid)
	if conn := u.l.Poll.GetServer(svid); conn != nil {
		conn.Write(msg)
	}
}

func (u *User) Reset() {
	u.pai = 0
	u.prepare = true
	u.mj_hands = u.mj_hands[:0]
	u.mj_history = u.mj_history[:0]
	u.mj_group = u.mj_group[:0]
	u.can_ops_group = u.can_ops_group[:0]
}

func (u *User) remove_mj(val uint8, num int) bool {
	tail := len(u.mj_hands) - 1
	for i, v := range u.mj_hands {
		if v == val {
			u.mj_hands[i], u.mj_hands[tail] = u.mj_hands[tail], u.mj_hands[i]
			tail--
			num--
			if num == 0 {
				u.mj_hands = u.mj_hands[:tail-num]
				return true
			}
		}
	}
	log.Printf("uid:%d remove_mj %v %d", u.uid, u.Mj(), num)
	return false
}

// 打麻将
func (u *User) DaMj(val uint8) bool {
	u.can_ops_flag = 0
	return u.remove_mj(val, 1)
}

// 摸麻将
func (u *User) MoMj(val ...uint8) {
	u.can_ops_flag = 0
	u.mj_hands = append(u.mj_hands, val...)
	pai := []string{}
	for _, v := range val {
		pai = append(pai, mj.Pai(v))
	}
	if len(val) == 1 {
		log.Printf("uid:%d MoMj %v", u.uid, pai)
	}
}

// 左吃麻将
func (u *User) LChiMj(val uint8) bool {
	u.can_ops_flag = 0
	val1, val2 := val+1, val+2
	if !u.remove_mj(val1, 1) {
		return false
	}
	if !u.remove_mj(val2, 1) {
		return false
	}
	u.mj_group = append(u.mj_group, mj.Group{Op: mj.LChi, Val: val})
	log.Printf("uid:%d LChiMj %v %v %v", u.uid, mj.Pai(val), mj.Pai(val+1), mj.Pai(val+2))
	return true
}

// 中吃麻将
func (u *User) MChiMj(val uint8) bool {
	u.can_ops_flag = 0
	val1, val2 := val-1, val+1
	if !u.remove_mj(val1, 1) {
		return false
	}
	if !u.remove_mj(val2, 1) {
		return false
	}
	u.mj_group = append(u.mj_group, mj.Group{Op: mj.MChi, Val: val})
	log.Printf("uid:%d MChiMj %v %v %v", u.uid, mj.Pai(val), mj.Pai(val-1), mj.Pai(val+1))
	return true
}

// 右吃麻将
func (u *User) RChiMj(val uint8) bool {
	u.can_ops_flag = 0
	val1, val2 := val-1, val-2
	if !u.remove_mj(val1, 1) {
		return false
	}
	if !u.remove_mj(val2, 1) {
		return false
	}
	u.mj_group = append(u.mj_group, mj.Group{Op: mj.RChi, Val: val})
	log.Printf("uid:%d RChiMj %v %v %v", u.uid, mj.Pai(val), mj.Pai(val-1), mj.Pai(val-2))
	return true
}

// 碰牌
func (u *User) PengMj(val uint8) bool {
	u.can_ops_flag = 0
	cnt := 0
	for _, v := range u.mj_hands {
		if v == val {
			cnt++
		}
	}
	if cnt < 2 {
		return false
	}

	if !u.remove_mj(val, 2) {
		return false
	}
	u.mj_group = append(u.mj_group, mj.Group{Op: mj.Peng, Val: val})
	log.Printf("uid:%d PengMj %v %v %v", u.uid, mj.Pai(val), mj.Pai(val), mj.Pai(val))
	return true
}

// 明杠
func (u *User) MGangMj(val uint8) bool {
	u.can_ops_flag = 0
	cnt := 0
	for _, v := range u.mj_hands {
		if v == val {
			cnt++
		}
	}
	if cnt < 3 {
		return false
	}

	if !u.remove_mj(val, 3) {
		return false
	}
	u.mj_group = append(u.mj_group, mj.Group{Op: mj.MGang, Val: val})
	log.Printf("uid:%d MGangMj %v %v %v %v", u.uid, mj.Pai(val), mj.Pai(val), mj.Pai(val), mj.Pai(val))
	return true
}

// 补杠
func (u *User) BGangMj(val uint8) bool {
	u.can_ops_flag = 0
	for _, v := range u.mj_group {
		if v.Op == mj.Peng && v.Val == val {
			v.Op = mj.BGang
		}
	}

	if !u.remove_mj(val, 1) {
		return false
	}

	log.Printf("uid:%d BGangMj %v %v %v %v", u.uid, mj.Pai(val), mj.Pai(val), mj.Pai(val), mj.Pai(val))
	return true
}

// 暗杠
func (u *User) AGangMj(val uint8) bool {
	u.can_ops_flag = 0
	cnt := 0
	for _, v := range u.mj_hands {
		if v == val {
			cnt++
		}
	}
	if cnt < 4 {
		return false
	}

	if !u.remove_mj(val, 4) {
		return false
	}
	u.mj_group = append(u.mj_group, mj.Group{Op: mj.AGang, Val: val})
	log.Printf("uid:%d AGangMj %v %v %v %v", u.uid, mj.Pai(val), mj.Pai(val), mj.Pai(val), mj.Pai(val))
	return true
}

// 点炮
func (u *User) DianPao(val uint8) bool {
	st := mj.New(u.mj_hands, val, u.mj_group)
	log.Printf("uid:%d DianPao %v", u.uid, mj.Pai(val))
	return st.CanHu()
}

// 自摸
func (u *User) Zimo() bool {
	st := mj.New(u.mj_hands, 0, u.mj_group)
	log.Printf("uid:%d Zimo", u.uid)
	return st.CanHu()
}

func (u *User) DealMj(op uint8, val uint8) bool {
	ok := false
	switch op {
	case mj.ChuPai:
		ok = u.DaMj(val)
	case mj.MGang:
		ok = u.MGangMj(val)
	case mj.BGang:
		ok = u.BGangMj(val)
	case mj.AGang:
		ok = u.AGangMj(val)
	case mj.Peng:
		ok = u.PengMj(val)
	case mj.LChi:
		ok = u.LChiMj(val)
	case mj.MChi:
		ok = u.MChiMj(val)
	case mj.RChi:
		ok = u.RChiMj(val)
	case mj.HuPai, mj.GuoPai:
		return true
	}
	if !ok {
		return ok
	}
	u.last_op = op
	return ok
}

// 手牌操作
func (u *User) CanOpSelf() int32 {
	if u.can_ops_flag > 0 {
		return u.can_ops_flag
	}
	st := mj.New(u.mj_hands, 0, nil)
	u.can_ops_group = st.CanOpSelf(u.last_op)
	u.waiting = true

	// 可以出牌
	mj.AddOp(&u.can_ops_flag, mj.ChuPai)
	// 其他操作
	for i := range u.can_ops_group {
		mj.AddOp(&u.can_ops_flag, u.can_ops_group[i].Op)
	}

	return u.can_ops_flag
}

func (u *User) IsCanOp(op int32) bool {
	return mj.HasOp(u.can_ops_flag, uint8(op))
}

func (u *User) CanOp() int32 {
	return u.can_ops_flag
}

// 可操作其他玩家的牌
func (u *User) CanOpOther(val uint8, op uint8, lz uint8) int32 {
	st := mj.Newlz(u.mj_hands, val, lz, nil)
	u.can_ops_group = st.CanOpOther(val, op)
	u.waiting = len(u.can_ops_group) > 0

	u.can_ops_flag = int32(0)
	if u.waiting {
		mj.AddOp(&u.can_ops_flag, mj.GuoPai)
	}
	for i := range u.can_ops_group {
		log.Printf("uid:%d CanOpOther %v %v %v", u.uid, mj.Pai(val), mj.Pai(lz), mj.Pai(u.can_ops_group[i].Val))
		mj.AddOp(&u.can_ops_flag, u.can_ops_group[i].Op)
	}

	return u.can_ops_flag
}

func (u *User) HuPai(pai, laizi uint8) (int32, int32) {
	stlz := mj.Newlz(u.mj_hands, pai, laizi, u.mj_group)
	return stlz.HuPai()
}

func (u *User) FanShu() int32 {
	f := int32(1)
	for i := range u.mj_group {
		if u.mj_group[i].Op&(mj.MGang|mj.AGang|mj.BGang) > 0 {
			f <<= 1
		}
	}
	return f
}

func (u *User) Mj() []int32 {
	mj := make([]int32, len(u.mj_hands))
	for i := range u.mj_hands {
		mj[i] = int32(u.mj_hands[i])
	}
	return mj
}
