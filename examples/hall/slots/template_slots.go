package slots

import (
	"frbg/examples/hall/db"
	"frbg/examples/proto"
	"log"
	"sync"
)

var data sync.Map

func GetFuConf() *SlotsConf {
	return &SlotsConf{
		GameID: SlotsFu,
		ElemConf: []*proto.SlotsElem{
			{ElemId: 1, ElemName: "A", Multi3: 1},
			{ElemId: 2, ElemName: "B", Multi3: 2},
			{ElemId: 3, ElemName: "C", Multi3: 3},
			{ElemId: 4, ElemName: "恭", Multi3: 6},
			{ElemId: 5, ElemName: "喜", Multi3: 8},
			{ElemId: 6, ElemName: "發", Multi3: 9},
			{ElemId: 7, ElemName: "財", Multi3: 10},
			{ElemId: ElemFree, ElemName: "Free"},
			{ElemId: ElemBonus, ElemName: "Bonus"},
			{ElemId: ElemWild, ElemName: "Wild"},
		},
		ReelsConf: []*ReelConf{
			{Weight: []int32{30, 20, 20, 10, 10, 10, 10, 3, 3, 3}, Length: 3},
			{Weight: []int32{30, 20, 20, 10, 10, 10, 10, 3, 3, 3}, Length: 4},
			{Weight: []int32{30, 20, 20, 10, 10, 10, 10, 3, 3, 3}, Length: 3},
		},
		RouteConf: []*proto.SlotsLine{
			{LineId: 1, LinePos: []int32{1, 4, 7}},
			{LineId: 2, LinePos: []int32{1, 5, 9}},
			{LineId: 3, LinePos: []int32{2, 5, 8}},
			{LineId: 4, LinePos: []int32{3, 5, 7}},
			{LineId: 5, LinePos: []int32{3, 6, 9}},
		},
		BetConf: &BetConf{
			Bet:   []int32{1000},
			Level: []int32{1, 2, 4, 8, 16, 32, 64},
			Lines: []int32{5},
		},
		BonusConf: &BonusConf{
			Elem:   []int32{1, 2, 3},
			Multi:  []int32{8, 16, 32},
			Weight: []int32{10, 4, 1},
		},
	}
}

// 获取数据
func GetSlotsData(uid uint32, gameId int32) *SlotsData {
	if value, ok := data.Load(uid); ok {
		return value.(*SlotsData)
	}
	var conf *SlotsConf
	if gameId == SlotsFu {
		conf = GetFuConf()
	}
	if conf == nil {
		return nil
	}
	slots := &SlotsData{
		SlotsConf: conf,
		Board:     []int32{1, 2, 3, 4, 5, 6, 7, ElemFree, ElemBonus, ElemWild},
		FreeSpin:  0,
		Bonus:     false,
		Free:      false,
	}
	data.Store(gameId, slots)
	return slots
}

// 删除数据
func DelSlotsData(uid uint32) {
	data.Delete(uid)
}

// 根据权重洗牌
func (s *SlotsData) shuffle() {
	idx := int32(0)
	for _, reel := range s.ReelsConf {
		sum := reel.Sum()
		for i := int32(0); i < reel.Length; i++ {
			if index := reel.Get(sum); index >= 0 {
				s.Board[idx] = s.ElemConf[index].ElemId
				idx++
			}
		}
	}
	idx = 0
	for _, reel := range s.ReelsConf {
		log.Printf("%v", s.Board[idx:idx+reel.Length])
		idx = idx + reel.Length
	}
}

// 摇奖
func (s *SlotsData) Spin(bet int64) (*proto.ResSlotsSpin, error) {
	free := false
	win := int64(0)
	lines := make([]int32, 0)
	board := make([]int32, len(s.Board))

	s.shuffle()
	copy(board, s.Board)
	for _, item := range s.RouteConf {
		lineId, posArr := item.LineId, item.LinePos
		length := len(posArr)
		ok := true
		dest := s.Board[posArr[0]]
		if dest != ElemBonus && dest != ElemFree {
			for i := 1; i < length; i++ {
				elem := s.Board[posArr[i]]
				// 如果为bonus或者freen
				if elem == ElemBonus || elem == ElemFree {
					ok = false
					break
				}
				// 位置元素为wild则跳过
				if elem == ElemWild {
					continue
				}
				// 目标元素为wild则更换
				if dest == ElemWild {
					dest = elem
					continue
				}
				// 元素不相同
				if elem != dest {
					ok = false
					break
				}
			}
			if !ok {
				continue
			}

			lines = append(lines, lineId)

			multi := int32(0)
			for _, elem := range s.ElemConf {
				if elem.ElemId == dest {
					multi = elem.Multi3
				}
			}
			if multi > 0 {
				win += int64(multi) * bet
			} else {
				log.Printf("elem:%d not found multi", dest)
			}
		}
	}
	rsp := &proto.ResSlotsSpin{
		Money:    0,
		Win:      win,
		Board:    board,
		Lines:    lines,
		Free:     free,
		LeftSpin: s.FreeSpin,
	}
	if money, err := db.UpdateMoney(s.Uid, win, "spin"); err != nil {
		return nil, err
	} else {
		rsp.Money = money
	}

	// free
	if count, pos := s.Elem(ElemFree); count >= 3 {
		rsp.Free = true
		rsp.FreeData = &proto.SlotsFree{
			Pos:      pos,
			FreeSpin: s.FreeConf.GetSpin(count),
		}
	}

	// bonus
	if count, pos := s.Elem(ElemBonus); count >= 3 {
		sum := s.BonusConf.Sum()
		b1, m1 := s.BonusConf.Get(sum)
		b2, _ := s.BonusConf.Get(sum)
		b3, _ := s.BonusConf.Get(sum)
		win := bet * int64(m1)

		rsp.Bonus = true
		rsp.BonusData = &proto.SlotsBonus{
			Pos:   pos,
			Board: []int32{b1, b2, b3},
			Win:   win,
		}

		if money, err := db.UpdateMoney(s.Uid, win, "bonus"); err != nil {
			return nil, err
		} else {
			rsp.BonusData.Money = money
		}
	}

	return rsp, nil
}

func (s *SlotsData) Elem(elem int32) (int32, []int32) {
	num := int32(0)
	pos := make([]int32, 3)
	for i, e := range s.Board {
		if e == elem {
			num++
			pos = append(pos, int32(i))
		}
	}
	if num >= 3 {
		return num, pos
	}
	return num, nil
}
