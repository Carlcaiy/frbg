package route

import (
	"frbg/local"
	"frbg/network"
)

type User struct {
	userID        uint32
	gateID        uint32
	gameID        uint32
	hallID        uint32
	roomID        uint32
	*network.Conn // 玩家可能从不同的网关过来，所以需要存一下网关ID
}

func (u *User) UserID() uint32 {
	return u.userID
}

func (u *User) GameID() uint32 {
	return u.gameID
}

func (u *User) GateID() uint32 {
	return u.gateID
}

type RoomTemplete struct {
	TempId    uint32
	UserCount uint32
	GameID    uint32
}

type RoomInstance struct {
	*RoomTemplete
	sitCount uint32
	status   int32
	users    []*User
	conn     *network.Conn
	roomID   uint32
	tevent   *local.TimerEvent
}
