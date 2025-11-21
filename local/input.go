package local

import (
	"frbg/codec"
	"frbg/def"
	"frbg/examples/pb"
	"frbg/network"
	"log"

	"google.golang.org/protobuf/proto"
)

var serverType uint8

type Input struct {
	c *network.Conn
	*codec.Message
}

func NewInput(conn *network.Conn, msg *codec.Message) *Input {
	return &Input{
		c:       conn,
		Message: msg,
	}
}

func (r *Input) Client() *network.Conn {
	return r.c
}

func (i *Input) Response(uid uint32, cmd uint16, msg proto.Message) error {
	// 封装消息
	var data *codec.Message
	if serverType == def.ST_Gate || serverType == def.ST_WsGate {
		// 网关服务器类型，直接封装为cmd
		data = codec.NewMessage(cmd, msg)
		log.Printf("Response uid:%d, cmd:%d, msg:%v", uid, cmd, data)
	} else {
		// 其他服务器类型，封装为PacketOut
		payload, err := proto.Marshal(msg)
		if err != nil {
			log.Printf("Response proto.Marshal() err:%s", err.Error())
			return err
		}
		data = codec.NewMessage(def.PacketOut, &pb.PacketOut{
			Uid:     uid,
			Cmd:     uint32(cmd),
			Payload: payload,
		})
		log.Printf("Response uid:%d, cmd:%d, msg:%v", uid, def.PacketOut, data)
	}
	return i.c.Write(data)
}
