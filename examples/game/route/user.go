package route

import (
	"frbg/mj"
)

type User struct {
	uid        uint32
	gateId     uint8 // 网关ID
	tap        int32
	offline    bool
	can_op     []*mj.Group
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

func (u *User) LChiMj(val uint8) {
	val1, val2 := val+1, val+2
	u.remove_mj(val1, 1)
	u.remove_mj(val2, 1)
	u.mj_group = append(u.mj_group, mj.Group{Op: mj.LChi, Val: val})
}

func (u *User) MChiMj(val uint8) {
	val1, val2 := val-1, val+2
	u.remove_mj(val1, 1)
	u.remove_mj(val2, 1)
	u.mj_group = append(u.mj_group, mj.Group{Op: mj.MChi, Val: val})
}

func (u *User) RChiMj(val uint8) {
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
	st := mj.New(u.mj_hands, val, u.mj_group)
	return st.CanHu()
}

func (u *User) Zimo() bool {
	st := mj.New(u.mj_hands, 0, u.mj_group)
	return st.CanHu()
}

func (u *User) CanOpSelf() {
	st := mj.New(u.mj_hands, 0, nil)
	u.can_op = st.CanOpSelf()
}

func (u *User) CanOpOther(val uint8) {
	st := mj.New(u.mj_hands, val, nil)
	u.can_op = st.CanOpOther(val)
}
