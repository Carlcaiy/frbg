package route

import "time"

// 清除长期空闲的桌子
func (l *Local) clearRoom() {
	now := time.Now()
	for _, room := range l.rooms {
		if now.Sub(room.endTime) >= time.Hour {
			delete(l.rooms, room.roomId)
		}
	}
}
