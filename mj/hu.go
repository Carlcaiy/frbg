package mj

import "sort"

const (
	PH  = 0x0001 // 屁胡
	MQQ = 0x0002 // 门前清
	PPH = 0x0004 // 碰碰胡
	QD  = 0x0008 // 七对
	QYS = 0x0010 // 清一色
	JYS = 0x0020 // 将一色
	FYS = 0x0040 // 风一色
	HH  = 0x0080 // 豪华
	HH2 = 0x0100 // 双豪华
	HH3 = 0x0200 // 三豪华
	QQR = 0x0400 // 全求人
	ZM  = 0x0800 // 自摸
)

var multi = map[int32]int32{
	HH3 | MQQ | PH: 10,
	HH2 | MQQ | PH: 10,
	HH | MQQ | PH:  10,
	QD:             8,
	HH | QD:        16,
}

type stat struct {
	pai   uint8
	val   []uint8
	num   []uint8
	Group []Group
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
		if val != 12 && val != 15 && val != 18 &&
			val != 22 && val != 25 && val != 28 &&
			val != 32 && val != 35 && val != 38 {
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
			Op: Hu,
		})
	}
	return ret
}

func (m *stat) CanOpOther(val uint8) []*Group {
	ret := make([]*Group, 0)
	for i := range m.num {
		if m.val[i] == val {
			if i+2 < len(m.val) && m.val[i+1] == val+1 && m.val[i+2] == val+2 {
				ret = append(ret, &Group{
					Op:  LChi,
					Val: val,
				})
			} else if i+1 < len(m.val) && i-1 >= 0 && m.val[i-1] == val-1 && m.val[i+1] == val+1 {
				ret = append(ret, &Group{
					Op:  MChi,
					Val: val,
				})
			} else if i-2 >= 0 && m.val[i-2] == val-2 && m.val[i-1] == val-1 {
				ret = append(ret, &Group{
					Op:  RChi,
					Val: val,
				})
			} else if m.num[i] >= 3 {
				ret = append(ret, &Group{
					Op:  Peng,
					Val: val,
				})
			} else if m.num[i] >= 4 {
				ret = append(ret, &Group{
					Op:  MGang,
					Val: val,
				})
			}
			break
		}
	}
	if m.CanHu() {
		ret = append(ret, &Group{
			Op:  Hu,
			Val: val,
		})
	}
	return ret
}

func (m *stat) HuPai() (int32, int32) {
	hus := int32(0)
	mul := int32(0)
	if m.hu233() {
		hus |= PH
	} else if m.huQd() {
		hus |= QD
	}
	if m.huJys() {
		hus |= JYS
	}
	if hus == 0 {
		return hus, mul
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
	return hus, mul
}
