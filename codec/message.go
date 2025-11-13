package codec

import (
	"fmt"

	"google.golang.org/protobuf/proto"
)

const HeartBeat uint32 = 0x1 // 心跳包

type Message struct {
	DestST uint8  // 目标服务TYPE
	GameID uint8  // 游戏ID
	GateID uint8  // 网关ID
	UserID uint32 // 用户UID
	Ver    uint16 // 版本
	Type   uint32 // 保留字段
	Cmd    uint16 // cmd
	Body   []byte // cmd + []byte
	All    []byte
}

func NewMessage(uid uint32, dest uint8, cmd uint16, gateId uint8, pro proto.Message) *Message {
	body, _ := proto.Marshal(pro)
	return &Message{
		UserID: uid,
		DestST: dest,
		GateID: gateId,
		Body:   body,
		Cmd:    cmd,
	}
}

func (m *Message) String() string {
	return fmt.Sprintf("DestST:%d GateID:%d UserID:%d Ver:%d Hold:%d Cmd:%d Len:%d", m.DestST, m.GateID, m.UserID, m.Ver, m.Type, m.Cmd, len(m.All))
}

func (m *Message) Pack() []byte {
	bs := make([]byte, HeaderLen+len(m.Body))
	byteOrder.PutUint16(bs, HeaderLen+uint16(len(m.Body)))
	bs[2] = m.DestST
	bs[3] = m.GameID
	bs[4] = m.GateID
	byteOrder.PutUint32(bs[5:], m.UserID)
	byteOrder.PutUint16(bs[9:], m.Ver)
	byteOrder.PutUint32(bs[11:], m.Type)
	byteOrder.PutUint16(bs[15:], m.Cmd)
	copy(bs[HeaderLen:], m.Body)
	return bs
}

func (m *Message) PackWith(cmd uint16, pro proto.Message) ([]byte, error) {
	body, err := proto.Marshal(pro)
	if err != nil {
		return nil, err
	}
	m.Cmd = cmd
	m.Body = body
	return m.Pack(), nil
}

func (m *Message) Unpack(pro proto.Message) error {
	return proto.Unmarshal(m.Body, pro)
}

func (m *Message) Bytes() []byte {
	return m.All
}
