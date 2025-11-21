package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"frbg/codec"
	"frbg/def"
	"frbg/examples/pb"
	"frbg/network"
	"log"
	"net"

	protobuf "google.golang.org/protobuf/proto"

	"github.com/gobwas/ws"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var uid int = 100005
var port int = 6666
var conn net.Conn
var err error

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	flag.IntVar(&uid, "u", uid, "-u 123")
	flag.IntVar(&port, "p", 6666, "-p 8080")
	flag.Parse()

	conn, _, _, err = ws.Dial(context.Background(), fmt.Sprintf("ws://localhost:%d", port))
	if err != nil {
		must(err)
		return
	}
	defer func() {
		conn.Close()
	}()

	must(rpc(def.ST_Gate, def.Login, &pb.LoginReq{Uid: uint32(uid), Password: "123123", From: 1, GateId: 1}, &pb.LoginRsp{}))
	must(rpc(def.ST_Hall, def.GetGameList, &pb.GetGameListReq{Uid: uint32(uid)}, &pb.GetGameListRsp{}))
	getRoomListRsp := &pb.GetRoomListRsp{}
	must(rpc(def.ST_Hall, def.GetRoomList, &pb.GetRoomListReq{Uid: uint32(uid), GameId: def.MahjongBanbisan}, getRoomListRsp))
	must(rpc(def.ST_Hall, def.EnterRoom, &pb.EnterRoomReq{Uid: uint32(uid), RoomId: uint32(getRoomListRsp.Rooms[0].RoomId)}, &pb.GetRoomListRsp{}))
}

func must(e error) {
	if e != nil {
		log.Fatalln(e)
	}
}

func rpc(svid uint8, cmd uint16, req protoreflect.ProtoMessage, rsp protoreflect.ProtoMessage) error {
	var msg *codec.Message
	if svid == def.ST_Gate || svid == def.ST_WsGate {
		msg = codec.NewMessage(cmd, req)
	} else {
		bs, _ := protobuf.Marshal(req)
		msg = codec.NewMessage(def.PacketIn, &pb.PacketIn{
			Svid:    uint32(network.Svid(svid, 1)),
			Cmd:     uint32(cmd),
			Payload: bs,
		})
	}
	if err = codec.WsWrite(conn, msg); err != nil {
		return err
	}
	log.Printf("start read scmd:%d", cmd)
	msg, err := codec.WsRead(conn)
	if err != nil {
		return err
	}
	log.Printf("read success:%s", msg.String())
	err = msg.Unpack(rsp)
	if err != nil {
		return err
	}
	bsi, _ := json.MarshalIndent(rsp, "", "  ")
	log.Println(string(bsi))
	return nil
}
