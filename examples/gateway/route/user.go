package route

import "frbg/network"

type User struct {
	uid           uint32 // 玩家uid
	gameId        uint8  // GameUid
	hallId        uint8  // HallId
	*network.Conn        // 连接
}

func (u *User) UserID() uint32 {
	return u.uid
}

func (u *User) GameID() uint8 {
	return u.gameId
}

func (u *User) GateID() uint8 {
	return 0
}
