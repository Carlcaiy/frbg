package mj

import "sort"

const (
	PH  = 1 // 屁胡
	MQQ = 2 // 门前清
	PPH = 3 // 碰碰胡
	QD  = 5 // 七对
	QYS = 4 // 清一色
	JYS = 6 // 将一色
	FYS = 7 // 风一色
	HH  = 8 // 豪华
)

type stat struct {
	val   []uint8
	num   []uint8
	Group []Group
}

func New(mj []uint8) *stat {
	sort.Slice(mj, func(i, j int) bool { return mj[i] < mj[j] })
	r := new(stat)
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
				m.Group = m.Group[:0]
				return true
			}
			m.Group = m.Group[:0]
			m.num[i] += 2
		}
	}
	return false
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
