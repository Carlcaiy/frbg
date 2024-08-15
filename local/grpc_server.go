package local

import (
	"frbg/examples/proto"
	"log"
	"net"

	"google.golang.org/grpc"
)

type GRPCServer struct {
	handle proto.RPCServer
	server *grpc.Server
	addr   string
}

func NewGRPCServer(addr string, method proto.RPCServer) *GRPCServer {
	return &GRPCServer{
		addr:   addr,
		handle: method,
	}
}

func (g *GRPCServer) Init() {
	net, err := net.Listen("tcp", g.addr)
	if err != nil {
		log.Println(err)
		return
	}
	g.server = grpc.NewServer()
	err = g.server.Serve(net)
	if err != nil {
		log.Println(err)
		return
	}
}

func (g *GRPCServer) Close() {
	g.server.Stop()
}
