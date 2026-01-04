package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"frbg/codec"
	core "frbg/core"
	"frbg/def"
	"frbg/examples/pb"
	"frbg/mj"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gobwas/ws"
	"google.golang.org/protobuf/proto"
)

var uid int = 100005
var port int = 6666
var conn net.Conn
var err error
var playerData *pb.StartGameRsp
var getRoomListRsp = &pb.GetRoomListRsp{}
var mjs []uint8
var errch = make(chan error, 1)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	flag.IntVar(&uid, "u", uid, "-u 123")
	flag.IntVar(&port, "p", 6666, "-p 8080")
	flag.Parse()

	conn, _, _, err = ws.Dial(context.Background(), fmt.Sprintf("ws://localhost:%d", port+1))
	if err != nil {
		log.Println(err)
		return
	}
	log.Printf("connect to server %d success", port+1)
	go Tick()
	go Loop()
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	select {
	case sig := <-ch:
		if sig == syscall.SIGQUIT || sig == syscall.SIGTERM || sig == syscall.SIGINT {
			log.Println("signal kill")
		}
	case err := <-errch:
		log.Println(err)
	}
	fmt.Println("close conn", conn.Close())
}

func logdata(data proto.Message, msg *codec.Message) {
	err = msg.Unpack(data)
	if err != nil {
		log.Println(err)
		return
	}
	bsi, _ := json.MarshalIndent(data, "", "  ")
	log.Println(string(bsi))
}

func Tick() {
	for {
		time.Sleep(5 * time.Second)
		conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
		msg := codec.AcquireMessage()
		msg.SetFlags(codec.FlagsHeartBeat)
		if err := codec.WsWrite(conn, msg); err != nil {
			log.Printf("send error:%s", err.Error())
			errch <- err
			return
		}
		codec.ReleaseMessage(msg)
	}
}

func Loop() {
	send(def.ST_Gate, def.Login, &pb.LoginReq{Uid: uint32(uid), Password: "123123", From: 1, GateId: 1})
	for {
		msg, err := codec.WsRead(conn)
		if err != nil {
			errch <- err
			break
		}
		switch msg.Cmd {
		case def.Error:
			rsp := new(pb.CommonRsp)
			logdata(rsp, msg)
			errch <- fmt.Errorf("error code:%d msg:%s", rsp.Code, rsp.Msg)
			return
		case def.Login:
			rsp := new(pb.LoginRsp)
			logdata(rsp, msg)
			send(def.ST_Hall, def.GetGameList, &pb.GetGameListReq{Uid: uint32(uid)})
		case def.GetGameList:
			rsp := new(pb.GetGameListRsp)
			logdata(rsp, msg)
			send(def.ST_Hall, def.GetRoomList, &pb.GetRoomListReq{Uid: uint32(uid), GameId: def.SID_MahjongBanbisan})
		case def.GetRoomList:
			logdata(getRoomListRsp, msg)
			send(def.ST_Hall, def.EnterRoom, &pb.EnterRoomReq{
				Uid:    uint32(uid),
				GateId: 1,
				GameId: def.SID_MahjongBanbisan,
				RoomId: uint32(getRoomListRsp.Rooms[0].RoomId),
			})
		case def.StartGame:
			rsp := new(pb.StartGameRsp)
			logdata(rsp, msg)
			playerData = rsp
		case def.GameFaPai:
			rsp := new(pb.MjFaPai)
			logdata(rsp, msg)
			for i := range rsp.Fapai {
				if rsp.Fapai[i].Uid == uint32(uid) {
					mjs = append(mjs, uint8(rsp.Fapai[i].MjVal))
				}
			}
			if rsp.CanOp&mj.ChuPai == mj.ChuPai {
				send(def.ST_Game, def.OptGame, &pb.MjOpt{
					Uid:    uint32(uid),
					RoomId: playerData.RoomId,
					Op:     mj.ChuPai,
				})
			}
		case def.BcOpt:
			rsp := new(pb.MjOpt)
			logdata(rsp, msg)
			if rsp.Uid == uint32(uid) && rsp.Op == mj.ChuPai && rsp.Mj > 0 {
				for i, mj := range mjs {
					if int32(mj) == rsp.Mj {
						mjs = append(mjs[:i], mjs[i+1:]...)
						break
					}
				}
			}
			if rsp.CanOp&mj.GuoPai == mj.GuoPai {
				send(def.ST_Game, def.OptGame, &pb.MjOpt{
					Uid:    uint32(uid),
					RoomId: playerData.RoomId,
					Op:     mj.GuoPai,
				})
			} else if rsp.CanOp&mj.ChuPai == mj.ChuPai {
				send(def.ST_Game, def.OptGame, &pb.MjOpt{
					Uid:    uint32(uid),
					RoomId: playerData.RoomId,
					Op:     mj.ChuPai,
				})
			}
		case def.Reconnect:
			rsp := new(pb.DeskSnapshot)
			logdata(rsp, msg)
			for _, info := range rsp.Info {
				if info.Uid == uint32(uid) {
					mjs = append(mjs, info.Hands...)
				}
			}
		}
	}
}

func send(svid uint8, cmd uint16, req proto.Message) {
	var msg *codec.Message
	if svid == def.ST_Gate {
		msg = codec.NewMessage(cmd, req)
		log.Printf("send gate cmd:%d", cmd)
	} else {
		bs, _ := proto.Marshal(req)
		packet := &pb.PacketIn{
			Svid:    uint32(core.Svid(svid, def.SID_MahjongBanbisan)),
			Cmd:     uint32(cmd),
			Payload: bs,
		}
		msg = codec.NewMessage(def.PacketIn, packet)
		log.Printf("send packetIn cmd:%d, svid:%d", packet.Cmd, packet.Svid)
	}

	if err = conn.SetWriteDeadline(time.Now().Add(3 * time.Second)); err != nil {
		log.Printf("set write deadline error:%s", err.Error())
		errch <- err
		return
	}
	if err = codec.WsWrite(conn, msg); err != nil {
		log.Printf("send error:%s", err.Error())
		errch <- err
	}
}
