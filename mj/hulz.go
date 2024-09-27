package mj

type statlz struct {
	pai   uint8   // 点炮的牌 >0表示点炮 =0表示自摸
	num   []int8  // 手牌数量集合
	Group []Group // 手牌将刻顺
}

func Newlz(mj []uint8, p uint8, lz uint8, g []Group) *statlz {
	st := &statlz{
		pai:   p,
		num:   make([]int8, Chun),
		Group: g,
	}
	for _, v := range mj {
		st.num[v]++
	}
	if p > 0 {
		st.num[p]++
	}
	if lz > 0 {
		st.num[Laizi] = st.num[lz]
		st.num[lz] = 0
	}
	return st
}

func (m *statlz) next(pai uint8) uint8 {
	for pai == Laizi || (m.num[pai] <= 0 && pai <= Tong9) {
		pai++
	}
	return pai
}

func (m *statlz) deal_kezi(pai uint8) bool {
	if pai > Tong9 {
		return true
	}
	if m.num[pai]+m.num[Laizi] >= 3 {
		m.num[pai] -= 3
		if m.num[pai] < 0 {
			m.num[Laizi] += m.num[pai]
		}
		m.Group = append(m.Group, Group{
			Val: pai,
			Op:  Ke,
		})
		j := m.next(pai)
		if m.deal_kezi(j) || m.deal_shunzi(j) {
			// m.Group = m.Group[:len(m.Group)-1]
			if m.num[pai] < 0 {
				m.num[Laizi] -= m.num[pai]
			}
			m.num[pai] += 3
			return true
		}
		m.Group = m.Group[:len(m.Group)-1]
		if m.num[pai] < 0 {
			m.num[Laizi] -= m.num[pai]
		}
		m.num[pai] += 3
	}
	return false
}

func (m *statlz) deal_shunzi(pai uint8) bool {
	if pai < Tiao1 {
		return false
	}
	if pai > Tong9 {
		return true
	}

	// 牌数和牌值一致
	if m.num[pai] >= 1 && m.num[pai+1]+m.num[Laizi] >= 1 && m.num[pai+2] >= 1 ||
		m.num[pai] >= 1 && m.num[pai+1] >= 1 && m.num[pai+2]+m.num[Laizi] >= 1 {
		m.num[pai] -= 1
		m.num[pai+1] -= 1
		m.num[pai+2] -= 1
		if m.num[pai+1] < 0 {
			m.num[Laizi] -= 1
		}
		if m.num[pai+2] < 0 {
			m.num[Laizi] -= 1
		}
		m.Group = append(m.Group, Group{
			Val: pai,
			Op:  Shun,
		})
		j := m.next(pai)
		if m.deal_kezi(j) || m.deal_shunzi(j) {
			// m.Group = m.Group[:len(m.Group)-1]
			m.num[pai] += 1
			m.num[pai+1] += 1
			m.num[pai+2] += 1
			if m.num[pai+1] == 0 {
				m.num[Laizi] += 1
			}
			if m.num[pai+2] == 0 {
				m.num[Laizi] += 1
			}
			return true
		}
		m.Group = m.Group[:len(m.Group)-1]
		m.num[pai] += 1
		m.num[pai+1] += 1
		m.num[pai+2] += 1
		if m.num[pai+1] == 0 {
			m.num[Laizi] += 1
		}
		if m.num[pai+2] == 0 {
			m.num[Laizi] += 1
		}
	}
	return false
}

// hu233
func (m *statlz) hu233() bool {
	for i := range m.num {
		if m.num[i] > 0 && i != Laizi && m.num[i]+m.num[Laizi] >= 2 {
			m.num[i] -= 2
			if m.num[i] < 0 {
				m.num[Laizi] -= 1
			}
			m.Group = append(m.Group, Group{
				Val: uint8(i),
				Op:  Jiang,
			})
			idx := m.next(0)
			if m.deal_kezi(idx) || m.deal_shunzi(idx) {
				m.num[i] += 2
				if m.num[i] == 1 {
					m.num[Laizi] += 1
				}
				// m.Group = m.Group[:len(m.Group)-1]
				return true
			}
			m.num[i] += 2
			if m.num[i] == 1 {
				m.num[Laizi] += 1
			}
			m.Group = m.Group[:len(m.Group)-1]
		}
	}
	return false
}

// 七对
func (m *statlz) huQd() bool {
	n := int8(0)
	for i := range m.num {
		if m.num[i] != 2 && m.num[i] != 4 {
			return false
		}
		n += m.num[i]
	}
	return n == 14
}

// 将一色
func (m *statlz) huJys() bool {
	for val := range m.num {
		if m.num[val] > 0 && val != Laizi &&
			val != Tiao2 && val != Tiao5 && val != Tiao8 &&
			val != Wan2 && val != Wan5 && val != Wan8 &&
			val != Tong2 && val != Tong5 && val != Tong8 {
			return false
		}
	}
	return true
}

// 豪华
func (m *statlz) huHh() int {
	cnt := 0
	for i := range m.Group {
		if m.Group[i].Op == AGang {
			cnt++
		}
	}
	for i := range m.num {
		if m.num[i] == 4 && i != Laizi {
			cnt++
		}
	}
	return cnt
}

// 清一色
func (m *statlz) huQys() bool {
	c := m.Group[0].Val / 10
	for i := range m.Group {
		if m.Group[i].Val/10 != c {
			return false
		}
	}
	return true
}

// 碰碰胡
func (m *statlz) huPph() bool {
	for i := range m.Group {
		if m.Group[i].Op == LChi || m.Group[i].Op == MChi || m.Group[i].Op == RChi || m.Group[i].Op == Shun {
			return false
		}
	}
	return true
}

// 全球人
func (m *statlz) huQqr() bool {
	if m.pai == 0 {
		return false
	}
	for i := range m.Group {
		if m.Group[i].Op == Ke || m.Group[i].Op == Shun {
			return false
		}
	}
	return m.num[m.pai] == 2
}

func (m *statlz) CanHu() bool {
	return m.hu233() || m.huQd() || m.huJys()
}

func (m *statlz) CanOpSelf() []*Group {
	ret := make([]*Group, 0)
	for i := range m.num {
		if m.num[i] == 4 {
			ret = append(ret, &Group{
				Op:  AGang,
				Val: uint8(i),
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

func (m *statlz) CanOpOther(val uint8) []*Group {
	ret := make([]*Group, 0)
	if m.num[val+1] > 0 && m.num[val+2] > 0 {
		ret = append(ret, &Group{
			Op:  LChi,
			Val: val,
		})
	}
	if m.num[val-1] > 0 && m.num[val+1] > 0 {
		ret = append(ret, &Group{
			Op:  MChi,
			Val: val,
		})
	}
	if m.num[val-2] > 0 && m.num[val-1] > 0 {
		ret = append(ret, &Group{
			Op:  RChi,
			Val: val,
		})
	}
	if m.num[val] >= 3 {
		ret = append(ret, &Group{
			Op:  Peng,
			Val: val,
		})
	}
	if m.num[val] >= 4 {
		ret = append(ret, &Group{
			Op:  MGang,
			Val: val,
		})
	}
	if m.CanHu() {
		ret = append(ret, &Group{
			Op:  Hu,
			Val: val,
		})
	}
	return ret
}

func (m *statlz) HuPai() int32 {
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
