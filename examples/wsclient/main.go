package main

import (
	"context"
	"flag"
	"fmt"
	"frbg/def"
	"frbg/examples/cmd"
	"frbg/examples/proto"
	"frbg/parser"
	"log"
	"net"
	"time"

	"github.com/gobwas/ws"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var uid int = 100005
var gateid int = 6667
var conn net.Conn
var err error

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	flag.IntVar(&uid, "u", uid, "-u 123")
	flag.IntVar(&gateid, "p", 6667, "-p 8080")
	flag.Parse()

	conn, _, _, err = ws.Dial(context.Background(), fmt.Sprintf("ws://localhost:%d", gateid))
	if err != nil {
		log.Println(err)
		return
	}
	defer func() {
		conn.Close()
	}()

	log.Fatalln(rpc(def.ST_Hall, cmd.SpinSlots, &proto.LoginReq{Password: "123123", From: 1, GateId: 1}, &proto.SlotsSpinRsp{}))
	log.Fatalln(rpc(def.ST_Hall, cmd.GetGameList, &proto.GetGameListReq{}, &proto.GetGameListRsp{}))
	log.Fatalln(rpc(def.ST_Hall, cmd.EnterSlots, &proto.EnterSlotsReq{GameId: def.SlotsFu}, &proto.EnterSlotsRsp{}))

	for {
		// log.Fatalln(rpc(def.ST_Hall, cmd.Test, &proto.Test{Uid: uint32(uid), StartTime: time.Now().Unix()}, &proto.Test{}))
		log.Fatalln(rpc(def.ST_Hall, cmd.SpinSlots, &proto.SlotsSpinReq{}, &proto.SlotsSpinRsp{}))

		time.Sleep(time.Second)
	}
}

func rpc(svrt uint8, cmd uint16, req protoreflect.ProtoMessage, rsp protoreflect.ProtoMessage) error {
	bs := parser.NewMessage(uint32(uid), svrt, cmd, 1, req).Pack()
	if err = parser.WsWrite(conn, bs); err != nil {
		log.Println("send msg:", err)
		return err
	}
	msg, err := parser.WsRead(conn)
	if err != nil {
		log.Println(err)
		return err
	}
	err = msg.Unpack(rsp)
	log.Println(rsp, err)
	return nil
}
