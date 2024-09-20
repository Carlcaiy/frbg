package route

import "frbg/network"

type User struct {
	Uid           uint32 `json:"uid"`     // 玩家uid
	GameId        uint8  `json:"game_id"` // GameUid
	HallId        uint8  `json:"hall_id"` // HallId
	Nick          string `json:"nick"`    // 昵称
	Sex           uint8  `json:"sex"`     // 0男 1女
	IconId        uint8  `json:"icon_id"` // 默认头像id
	*network.Conn        // 连接
}

func (u *User) UserID() uint32 {
	return u.Uid
}

func (u *User) GameID() uint8 {
	return u.GameId
}

func (u *User) GateID() uint8 {
	return 0
}
