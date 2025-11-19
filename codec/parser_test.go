package codec

import (
	"bytes"
	"fmt"
	"frbg/examples/proto"
	"testing"
)

func TestWs(t *testing.T) {
	rw := bytes.NewBuffer([]byte{})
	WsWrite(rw, NewMessage(0x01, uint8(0x02), 0x02, &proto.GameInfo{GameId: 111}))
	msg, err := WsRead(rw)
	fmt.Println(msg, err)
	req := new(proto.GameInfo)
	err = msg.Unpack(req)
	fmt.Println(err, req)
}

func TestTcp(t *testing.T) {
	rw := bytes.NewBuffer([]byte{})
	TcpWrite(rw, NewMessage(0x01, uint8(0x02), 0x02, &proto.GameInfo{GameId: 111}))
	msg, err := TcpRead(rw)
	fmt.Println(msg, err)
	req := new(proto.GameInfo)
	err = msg.Unpack(req)
	fmt.Println(err, req)
}
