package local

import (
	"frbg/codec"
	core "frbg/core"
	"frbg/def"
	"frbg/examples/pb"
	"log"

	"google.golang.org/protobuf/proto"
)

var serverType uint8

type Input struct {
	c core.IConn
	*codec.Message
}

func NewInput(conn core.IConn, msg *codec.Message) *Input {
	return &Input{
		c:       conn,
		Message: msg,
	}
}

func (r *Input) Client() core.IConn {
	return r.c
}

func (r *Input) Rpc(msg proto.Message) error {
	data := codec.NewMessage(r.Cmd, msg)
	data.Seq = r.Seq
	return r.c.Write(data)
}

func (i *Input) Response(uid uint32, cmd uint16, msg proto.Message) error {
	// 封装消息
	var data *codec.Message
	if serverType == def.ST_Gate {
		// 网关服务器类型，直接封装为cmd
		data = codec.NewMessage(cmd, msg)
		data.Seq = i.Seq
		log.Printf("Response uid:%d, cmd:%d, msg:%v", uid, cmd, data)
	} else {
		// 其他服务器类型，封装为PacketOut
		payload, err := proto.Marshal(msg)
		if err != nil {
			log.Printf("Response proto.Marshal() err:%s", err.Error())
			return err
		}
		data = codec.NewMessage(def.PacketOut, &pb.PacketOut{
			Uid:     []uint32{uid},
			Cmd:     uint32(cmd),
			Payload: payload,
		})
		data.Seq = i.Seq
		log.Printf("Response uid:%d, cmd:%d, msg:%v", uid, cmd, data)
	}
	return i.c.Write(data)
}
