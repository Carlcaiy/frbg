package def

type ServerType uint8

const (
	ST_User      = 1 // 客户端
	ST_Gate      = 3 // 网关服
	ST_Login     = 4 // 登录服
	ST_Hall      = 5 // 大厅服
	ST_Broadcast = 6 // 广播服
	ST_Money     = 7 // 金币服
	ST_Game      = 8 // 游戏服
)

func (s ServerType) String() string {
	switch s {
	case ST_User:
		return "客户端"
	case ST_Gate:
		return "网关服"
	case ST_Login:
		return "登录服"
	case ST_Hall:
		return "大厅服"
	case ST_Broadcast:
		return "广播服"
	case ST_Money:
		return "金币服"
	case ST_Game:
		return "游戏服"
	default:
		return "未知的服务"
	}
}

type EventType int

const (
	ET_Close EventType = iota
	ET_Error
	ET_Timer
)

type Compoment interface {
	Init()
	Run()
	Close()
}
