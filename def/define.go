package def

type ServerType int32

const (
	ST_User      = 1 // 客户端
	ST_Gate      = 2 // 网关服
	ST_Login     = 3 // 登录服
	ST_Hall      = 4 // 大厅服
	ST_Broadcast = 5 // 广播服
	ST_Money     = 6 // 金币服
	ST_Game      = 7 // 游戏服
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
