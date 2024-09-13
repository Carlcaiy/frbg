package parser

import (
	"bytes"
	"frbg/def"
	"frbg/examples/proto"
	"log"
	"testing"
)

func TestParser(t *testing.T) {
	bs := NewMessage(101, def.ST_Hall, 1000, 1, &proto.HeartBeat{
		ServerType: uint32(def.ST_User),
		ServerId:   100,
	}).Pack()
	r := bytes.NewReader(bs)
	msg2, err := Parse(r)

	log.Println(msg2, err)

	bs2, _ := Pack(101, 0, 1000, &proto.HeartBeat{
		ServerType: uint32(def.ST_User),
		ServerId:   100,
	})
	r1 := bytes.NewReader(bs2)
	msg3, err := Parse(r1)

	log.Println(msg3, err)
}
