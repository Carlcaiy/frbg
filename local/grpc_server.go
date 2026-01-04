package local

import (
	"frbg/examples/pb"
	"log"
	"net"

	"google.golang.org/grpc"
)

type GRPCServer struct {
	handle pb.RPCServer
	server *grpc.Server
	addr   string
}

func NewGRPCServer(addr string, method pb.RPCServer) *GRPCServer {
	return &GRPCServer{
		addr:   addr,
		handle: method,
	}
}

func (g *GRPCServer) Init() {
	net, err := net.Listen("tcp", g.addr)
	if err != nil {
		log.Printf("listen error:%s", err.Error())
		return
	}
	g.server = grpc.NewServer()
	err = g.server.Serve(net)
	if err != nil {
		log.Printf("serve error:%s", err.Error())
		return
	}
}

func (g *GRPCServer) Close() {
	g.server.Stop()
}
