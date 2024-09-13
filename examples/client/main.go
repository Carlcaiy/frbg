package main

import (
	"flag"
	"fmt"
	"frbg/def"
	"frbg/examples/cmd"
	"frbg/examples/proto"
	"frbg/parser"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
)

var uid uint32 = 123
var roomId uint32 = 0
var gateid int = 6666

var req = make(map[int]func())
var des = make(map[int]string)

func add(cmd int, str string, f func()) {
	req[cmd] = f
	des[cmd] = str
}

func auth() {
	u := "https:localhost:8080"
	client := &http.Client{}
	response, err := client.PostForm(u, url.Values{"name": []string{"caiyunfeng"}, "password": []string{"123123"}})
	if err != nil {
		log.Println("Error sending request:", err)
		return
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Println("error reading response body:", err)
		return
	}

	log.Println("Response", string(body))
}

func main() {
	var uid64 = 0
	flag.IntVar(&uid64, "u", 123, "-u 123")
	flag.IntVar(&gateid, "p", 6666, "-p 6666")
	flag.Parse()
	uid = uint32(uid64)
	auth()

	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", gateid))
	if err != nil {
		panic(err)
	}
	add(cmd.Login, "请求登录", func() {
		bs := parser.NewMessage(uid, def.ST_Gate, cmd.Login, 1, &proto.ReqGateLogin{}).Pack()
		conn.Write(bs)
	})
	add(cmd.ReqRoomList, "请求房间列表", func() {
		bs := parser.NewMessage(uid, def.ST_Hall, cmd.ReqRoomList, 1, &proto.ReqRoomList{}).Pack()
		conn.Write(bs)
	})
	add(cmd.ReqEnterRoom, "请求进房间", func() {
		log.Println("请输入进入的房间")
		roomId := uint32(0)
		fmt.Scanln(&roomId)
		bs := parser.NewMessage(uid, def.ST_Hall, cmd.ReqEnterRoom, 1, &proto.ReqEnterRoom{}).Pack()
		conn.Write(bs)
	})
	add(cmd.Tap, "猜测一个数值", func() {
		log.Println("请输入键入的数值：")
		num := int32(0)
		fmt.Scanln(&num)
		bs := parser.NewMessage(uid, def.ST_Hall, cmd.Tap, 1, &proto.Tap{
			RoomId: roomId,
			Tap:    num,
		}).Pack()
		conn.Write(bs)
	})
	add(cmd.ReqLeaveRoom, "请求离开房间", func() {
		bs := parser.NewMessage(uid, def.ST_Hall, cmd.ReqLeaveRoom, 1, &proto.ReqLeaveRoom{
			RoomId: roomId,
		}).Pack()
		conn.Write(bs)
	})

	go func() {
		show_op()
		for {
			msg, err := parser.Parse(conn)
			if err != nil {
				break
			}
			log.Println("receive msg:", msg.Cmd)
			switch msg.Cmd {
			case cmd.Login:
				p := new(proto.ResGateLogin)
				err := msg.UnPack(p)
				if err != nil {
					log.Println(err)
					continue
				}
				if p.Ret == 0 {
					log.Println("login success")
					show_op()
				} else if p.Ret == 1 {
					log.Println("error: relogin")
				}
			case cmd.GateKick:
				p := new(proto.GateKick)
				msg.UnPack(p)
				if p.Type == proto.KickType_Unknow {
					log.Println("游戏服务关闭")
				} else if p.Type == proto.KickType_Squeeze {
					log.Println("挤号")
				} else if p.Type == proto.KickType_GameNotFound {
					log.Println("服务未发现")
				} else {
					log.Println("踢出：位置错误")
				}
			case cmd.ResRoomList:
				p := new(proto.ResRoomList)
				msg.UnPack(p)
				log.Println(p.String())
				show_op()
			case cmd.ResEnterRoom:
				p := new(proto.ResEnterRoom)
				msg.UnPack(p)
				for _, uids := range p.Uids {
					log.Println("房间玩家", uids)
				}
			case cmd.GameStart:
				p := new(proto.StartGame)
				msg.UnPack(p)
				log.Println(p.String())
			case cmd.SyncData:
				p := new(proto.SyncData)
				msg.UnPack(p)
				log.Println(p.String())
				roomId = p.RoomId
			case cmd.GameOver:
				p := new(proto.GameOver)
				msg.UnPack(p)
				log.Println(p.String())
			case cmd.Tap:
				p := new(proto.Tap)
				msg.UnPack(p)
				log.Println(p.String())
			case cmd.Round:
				show_op()
			case cmd.CountDown:
				log.Println("剩余时间10")
				// show_op()
			}
		}
	}()

	sig := make(chan os.Signal, 10)
	signal.Notify(sig, syscall.SIGTERM)
	<-sig

	show_op()
}

func show_op() {
	for {
		for cmd, des := range des {
			log.Printf("%d: %s\n", cmd, des)
		}
		log.Println("选择想执行的操作")
		cmd := 0
		fmt.Scanln(&cmd)
		if f, ok := req[cmd]; ok {
			f()
			break
		}
	}
}
