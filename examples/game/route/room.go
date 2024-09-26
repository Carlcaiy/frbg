package route

type xxxx struct {
	rooms map[uint32]*Room // 房间配置
}

func (p *xxxx) getRoom(rid uint32) *Room {
	if rid > 0 {
		return p.rooms[rid]
	} else {
		return &Room{}
	}
}
