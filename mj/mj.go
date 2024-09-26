package mj

import "fmt"

// 麻将枚举
const (
	Dong, Nan, Xi, Bei, Zhong, Fa, Bai                            = 1, 2, 3, 4, 5, 6, 7
	Tiao1, Tiao2, Tiao3, Tiao4, Tiao5, Tiao6, Tiao7, Tiao8, Tiao9 = 11, 12, 13, 14, 15, 16, 17, 18, 19
	Wan1, Wan2, Wan3, Wan4, Wan5, Wan6, Wan7, Wan8, Wan9          = 21, 22, 23, 24, 25, 26, 27, 28, 29
	Tong1, Tong2, Tong3, Tong4, Tong5, Tong6, Tong7, Tong8, Tong9 = 31, 32, 33, 34, 35, 36, 37, 38, 39
)

// 名称枚举
var feng = [7]string{"东风", "南风", "西风", "北风", "红中", "发財", "白板"}
var tiao = [9]string{"一条", "二条", "三条", "四条", "五条", "六条", "七条", "八条", "九条"}
var wan = [9]string{"一万", "二万", "三万", "四万", "五万", "六万", "七万", "八万", "九万"}
var tong = [9]string{"一筒", "二筒", "三筒", "四筒", "五筒", "六筒", "七筒", "八筒", "九筒"}

// 操作枚举
const (
	LChi, MChi, RChi    = 1, 2, 3
	Peng                = 4
	MGang, BGang, AGang = 5, 6, 7
	Jiang, Shun, Ke     = 8, 9, 10
)

type Group struct {
	Op  uint8
	Val uint8
}

func (p *Group) String() string {
	str := ""
	switch p.Op {
	case LChi:
		str = fmt.Sprintf("吃:[%s]%s%s", Pai(p.Val), Pai(p.Val+1), Pai(p.Val+2))
	case MChi:
		str = fmt.Sprintf("吃:%s[%s]%s", Pai(p.Val-1), Pai(p.Val), Pai(p.Val+1))
	case RChi:
		str = fmt.Sprintf("吃:%s%s[%s]", Pai(p.Val-2), Pai(p.Val-1), Pai(p.Val))
	case Peng:
		str = fmt.Sprintf("碰:%s%s[%s]", Pai(p.Val), Pai(p.Val), Pai(p.Val))
	case MGang:
		str = fmt.Sprintf("明杠:%s", Pai(p.Val))
	case BGang:
		str = fmt.Sprintf("补杠:%s", Pai(p.Val))
	case AGang:
		str = fmt.Sprintf("暗杠:%s", Pai(p.Val))
	case Jiang:
		str = fmt.Sprintf("将:%s%s", Pai(p.Val), Pai(p.Val))
	case Shun:
		str = fmt.Sprintf("顺:%s%s%s", Pai(p.Val), Pai(p.Val+1), Pai(p.Val+2))
	case Ke:
		str = fmt.Sprintf("刻:%s%s%s", Pai(p.Val), Pai(p.Val), Pai(p.Val))
	}
	return str
}

func Pai(v byte) string {
	if v < 8 {
		return feng[v-1]
	} else if v < 20 {
		return tiao[v-11]
	} else if v < 30 {
		return wan[v-21]
	} else if v < 40 {
		return tong[v-31]
	}
	return "花牌"
}
