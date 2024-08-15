package slots

import (
	"frbg/examples/proto"
	"math/rand"
)

type BetConf struct {
	Bet   []int32 // 下注选项
	Level []int32 // 下注等级
	Lines []int32 // 线数选项
}

type ReelConf struct {
	Weight []int32
	Length int32
}

func (p *ReelConf) Sum() int32 {
	sum := int32(0)
	for i := range p.Weight {
		sum += p.Weight[i]
	}
	return sum
}

func (p *ReelConf) Get(sum int32) int32 {
	random := rand.Int31n(sum)
	for i, weight := range p.Weight {
		if random < weight {
			return int32(i)
		} else {
			random -= weight
		}
	}
	return -1
}

type FreeConf struct {
	Count3 int32 `yaml:"count3"`
	Count4 int32 `yaml:"count4"`
	Count5 int32 `yaml:"count5"`
}

func (f *FreeConf) GetSpin(count int32) int32 {
	if count == 3 {
		return f.Count3
	}
	if count == 4 {
		return f.Count4
	}
	if count == 5 {
		return f.Count5
	}
	return 0
}

type BonusConf struct {
	Elem   []int32
	Multi  []int32
	Weight []int32
}

func (b *BonusConf) Sum() int32 {
	sum := int32(0)
	for i := range b.Weight {
		sum += b.Weight[i]
	}
	return sum
}

func (b *BonusConf) Get(sum int32) (int32, int32) {
	random := rand.Int31n(sum)
	for i, weight := range b.Weight {
		if random < weight {
			return b.Elem[i], b.Multi[i]
		} else {
			random -= weight
		}
	}
	return 0, 0
}

type SlotsConf struct {
	GameID    int32 // 游戏ID
	ElemConf  []*proto.SlotsElem
	RouteConf []*proto.SlotsLine // 中奖线路
	ReelsConf []*ReelConf        // 摇奖元素概率
	BetConf   *BetConf           // 摇奖选项
	FreeConf  *FreeConf          // 免费游戏配置
	BonusConf *BonusConf         // bonus配置
}

type SlotsData struct {
	*SlotsConf
	Uid      uint32
	Board    []int32 // 展示ID
	FreeSpin int32   // 免费摇奖
	Bonus    bool    // Bonus
	Free     bool    //
}
