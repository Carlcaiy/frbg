package route

import (
	"frbg/network"
	"frbg/timer"
)

type User struct {
	userID        uint32
	gateID        uint8
	hallID        uint8
	deskID        uint32
	*network.Conn // 玩家可能从不同的网关过来，所以需要存一下网关ID
}

func (u *User) UserID() uint32 {
	return u.userID
}

func (u *User) GameID() uint8 {
	return 0
}

func (u *User) GateID() uint8 {
	return u.gateID
}

type DeskTemplete struct {
	RoomId    uint32
	GameID    uint8
	UserCount uint32
}

type DeskInstance struct {
	*DeskTemplete
	sitCount        uint32 // 坐下数量
	status          int32  // 房间状态 0等待中 1游戏中
	users           []*User
	conn            *network.Conn
	deskID          uint32
	delayStartEvent *timer.Task
}
