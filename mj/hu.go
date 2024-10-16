package mj

import "sort"

type stat struct {
	pai   uint8   // 点炮的牌 >0表示点炮 =0表示自摸
	val   []uint8 // 手牌牌值集合
	num   []int8  // 手牌数量集合
	Group []Group // 手牌将刻顺
}

func New(mj []uint8, p uint8, g []Group) *stat {
	if p > 0 {
		mj = append(mj, p)
	}
	sort.Slice(mj, func(i, j int) bool { return mj[i] < mj[j] })
	r := &stat{
		pai:   p,
		Group: g,
	}
	tail := uint8(0)
	idx := -1
	for i := range mj {
		if mj[i] == tail {
			r.num[idx]++
		} else {
			idx++
			tail = mj[i]
			r.val = append(r.val, tail)
			r.num = append(r.num, 1)
		}
	}
	return r
}

func (m *stat) next(i int) int {
	for i < len(m.num) && m.num[i] == 0 {
		i++
	}
	return i
}

func (m *stat) deal_kezi(i int) bool {
	if i >= len(m.num) {
		return true
	}
	if m.num[i] >= 3 {
		m.num[i] -= 3
		m.Group = append(m.Group, Group{
			Val: m.val[i],
			Op:  Ke,
		})
		j := m.next(i)
		if m.deal_kezi(j) || m.deal_shunzi(j) {
			m.Group = m.Group[:len(m.Group)-1]
			m.num[i] += 3
			return true
		} else {
			m.Group = m.Group[:len(m.Group)-1]
			m.num[i] += 3
			return false
		}
	}
	return false
}

func (m *stat) deal_shunzi(i int) bool {
	if i >= len(m.num) {
		return true
	}
	if i+2 >= len(m.num) {
		return false
	}
	if m.val[i] < Tiao1 {
		return false
	}
	// 牌数和牌值一致
	if m.num[i] >= 1 && m.num[i+1] >= 1 && m.num[i+2] >= 1 &&
		m.val[i]+1 == m.val[i+1] && m.val[i+1]+1 == m.val[i+2] {
		m.Group = append(m.Group, Group{
			Val: m.val[i],
			Op:  Shun,
		})
		m.num[i] -= 1
		m.num[i+1] -= 1
		m.num[i+2] -= 1
		j := m.next(i)
		if m.deal_kezi(j) || m.deal_shunzi(j) {
			m.Group = m.Group[:len(m.Group)-1]
			m.num[i] += 1
			m.num[i+1] += 1
			m.num[i+2] += 1
			return true
		} else {
			m.Group = m.Group[:len(m.Group)-1]
			m.num[i] += 1
			m.num[i+1] += 1
			m.num[i+2] += 1
			return false
		}
	}
	return false
}

// hu233
func (m *stat) hu233() bool {
	for i := range m.val {
		if m.num[i] >= 2 {
			m.num[i] -= 2
			m.Group = append(m.Group, Group{
				Val: m.val[i],
				Op:  Jiang,
			})
			if m.deal_kezi(0) || m.deal_shunzi(0) {
				m.num[i] += 2
				m.Group = m.Group[:len(m.Group)-1]
				return true
			}
			m.Group = m.Group[:len(m.Group)-1]
			m.num[i] += 2
		}
	}
	return false
}

// 七对
func (m *stat) huQd() bool {
	if len(m.num) != 7 {
		return false
	}
	for i := range m.num {
		if m.num[i] != 2 && m.num[i] != 4 {
			return false
		}
	}
	return true
}

// 将一色
func (m *stat) huJys() bool {
	for i := range m.val {
		val := m.val[i]
		if val != Tiao2 && val != Tiao5 && val != Tiao8 &&
			val != Wan2 && val != Wan5 && val != Wan8 &&
			val != Tong2 && val != Tong5 && val != Tong8 {
			return false
		}
	}
	return true
}

// 豪华
func (m *stat) huHh() int {
	cnt := 0
	for i := range m.Group {
		if m.Group[i].Op == AGang {
			cnt++
		}
	}
	for i := range m.num {
		if m.num[i] == 4 {
			cnt++
		}
	}
	return cnt
}

// 清一色
func (m *stat) huQys() bool {
	c := m.val[0] / 10
	for i := range m.val {
		if m.val[i]/10 != c {
			return false
		}
	}
	for i := range m.Group {
		if m.Group[i].Val/10 != c {
			return false
		}
	}
	return true
}

// 碰碰胡
func (m *stat) huPph() bool {
	for i := range m.Group {
		if m.Group[i].Op == LChi || m.Group[i].Op == MChi || m.Group[i].Op == RChi {
			return false
		}
	}
	for i := range m.num {
		if m.num[i]%3 == 1 {
			return false
		}
	}
	return true
}

// 全球人
func (m *stat) huQqr() bool {
	if m.pai == 0 {
		return false
	}
	if len(m.val) != 1 {
		return false
	}
	if m.num[0] != 2 {
		return false
	}
	return true
}

func (m *stat) CanHu() bool {
	return m.hu233() || m.huQd() || m.huJys()
}

func (m *stat) CanOpSelf() []*Group {
	ret := make([]*Group, 0)
	for i := range m.num {
		if m.num[i] == 4 {
			ret = append(ret, &Group{
				Op:  AGang,
				Val: m.val[i],
			})
		}
	}
	if m.CanHu() {
		ret = append(ret, &Group{
			Op: HuPai,
		})
	}
	return ret
}

func (m *stat) CanOpOther(val uint8, op uint8) []*Group {
	ret := make([]*Group, 0)
	if op == DaPai {
		for i := range m.num {
			if m.val[i] == val {
				if i+2 < len(m.val) && m.val[i+1] == val+1 && m.val[i+2] == val+2 {
					ret = append(ret, &Group{
						Op:  LChi,
						Val: val,
					})
				}
				if i+1 < len(m.val) && i-1 >= 0 && m.val[i-1] == val-1 && m.val[i+1] == val+1 {
					ret = append(ret, &Group{
						Op:  MChi,
						Val: val,
					})
				}
				if i-2 >= 0 && m.val[i-2] == val-2 && m.val[i-1] == val-1 {
					ret = append(ret, &Group{
						Op:  RChi,
						Val: val,
					})
				}
				if m.num[i] >= 3 {
					ret = append(ret, &Group{
						Op:  Peng,
						Val: val,
					})
				}
				if m.num[i] >= 4 {
					ret = append(ret, &Group{
						Op:  MGang,
						Val: val,
					})
				}
				break
			}
		}
	}
	if m.CanHu() {
		ret = append(ret, &Group{
			Op:  HuPai,
			Val: val,
		})
	}
	return ret
}

func (m *stat) HuPai() int32 {
	hus := int32(0)
	if m.hu233() {
		hus |= PH
	} else if m.huQd() {
		hus |= QD
	}
	if m.huJys() {
		hus |= JYS
	}
	if hus == 0 {
		return hus
	}
	if m.huPph() {
		hus |= PPH
	}
	if m.huQys() {
		hus |= QYS
	}
	if m.huQqr() {
		hus |= QQR
	}
	n := m.huHh()
	if n == 1 {
		hus |= HH
	} else if n == 2 {
		hus |= HH2
	} else if n == 3 {
		hus |= HH3
	}
	return hus
}
