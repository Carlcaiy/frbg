package route

import (
	"frbg/network"
	"sync"
)

type User struct {
	Uid    uint32        `json:"uid"`     // 玩家uid
	GameId uint8         `json:"game_id"` // GameUid
	HallId uint8         `json:"hall_id"` // HallId
	Nick   string        `json:"nick"`    // 昵称
	Sex    uint8         `json:"sex"`     // 0男 1女
	IconId uint8         `json:"icon_id"` // 默认头像id
	conn   *network.Conn // 连接
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

type Clients struct {
	mu      sync.RWMutex
	fd_uid  map[int]uint32           // fd -> uid
	m_users map[uint32]*network.Conn // uid -> conn
}

func NewClients() *Clients {
	return &Clients{
		m_users: make(map[uint32]*network.Conn),
		fd_uid:  make(map[int]uint32),
	}
}

func (u *Clients) GetClient(uid uint32) *network.Conn {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.m_users[uid]
}

func (u *Clients) SetClient(uid uint32, conn *network.Conn) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.m_users[uid] = conn
	u.fd_uid[conn.Fd] = uid
}

func (u *Clients) DelClient(conn *network.Conn) {
	u.mu.Lock()
	defer u.mu.Unlock()
	delete(u.m_users, u.fd_uid[conn.Fd])
	delete(u.fd_uid, conn.Fd)
}
