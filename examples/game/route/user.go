package route

import (
	"frbg/core"
	"frbg/def"
	"frbg/examples/pb"
	"frbg/mj"
	"log"
	"sort"

	"google.golang.org/protobuf/proto"
)

type User struct {
	l           *Local
	uid         uint32
	gateId      uint16 // 网关ID
	pai         int32
	offline     bool
	canOpsGroup []*mj.Group
	waiting     bool       // 是否等待操作
	waitOp      uint8      // 等待期间收集到的操作
	mjHands     []uint8    // 麻将
	mjHistory   []uint8    // 出牌
	mjGroup     []mj.Group // 麻将组
	seat        int
	canOpsFlag  int32
	huType      int32
	lastOp      mj.Group // 上一次操作
	prepare     bool     // 准备状态
}

func (u *User) Seat() int {
	return u.seat
}

func (u *User) Send(cmd uint16, data proto.Message) {
	svid := core.Svid(def.ST_Gate, uint8(u.gateId))
	u.l.SendTo(svid, u.uid, cmd, data)
}

func (u *User) Reset() {
	u.pai = 0
	u.prepare = true
	u.mjHands = u.mjHands[:0]
	u.mjHistory = u.mjHistory[:0]
	u.mjGroup = u.mjGroup[:0]
	u.canOpsGroup = u.canOpsGroup[:0]
	u.waiting = false
	u.waitOp = 0
}

func (u *User) removeMj(val uint8, num int) bool {
	tail := len(u.mjHands) - 1
	for i := 0; i <= tail; {
		if u.mjHands[i] == val {
			u.mjHands[i], u.mjHands[tail] = u.mjHands[tail], u.mjHands[i]
			num--
			if num == 0 {
				u.mjHands = u.mjHands[:tail]
				log.Printf("uid:%d remove_mj %v %d", u.uid, u.Mj(), val)
				return true
			}
			tail--
		} else {
			i++
		}
	}
	log.Printf("uid:%d remove_mj %v %d", u.uid, u.Mj(), val)
	return false
}

// 打麻将
func (u *User) DaMj(val uint8) bool {
	return u.removeMj(val, 1)
}

// 摸麻将
func (u *User) MoMj(val ...uint8) {
	u.lastOp = mj.Group{
		Op:  mj.MoPai,
		Val: val[0],
	}
	u.canOpsFlag = 0
	u.mjHands = append(u.mjHands, val...)
	log.Printf("uid:%d MoMj %v", u.uid, val)
}

func (u *User) Haidilao(val ...uint8) {
	u.lastOp = mj.Group{
		Op:  mj.HdlPai,
		Val: val[0],
	}
	u.canOpsFlag = 0
	u.mjHands = append(u.mjHands, val...)
	log.Printf("uid:%d Haidilao %v", u.uid, val)
}

// 左吃麻将
func (u *User) LChiMj(val uint8) bool {
	val1, val2 := val+1, val+2
	if !u.removeMj(val1, 1) {
		return false
	}
	if !u.removeMj(val2, 1) {
		return false
	}
	u.mjGroup = append(u.mjGroup, mj.Group{Op: mj.LChi, Val: val})
	log.Printf("uid:%d LChiMj %v %v %v", u.uid, mj.Pai(val), mj.Pai(val+1), mj.Pai(val+2))
	return true
}

// 中吃麻将
func (u *User) MChiMj(val uint8) bool {
	val1, val2 := val-1, val+1
	if !u.removeMj(val1, 1) {
		return false
	}
	if !u.removeMj(val2, 1) {
		return false
	}
	u.mjGroup = append(u.mjGroup, mj.Group{Op: mj.MChi, Val: val})
	log.Printf("uid:%d MChiMj %v %v %v", u.uid, mj.Pai(val), mj.Pai(val-1), mj.Pai(val+1))
	return true
}

// 右吃麻将
func (u *User) RChiMj(val uint8) bool {
	val1, val2 := val-1, val-2
	if !u.removeMj(val1, 1) {
		return false
	}
	if !u.removeMj(val2, 1) {
		return false
	}
	u.mjGroup = append(u.mjGroup, mj.Group{Op: mj.RChi, Val: val})
	log.Printf("uid:%d RChiMj %v %v %v", u.uid, mj.Pai(val), mj.Pai(val-1), mj.Pai(val-2))
	return true
}

// 碰牌
func (u *User) PengMj(val uint8) bool {
	cnt := 0
	for _, v := range u.mjHands {
		if v == val {
			cnt++
		}
	}
	if cnt < 2 {
		return false
	}

	if !u.removeMj(val, 2) {
		return false
	}
	u.mjGroup = append(u.mjGroup, mj.Group{Op: mj.Peng, Val: val})
	log.Printf("uid:%d PengMj %v %v %v", u.uid, mj.Pai(val), mj.Pai(val), mj.Pai(val))
	return true
}

// 明杠
func (u *User) MGangMj(val uint8) bool {
	cnt := 0
	for _, v := range u.mjHands {
		if v == val {
			cnt++
		}
	}
	if cnt < 3 {
		return false
	}

	if !u.removeMj(val, 3) {
		return false
	}
	u.mjGroup = append(u.mjGroup, mj.Group{Op: mj.MGang, Val: val})
	log.Printf("uid:%d MGangMj %v %v %v %v", u.uid, mj.Pai(val), mj.Pai(val), mj.Pai(val), mj.Pai(val))
	return true
}

