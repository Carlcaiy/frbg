package local

import (
	"frbg/examples/proto"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GRPC struct {
	proto.RPCClient
	client *grpc.ClientConn
	addr   string
}

func NewGRPC(addr string) *GRPC {
	return &GRPC{
		addr: addr,
	}
}

func (g *GRPC) Init() {
	client, err := grpc.NewClient("addr", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Println(err)
		return
	}
	g.client = client
	g.RPCClient = proto.NewRPCClient(client)
}

func (g *GRPC) Close() {
	g.client.Close()
}
