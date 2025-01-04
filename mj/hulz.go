package mj

type statlz struct {
	laizi  uint8   // 癞子牌
	pai    uint8   // 点炮的牌 >0表示点炮 =0表示自摸
	num    []int8  // 手牌数量集合
	groups []Group // 手牌将刻顺
}

// 格式化结构体
func (s *statlz) String() string {
	str := ""
	for p := range s.num {
		if s.num[p] == 0 {
			continue
		}
		str += "["
		for i := int8(0); i < s.num[p]; i++ {
			if i != 0 {
				str += " "
			}
			str += Pai(uint8(p))
		}
		str += "]"
	}

	for i := range s.groups {
		str += s.groups[i].String()
	}

	return str
}

func Newlz(mj []uint8, pai uint8, laizi uint8, group []Group) *statlz {
	st := &statlz{
		pai:    pai,
		num:    make([]int8, Chun),
		groups: group,
	}
	for _, v := range mj {
		st.num[v]++
	}
	if pai > 0 {
		st.num[pai]++
	}
	if laizi > 0 {
		st.laizi = laizi
		st.num[Laizi] = st.num[laizi]
		st.num[laizi] = 0
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
		m.groups = append(m.groups, Group{
			Val: pai,
			Op:  Ke,
		})
		j := m.next(pai)
		if m.deal_kezi(j) || m.deal_shunzi(j) {
			if m.num[pai] < 0 {
				m.num[Laizi] -= m.num[pai]
			}
			m.num[pai] += 3
			return true
		}
		m.groups = m.groups[:len(m.groups)-1]
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
		m.groups = append(m.groups, Group{
			Val: pai,
			Op:  Shun,
		})
		j := m.next(pai)
		if m.deal_kezi(j) || m.deal_shunzi(j) {
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
		m.groups = m.groups[:len(m.groups)-1]
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
			m.groups = append(m.groups, Group{
				Val: uint8(i),
				Op:  Jiang,
			})
			idx := m.next(0)
			if m.deal_kezi(idx) || m.deal_shunzi(idx) {
				m.num[i] += 2
				if m.num[i] == 1 {
					m.num[Laizi] += 1
				}
				return true
			}
			m.num[i] += 2
			if m.num[i] == 1 {
				m.num[Laizi] += 1
			}
			m.groups = m.groups[:len(m.groups)-1]
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
	for i := range m.groups {
		if m.groups[i].Op == AGang {
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
	c := m.groups[0].Val / 10
	for i := range m.groups {
		if m.groups[i].Val/10 != c {
			return false
		}
	}
	return true
}

// 碰碰胡
func (m *statlz) huPph() bool {
	for i := range m.groups {
		if m.groups[i].Op == LChi || m.groups[i].Op == MChi || m.groups[i].Op == RChi || m.groups[i].Op == Shun {
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
	for i := range m.groups {
		if m.groups[i].Op == Ke || m.groups[i].Op == Shun {
			return false
		}
	}
	return m.num[m.pai] == 2
}

func (m *statlz) huMqq() bool {
	if m.pai > 0 {
		return false
	}
	for i := range m.groups {
		if m.groups[i].Op == LChi || m.groups[i].Op == MChi || m.groups[i].Op == RChi || m.groups[i].Op == Peng || m.groups[i].Op == MGang || m.groups[i].Op == BGang {
			return false
		}
	}
	return true
}

func (m *statlz) huYh() bool {
	return m.num[Laizi] == 0
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
			Op: HuPai,
		})
	}
	return ret
}

func (m *statlz) CanOpOther(val, op uint8) []*Group {
	ret := make([]*Group, 0)
	if op == DaPai {
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
	}
	if m.CanHu() {
		ret = append(ret, &Group{
			Op:  HuPai,
			Val: val,
		})
	}
	return ret
}

func (m *statlz) HuPai() (int32, int32) {
	lzHx := m.huType()
	lzScore := m.huScore(lzHx)
	if m.laizi > 0 && m.num[m.laizi] > 0 {
		m.num[m.laizi] = m.num[Laizi]
		m.num[Laizi] = 0
		hx := m.huType()
		score := m.huScore(hx)
		if score >= lzScore {
			return hx, score
		}
	}
	return lzHx, lzScore
}

func (m *statlz) huType() int32 {
	hus := int32(0)
	// 基础胡型 平胡,七对,将一色
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
	// 碰碰胡
	if m.huPph() {
		hus |= PPH
	}
	// 清一色
	if m.huQys() {
		hus |= QYS
	}
	// 全求人
	if m.huQqr() {
		hus |= QQR
	}
	// 门前清
	if m.huMqq() && hus == PH {
		hus |= MQQ
	}
	// 豪华
	n := m.huHh()
	if n == 1 {
		hus |= HH
	} else if n == 2 {
		hus |= HH2
	} else if n == 3 {
		hus |= HH3
	}
	// 自摸
	if m.pai == 0 && hus&MQQ == 0 {
		hus |= ZM
	}
	// 硬胡
	if m.huYh() {
		hus |= YH
	}
	return hus
}

func (m *statlz) huScore(hx int32) int32 {
	base := int32(0)
	// 七对|清一色|将一色|全球人|碰碰胡 8分
	// 门前清 6分
	// 平胡 2分
	if hx&(QD|JYS|QYS|QQR|PPH) > 0 {
		base = 8
	} else if hx&MQQ > 0 {
		base = 6
	} else {
		base = 2
	}
	if hx&HH > 0 {
		base *= 2
	}
	if hx&HH2 > 0 {
		base *= 4
	}
	if hx&HH3 > 0 {
		base *= 8
	}
	if hx&YH > 0 {
		base *= 2
	}
	return base
}
