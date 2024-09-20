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
	"time"

	"github.com/gobwas/ws"
)

var uid int = 123
var gateid int = 6667

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	flag.IntVar(&uid, "u", 123, "-u 123")
	flag.IntVar(&gateid, "p", 6667, "-p 8080")
	flag.Parse()

	conn, _, _, err := ws.Dial(context.Background(), fmt.Sprintf("ws://localhost:%d", gateid))
	if err != nil {
		log.Println(err)
		return
	}
	bs := parser.NewMessage(uint32(uid), def.ST_Gate, cmd.Login, 1, &proto.LoginReq{Password: "123123", From: 1, GateId: 1}).Pack()
	if err = parser.WsWrite(conn, bs); err != nil {
		log.Println("send msg:", err)
		return
	}
	msg, err := parser.WsRead(conn)
	if err != nil {
		log.Println(err)
		return
	}
	rsp := new(proto.LoginRsp)
	msg.Unpack(rsp)
	log.Println(rsp)
	for {
		bs := parser.NewMessage(uint32(uid), def.ST_Hall, cmd.Test, 1, &proto.Test{Uid: uint32(uid), StartTime: time.Now().Unix()}).Pack()
		if err = parser.WsWrite(conn, bs); err != nil {
			log.Println("send msg:", err)
			break
		}

		test := new(proto.Test)
		msg, err := parser.WsRead(conn)
		if err != nil {
			log.Println(err)
			break
		}
		msg.Unpack(test)
		log.Println("receive msg:", test)

		time.Sleep(time.Second)
	}
}
