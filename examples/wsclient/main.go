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
var gateid int = 8080

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	flag.IntVar(&uid, "u", 123, "-u 123")
	flag.IntVar(&gateid, "p", 8080, "-p 8080")
	flag.Parse()

	conn, _, _, err := ws.Dial(context.Background(), fmt.Sprintf("ws://localhost:%d", gateid))
	if err != nil {
		log.Println(err)
		return
	}
	for {
		bs := parser.NewMessage(uint32(uid), def.ST_Gate, cmd.Test, 1, &proto.Test{Uid: 11111, StartTime: time.Now().Unix()}).Pack()
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
		msg.UnPack(test)
		log.Println("receive msg:", test)

		time.Sleep(time.Second)
	}
}
