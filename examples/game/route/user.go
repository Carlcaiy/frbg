package route

import (
	"frbg/mj"
)

type User struct {
	uid        uint32
	gateId     uint8 // 网关ID
	pai        int32
	offline    bool
	can_ops    []*mj.Group
	waiting    bool       // 是否等待操作
	wait_op    uint8      // 等待期间收集到的操作
	mj_hands   []uint8    // 麻将
	mj_history []uint8    // 出牌
	mj_group   []mj.Group // 麻将组
}

func (u *User) Reset() {
	u.mj_hands = u.mj_hands[:0]
	u.mj_history = u.mj_history[:0]
	u.mj_group = u.mj_group[:0]
}

func (u *User) remove_mj(val uint8, num int) bool {
	tail := len(u.mj_hands)
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
	return false
}

// 打麻将
func (u *User) DaMj(val uint8) bool {
	return u.remove_mj(val, 1)
}

// 摸麻将
func (u *User) MoMj(val uint8) {
	u.mj_hands = append(u.mj_hands, val)
}

// 左吃麻将
func (u *User) LChiMj(val uint8) bool {
	val1, val2 := val+1, val+2
	if !u.remove_mj(val1, 1) {
		return false
	}
	if !u.remove_mj(val2, 1) {
		return false
	}
	u.mj_group = append(u.mj_group, mj.Group{Op: mj.LChi, Val: val})
	return true
}

// 中吃麻将
func (u *User) MChiMj(val uint8) bool {
	val1, val2 := val-1, val+2
	if !u.remove_mj(val1, 1) {
		return false
	}
	if !u.remove_mj(val2, 1) {
		return false
	}
	u.mj_group = append(u.mj_group, mj.Group{Op: mj.MChi, Val: val})
	return true
}

// 右吃麻将
func (u *User) RChiMj(val uint8) bool {
	val1, val2 := val-1, val-2
	if !u.remove_mj(val1, 1) {
		return false
	}
	if !u.remove_mj(val2, 1) {
		return false
	}
	u.mj_group = append(u.mj_group, mj.Group{Op: mj.RChi, Val: val})
	return true
}

// 碰牌
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

	if !u.remove_mj(val, 2) {
		return false
	}
	u.mj_group = append(u.mj_group, mj.Group{Op: mj.Peng, Val: val})
	return true
}

// 明杠
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

	if !u.remove_mj(val, 3) {
		return false
	}
	u.mj_group = append(u.mj_group, mj.Group{Op: mj.MGang, Val: val})
	return true
}

// 补杠
func (u *User) BGangMj(val uint8) bool {
	for _, v := range u.mj_group {
		if v.Op == mj.Peng && v.Val == val {
			v.Op = mj.BGang
		}
	}
	if !u.remove_mj(val, 1) {
		return false
	}
	return true
}

// 暗杠
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

	if !u.remove_mj(val, 4) {
		return false
	}
	u.mj_group = append(u.mj_group, mj.Group{Op: mj.AGang, Val: val})
	return true
}

// 点炮
func (u *User) DianPao(val uint8) bool {
	st := mj.New(u.mj_hands, val, u.mj_group)
	return st.CanHu()
}

// 自摸
func (u *User) Zimo() bool {
	st := mj.New(u.mj_hands, 0, u.mj_group)
	return st.CanHu()
}

// 手牌操作
func (u *User) CanOpSelf() int32 {
	st := mj.New(u.mj_hands, 0, nil)
	u.can_ops = st.CanOpSelf()

	op := int32(mj.DaPai)
	for i := range u.can_ops {
		op |= 1 << (u.can_ops[i].Op - 1)
	}

	return op
}

// 可操作其他玩家的牌
func (u *User) CanOpOther(val uint8, op uint8, lz uint8) int32 {
	st := mj.Newlz(u.mj_hands, val, lz, nil)
	u.can_ops = st.CanOpOther(val, op)
	u.waiting = len(u.can_ops) > 0

	can_op := int32(0)
	for i := range u.can_ops {
		can_op |= 1 << (u.can_ops[i].Op - 1)
	}

	return can_op
}
