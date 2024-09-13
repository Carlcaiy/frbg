package parser

import (
	"fmt"
	"io"

	"github.com/gobwas/ws/wsutil"
)

func WsRead(r io.ReadWriter) (p *Message, err error) {
	all, err := wsutil.ReadServerBinary(r)
	if err != nil {
		return nil, err
	}
	length := byteOrder.Uint16(all[:2])
	if int(length) != len(all[2:]) {
		return nil, fmt.Errorf("binary unpack len error")
	}
	if length < HeaderLen {
		return nil, fmt.Errorf("binary unpack error")
	}
	p = &Message{}
	p.DestST = all[2]
	p.GateID = all[3]
	p.UserID = byteOrder.Uint32(all[4:])
	p.Ver = byteOrder.Uint16(all[8:])
	p.Hold = byteOrder.Uint32(all[10:])
	p.Cmd = byteOrder.Uint16(all[14:])
	p.Body = all[16:]
	p.All = all
	return p, nil
}

func WsWrite(r io.ReadWriter, bs []byte) error {
	return wsutil.WriteServerBinary(r, bs)
}
