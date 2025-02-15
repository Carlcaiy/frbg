package cmd

// 网关消息
const (
	GateBegin    = 1000 // Gate
	Login        = 1001 // 登录网关
	GateKick     = 1003 // 网关踢人
	MultiBC      = 1004 // 多播
	HeartBeat    = 1005 // 心跳
	Regist       = 1006 // 注册
	Logout       = 1007 // 离开网关
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
	// EnterRoom   = 2003 // 进房请求
	// LeaveRoom  = 2006 // 请求离开房间
	EnterSlots = 2008 // 进入老虎机
	SpinSlots  = 2009 // 老虎机摇奖
	LeaveSlots = 2010 // 离开老虎机
	HallEnd    = 2999 // Hall
)

// 游戏消息
const (
	GameBegin = 3000 // Game
	EnterRoom = 3001 // 进入游戏
	BcOpt     = 3002 // 广播操作
	OptGame   = 3003 // 点击游戏
	Round     = 3004 // 回合
	GameOver  = 3005 // 游戏结束
	Reconnect = 3006 // 重连
	CountDown = 3007 // 倒计时
	GameFaPai = 3008 // 发牌
	LeaveRoom = 3009 // 离开游戏
	GameEnd   = 3999 // Game
)
