package local

import (
	"frbg/examples/pb"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GRPC struct {
	pb.RPCClient
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
		log.Printf("connect error:%s", err.Error())
		return
	}
	g.client = client
	g.RPCClient = pb.NewRPCClient(client)
}

func (g *GRPC) Close() {
	g.client.Close()
}
