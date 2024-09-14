package local

import "frbg/network"

type UserImplement struct {
	userId uint32
	gameId uint8
	gateId uint8
	*network.Conn
}

func (u *UserImplement) UserID() uint32 {
	return u.userId
}

func (u *UserImplement) GameID() uint8 {
	return u.gameId
}

func (u *UserImplement) GateID() uint8 {
	return u.gateId
}