// 补杠
func (u *User) BGangMj(val uint8) bool {
	for _, v := range u.mjGroup {
		if v.Op == mj.Peng && v.Val == val {
			v.Op = mj.BGang
		}
	}

	if !u.removeMj(val, 1) {
		return false
	}

	log.Printf("uid:%d BGangMj %v %v %v %v", u.uid, mj.Pai(val), mj.Pai(val), mj.Pai(val), mj.Pai(val))
	return true
}

// 暗杠
func (u *User) AGangMj(val uint8) bool {
	cnt := 0
	for _, v := range u.mjHands {
		if v == val {
			cnt++
		}
	}
	if cnt < 4 {
		return false
	}

	if !u.removeMj(val, 4) {
		return false
	}
	u.mjGroup = append(u.mjGroup, mj.Group{Op: mj.AGang, Val: val})
	log.Printf("uid:%d AGangMj %v %v %v %v", u.uid, mj.Pai(val), mj.Pai(val), mj.Pai(val), mj.Pai(val))
	return true
}

// 点炮
func (u *User) DianPao(val uint8, laizi uint8) bool {
	st := mj.Newlz(u.mjHands, val, laizi, u.mjGroup)
	log.Printf("uid:%d DianPao %v", u.uid, mj.Pai(val))
	return st.CanHu()
}

// 自摸
func (u *User) Zimo(laizi uint8) bool {
	st := mj.Newlz(u.mjHands, 0, laizi, u.mjGroup)
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
		ok = true
	}
	if !ok {
		return ok
	}
	u.canOpsFlag = 0
	u.lastOp.Op = op
	u.lastOp.Val = val
	return ok
}

// 手牌操作
func (u *User) CanOpSelf(lz uint8) int32 {
	if u.canOpsFlag > 0 {
		return u.canOpsFlag
	}
	st := mj.Newlz(u.mjHands, 0, lz, u.mjGroup)
	u.canOpsGroup = st.CanOpSelf()
	u.waiting = true

	// 可以出牌
	if u.lastOp.Op != mj.HdlPai {
		mj.AddOp(&u.canOpsFlag, mj.ChuPai)
	}
	// 其他操作
	for i := range u.canOpsGroup {
		mj.AddOp(&u.canOpsFlag, u.canOpsGroup[i].Op)
	}

	return u.canOpsFlag
}

func (u *User) CanOps() []*pb.CanOp {
	var canOps []*pb.CanOp
	for _, g := range u.canOpsGroup {
		canOps = append(canOps, &pb.CanOp{
			Op: int32(g.Op),
			Mj: int32(g.Val),
		})
	}
	if mj.HasOp(u.canOpsFlag, mj.ChuPai) {
		canOps = append(canOps, &pb.CanOp{
			Op: int32(mj.ChuPai),
			Mj: int32(u.mjHands[0]),
		})
	}
	if mj.HasOp(u.canOpsFlag, mj.GuoPai) {
		canOps = append(canOps, &pb.CanOp{
			Op: int32(mj.GuoPai),
			Mj: int32(0),
		})
	}
	sort.Slice(canOps, func(i, j int) bool {
		return canOps[i].Op >= canOps[j].Op
	})
	return canOps
}

func (u *User) IsCanOp(op int32) bool {
	return mj.HasOp(u.canOpsFlag, uint8(op))
}

func (u *User) CanOp() int32 {
	return u.canOpsFlag
}

// 可操作其他玩家的牌
func (u *User) CanOpOther(val uint8, op uint8, lz uint8) int32 {
	st := mj.Newlz(u.mjHands, val, lz, nil)
	u.canOpsGroup = st.CanOpOther(val, op)
	u.waiting = len(u.canOpsGroup) > 0

	u.canOpsFlag = int32(0)
	if u.waiting {
		mj.AddOp(&u.canOpsFlag, mj.GuoPai)
	}
	log.Printf("uid:%d mj:%v", u.uid, u.Mj())
	for i := range u.canOpsGroup {
		log.Printf("canop:%d", u.canOpsGroup[i].Op)
		mj.AddOp(&u.canOpsFlag, u.canOpsGroup[i].Op)
	}

	return u.canOpsFlag
}

func (u *User) HuPai(pai, laizi uint8) (int32, int32) {
	stlz := mj.Newlz(u.mjHands, pai, laizi, u.mjGroup)
	return stlz.HuPai()
}

func (u *User) FanShu() int32 {
	f := int32(1)
	for i := range u.mjGroup {
		if u.mjGroup[i].Op&(mj.MGang|mj.AGang|mj.BGang) > 0 {
			f <<= 1
		}
	}
	return f
}

func (u *User) Mj() []int32 {
	mj := make([]int32, len(u.mjHands))
	for i := range u.mjHands {
		mj[i] = int32(u.mjHands[i])
	}
	return mj
}

func (u *User) GroupString() string {
	str := ""
	for i := range u.mjGroup {
		str += " " + u.mjGroup[i].String()
	}
	return str
}
