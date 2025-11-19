package codec

import (
	"fmt"

	"google.golang.org/protobuf/proto"
)

const HeartBeat uint32 = 0x1 // 心跳包

type Header struct {
	Len uint16 // 消息长度
}

type Message struct {
	DestType uint8  // 目标服务类型
	DestId   uint8  // 目标服务ID
	Ver      uint16 // 版本
	Type     uint32 // 保留字段
	Cmd      uint16 // cmd
	Body     []byte // cmd + []byte
	All      []byte
}

func NewMessage(serverType uint8, serveID uint8, cmd uint16, pro proto.Message) *Message {
	body, _ := proto.Marshal(pro)
	return &Message{
		DestType: serverType,
		DestId:   serveID,
		Body:     body,
		Cmd:      cmd,
	}
}

func (m *Message) String() string {
	return fmt.Sprintf("Serve:%d ServeID:%d Ver:%d Hold:%d Cmd:%d Len:%d", m.DestType, m.DestId, m.Ver, m.Type, m.Cmd, len(m.All))
}

func (m *Message) Pack() []byte {
	bs := make([]byte, HeaderLen+len(m.Body))
	byteOrder.PutUint16(bs, HeaderLen+uint16(len(m.Body)))
	bs[2] = m.DestType
	bs[3] = m.DestId
	byteOrder.PutUint16(bs[4:], m.Ver)
	byteOrder.PutUint32(bs[6:], m.Type)
	byteOrder.PutUint16(bs[10:], m.Cmd)
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
