package local

import (
	"fmt"
	"frbg/codec"
	core "frbg/core"

	"google.golang.org/protobuf/proto"
)

type Input struct {
	core.IConn
	*codec.Message
}

func NewInput(conn core.IConn, msg *codec.Message) *Input {
	return &Input{
		IConn:   conn,
		Message: msg,
	}
}
func (r *Input) Rpc(msg proto.Message) error {
	bs, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal rpc msg failed: %w", err)
	}
	r.Payload = bs
	return r.Write(r.Message)
}
