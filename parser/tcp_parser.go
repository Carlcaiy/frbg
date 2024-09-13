package parser

import (
	"encoding/binary"
	"fmt"
	"io"

	"google.golang.org/protobuf/proto"
)

const (
	HeaderLen = 16
)

// 封包
// len + dest + gate + ver + uid + hold + cmd + data
// 2  +  1   +   1   +  2  +  4  +   4 +   2 +  data

var byteOrder binary.ByteOrder = binary.BigEndian

type Message struct {
	DestST uint8  // 目标服务器类型
	GateID uint8  // 网关信息
	UserID uint32 // 用户UID
	Ver    uint16 // 版本
	Hold   uint32 // 保留字段
	Cmd    uint16 // cmd
	Body   []byte // cmd + []byte
	All    []byte
}

func (m *Message) Bytes() []byte {
	return m.All
}

func Parse(r io.Reader) (p *Message, err error) {
	lenBs := make([]byte, 2)
	// 长度
	_, err = io.ReadFull(r, lenBs[:2])
	if err != nil {
		return nil, err
	}

	// 检测长度
	len := byteOrder.Uint16(lenBs[:2])
	if len < HeaderLen {
		return nil, fmt.Errorf("parse error len:%d", len)
	}

	all := make([]byte, len+2)
	_, err = io.ReadFull(r, all[2:])
	if err != nil {
		return nil, err
	}
	copy(all, lenBs)

	p = &Message{}
	p.DestST = all[2]
	p.GateID = all[3]
	p.UserID = byteOrder.Uint32(all[4:])
	p.Ver = byteOrder.Uint16(all[8:])
	p.Hold = byteOrder.Uint32(all[10:])
	p.Cmd = byteOrder.Uint16(all[14:])
	p.Body = all[16:]
	p.All = all
	return
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

func (m *Message) Pack() []byte {
	bs := make([]byte, HeaderLen+len(m.Body))
	byteOrder.PutUint16(bs, uint16(len(m.Body)))
	bs[2] = m.DestST
	bs[3] = m.GateID
	byteOrder.PutUint32(bs, m.UserID)
	byteOrder.PutUint16(bs, m.Ver)
	byteOrder.PutUint32(bs, m.Hold)
	byteOrder.PutUint16(bs, m.Cmd)
	copy(bs[HeaderLen:], m.Body)
	return bs
}

func (m *Message) PackCmd(cmd uint16, pro proto.Message) ([]byte, error) {
	body, err := proto.Marshal(pro)
	if err != nil {
		return nil, err
	}
	m.Cmd = cmd
	m.Body = body
	return m.Pack(), nil
}

func (m *Message) PackProto(pro proto.Message) ([]byte, error) {
	body, err := proto.Marshal(pro)
	if err != nil {
		return nil, err
	}
	m.Body = body
	return m.Pack(), nil
}

func Pack(uid uint32, dest uint8, cmd uint16, pro proto.Message) ([]byte, error) {
	body, err := proto.Marshal(pro)
	if err != nil {
		return nil, err
	}
	msg := &Message{
		UserID: uid,
		DestST: dest,
		Cmd:    cmd,
		Body:   body,
	}
	return msg.Pack(), nil
}

func (m *Message) UnPack(pro proto.Message) error {
	return proto.Unmarshal(m.Body, pro)
}
