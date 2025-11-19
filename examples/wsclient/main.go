package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"frbg/codec"
	"frbg/def"
	"frbg/examples/cmd"
	"frbg/examples/proto"
	"log"
	"net"

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

	must(rpc(def.ST_Gate, cmd.Login, &proto.LoginReq{Uid: uint32(uid), Password: "123123", From: 1, GateId: 1}, &proto.LoginRsp{}))
	must(rpc(def.ST_Hall, cmd.GetGameList, &proto.GetGameListReq{Uid: uint32(uid)}, &proto.GetGameListRsp{}))
	getRoomListRsp := &proto.GetRoomListRsp{}
	must(rpc(def.ST_Hall, cmd.GetRoomList, &proto.GetRoomListReq{Uid: uint32(uid), GameId: def.MahjongBanbisan}, getRoomListRsp))
	must(rpc(def.ST_Hall, cmd.EnterRoom, &proto.EnterRoomReq{Uid: uint32(uid), RoomId: uint32(getRoomListRsp.Rooms[0].RoomId)}, &proto.GetRoomListRsp{}))
}

func must(e error) {
	if e != nil {
		log.Fatalln(e)
	}
}

func rpc(svrt uint8, scmd uint16, req protoreflect.ProtoMessage, rsp protoreflect.ProtoMessage) error {
	log.Printf("svrt:%d scmd:%d req:%s", svrt, scmd, req)
	if err = codec.WsWrite(conn, codec.NewMessage(svrt, 1, scmd, req)); err != nil {
		return err
	}
	log.Printf("start read scmd:%d", scmd)
	msg, err := codec.WsRead(conn)
	if err != nil {
		return err
	}
	log.Printf("read success:%d", scmd)
	err = msg.Unpack(rsp)
	if err != nil {
		return err
	}
	bsi, _ := json.MarshalIndent(rsp, "", "  ")
	log.Println(string(bsi))
	return nil
}
