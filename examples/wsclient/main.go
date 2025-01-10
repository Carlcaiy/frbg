package main

import (
	"context"
	"encoding/json"
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
		must(err)
		return
	}
	defer func() {
		conn.Close()
	}()

	must(rpc(def.ST_Gate, cmd.Login, &proto.LoginReq{Password: "123123", From: 1, GateId: 1}, &proto.LoginRsp{}))
	must(rpc(def.ST_Hall, cmd.GetGameList, &proto.GetGameListReq{}, &proto.GetGameListRsp{}))
	slots := &proto.EnterSlotsRsp{}
	must(rpc(def.ST_Hall, cmd.EnterSlots, &proto.EnterSlotsReq{GameId: def.SlotsFu}, slots))

	for {
		// must(rpc(def.ST_Hall, cmd.Test, &proto.Test{Uid: uint32(uid), StartTime: time.Now().Unix()}, &proto.Test{}))
		rsp := &proto.SlotsSpinRsp{}
		must(rpc(def.ST_Hall, cmd.SpinSlots, &proto.SlotsSpinReq{Uid: int32(uid), GameId: def.SlotsFu, Bet: slots.Bet[0], Level: slots.Level[0]}, rsp))
		bs, _ := json.MarshalIndent(rsp, "", "  ")
		log.Println(string(bs))
		time.Sleep(time.Second)
	}
}

func must(e error) {
	if e != nil {
		log.Fatalln(e)
	}
}

func rpc(svrt uint8, scmd uint16, req protoreflect.ProtoMessage, rsp protoreflect.ProtoMessage) error {
	log.Printf("svrt:%d scmd:%d req:%s", svrt, scmd, req)
	bs := parser.NewMessage(uint32(uid), svrt, scmd, 1, req).Pack()
	if err = parser.WsWrite(conn, bs); err != nil {
		return err
	}
	log.Printf("start read scmd:%d", scmd)
	msg, err := parser.WsRead(conn)
	if err != nil {
		return err
	}
	log.Printf("read success:%d", scmd)
	err = msg.Unpack(rsp)
	if err != nil {
		return err
	}
	if cmd.Login == scmd {
		uid = int(msg.UserID)
	}
	log.Println(rsp)
	return nil
}
