package codec

import (
	"encoding/binary"
	"fmt"
	"frbg/def"
	"io"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"google.golang.org/protobuf/proto"
)

const (
	HeaderLen = 17
)

// 封包
// len + dest + destid + gate + ver + uid + hold + cmd + data
// 2  +  1   +   1   +   1   +  2  +  4  +   4 +   2 +  data

var byteOrder binary.ByteOrder = binary.BigEndian

func Read(r io.ReadWriter, st uint8) (p *Message, err error) {
	if st == def.ST_WsGate {
		return WsRead(r)
	}
	return TcpRead(r)
}

func WsRead(r io.ReadWriter) (p *Message, err error) {
	all, opCode, err := wsutil.ReadData(r, ws.StateClientSide)
	if opCode == ws.OpClose {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	length := byteOrder.Uint16(all[:2])
	if int(length) != len(all) {
		return nil, fmt.Errorf("binary unpack len error")
	}
	if length < HeaderLen {
		return nil, fmt.Errorf("binary unpack error")
	}
	p = &Message{}
	p.DestST = all[2]
	p.DestID = all[3]
	p.GateID = all[4]
	p.UserID = byteOrder.Uint32(all[5:])
	p.Ver = byteOrder.Uint16(all[9:])
	p.Hold = byteOrder.Uint32(all[11:])
	p.Cmd = byteOrder.Uint16(all[15:])
	p.Body = all[17:]
	p.All = all
	return p, nil
}

func WsWrite(r io.ReadWriter, bs []byte) error {
	return wsutil.WriteServerBinary(r, bs)
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
	p.DestID = all[3]
	p.GateID = all[4]
	p.UserID = byteOrder.Uint32(all[5:])
	p.Ver = byteOrder.Uint16(all[9:])
	p.Hold = byteOrder.Uint32(all[11:])
	p.Cmd = byteOrder.Uint16(all[15:])
	p.Body = all[17:]
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
