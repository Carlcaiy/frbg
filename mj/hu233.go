package mj

import "sort"

type stat struct {
	val   []byte
	num   []byte
	Group []Group
}

func New(mj []byte) *stat {
	sort.Slice(mj, func(i, j int) bool { return mj[i] < mj[j] })
	r := new(stat)
	tail := byte(0)
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

func (m *stat) Pihu() bool {
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

func (m *stat) QiDui() bool {
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

func (m *stat) CanHu() bool {
	return m.Pihu() || m.QiDui()
}
