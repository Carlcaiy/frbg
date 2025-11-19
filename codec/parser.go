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
	HeaderLen = 12
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
	p.DestType = all[2]
	p.DestId = all[3]
	p.Ver = byteOrder.Uint16(all[4:])
	p.Type = byteOrder.Uint32(all[6:])
	p.Cmd = byteOrder.Uint16(all[10:])
	p.Body = all[12:]
	p.All = all
	return p, nil
}

func WsWrite(r io.ReadWriter, msg *Message) error {
	return wsutil.WriteServerBinary(r, msg.Pack())
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
	p.DestType = all[2]
	p.DestId = all[3]
	p.Ver = byteOrder.Uint16(all[4:])
	p.Type = byteOrder.Uint32(all[6:])
	p.Cmd = byteOrder.Uint16(all[10:])
	p.Body = all[12:]
	p.All = all
	return
}

func TcpWrite(r io.Writer, msg *Message) error {
	_, err := r.Write(msg.Pack())
	return err
}

func Pack(dest uint8, cmd uint16, pro proto.Message) ([]byte, error) {
	body, err := proto.Marshal(pro)
	if err != nil {
		return nil, err
	}
	msg := &Message{
		DestType: dest,
		Cmd:      cmd,
		Body:     body,
	}
	return msg.Pack(), nil
}
