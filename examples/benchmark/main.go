package main

import (
	"flag"
	"fmt"
	"frbg/codec"
	"frbg/def"
	"frbg/examples/pb"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var num int = 100
var gateid int = 6666
var close bool = false

func main() {
	flag.IntVar(&num, "n", 100, "-n 100")
	flag.IntVar(&gateid, "p", 6666, "-p 6666")
	flag.Parse()

	wg := &sync.WaitGroup{}
	for u := 100; u < num+100; u++ {
		wg.Add(1)
		go func(uid uint32) {
			log.Println("新增连接", uid)
			conn, err := net.Dial("tcp", fmt.Sprintf(":%d", gateid))
			if err != nil {
				panic(err)
			}
			var t time.Duration = 0
			var c int = 0
			req := func(servetType def.ServerType) {
				msg := &codec.Message{}
				bs, _ := msg.PackWith(def.Test, &pb.Test{
					Uid:       uid,
					StartTime: time.Now().UnixNano(),
				})
				conn.Write(bs)
			}
			req(def.ST_Gate)
			for !close {
				msg, err := codec.TcpRead(conn)
				if err != nil {
					break
				}
				log.Println("receive msg:", msg.Cmd)
				switch msg.Cmd {
				case def.Test:
					p := new(pb.Test)
					err := msg.Unpack(p)
					if err != nil {
						log.Println(err)
						continue
					}
					p.EndTime = time.Now().UnixNano()
					t += time.Duration(p.EndTime - p.StartTime)
					c++
					req(def.ST_Hall)
				}
			}
			conn.Close()
			log.Printf("total=%v count=%v single=%v\n", t, c, time.Duration(int(t)/c))
			wg.Done()
		}(uint32(u))
		time.Sleep(time.Millisecond)
	}

	sig := make(chan os.Signal, 10)
	signal.Notify(sig, syscall.SIGTERM, os.Interrupt)
	<-sig
	close = true
	wg.Wait()
}
