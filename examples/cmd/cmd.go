package cmd

// 网关消息
const (
	GateBegin    = 1000 // Gate
	Login        = 1001 // 登录网关
	GateKick     = 1003 // 网关踢人
	GateMulti    = 1004 // 多播
	HeartBeat    = 1005 // 心跳
	Regist       = 1006 // 注册
	ReqGateLeave = 1007 // 离开网关
	ResGateLeave = 1008 // 离开网关
	Offline      = 1009 // 断线
	Test         = 1010 // 测试
	ToClient     = 1011 // 发送给客户端
	GateEnd      = 1999 // Gate
)

// 大厅消息
const (
	HallBegin   = 2000 // Hall
	GetGameList = 2001 // 获取游戏列表
	GetRoomList = 2002 // 获取房间列表
	EnterRoom   = 2003 // 进房请求
	LeaveRoom   = 2006 // 请求离开房间
	SlotsEnter  = 2008 // 进入老虎机
	SlotsSpin   = 2009 // 老虎机摇奖
	SlotsLeave  = 2010 // 离开老虎机
	HallEnd     = 2999 // Hall
)

// 游戏消息
const (
	GameBegin = 3000 // Game
	GameStart = 3001 // 进入游戏
	SyncData  = 3002 // 同步数据
	Tap       = 3003 // 点击游戏
	Round     = 3004 // 回合
	GameOver  = 3005 // 游戏结束
	Reconnect = 3006 // 重连
	CountDown = 3007 // 倒计时
	GameEnd   = 3999 // Game
)
