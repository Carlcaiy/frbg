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

func (m *Message) Bytes() []byte {
	return m.All
}

func TcpRead(r io.Reader) (p *Message, err error) {
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

	all := make([]byte, len)
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
