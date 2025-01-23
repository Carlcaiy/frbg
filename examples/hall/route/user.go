package route

import (
	"frbg/network"
	"time"
)

type User struct {
	userID        uint32
	gateID        uint8
	MatchTime     time.Time
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
