package main

import (
	"context"
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
		log.Printf("connect error:%s", err.Error())
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
		log.Printf("error:%s", err.Error())
	}
	fmt.Println("close conn", conn.Close())
}

func logdata(data proto.Message, msg *codec.Message) {
	err = msg.Unpack(data)
	if err != nil {
		log.Printf("unpack error:%s", err.Error())
		return
	}
	// bsi, _ := json.MarshalIndent(data, "", "  ")
	// log.Printf("recv:%s", string(bsi))
}

func Tick() {
	for {
		time.Sleep(5 * time.Second)
		conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
		msg := codec.AcquireMessage()
		msg.SetFlags(codec.FlagsHeartBeat)
		if err := codec.WsWriteBySide(conn, ws.StateClientSide, msg); err != nil {
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
		msg, err := codec.WsReadBySide(conn, ws.StateClientSide)
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
			// send(def.ST_Gate, def.Logout, &pb.LogoutReq{Uid: uint32(uid)})
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
			log.Printf("uid:%d pai:%v", uid, mjs)
			if mj.HasOp(rsp.CanOp, mj.ChuPai) {
				send(def.ST_Game, def.OptGame, &pb.MjOpt{
					Uid:    uint32(uid),
					RoomId: playerData.RoomId,
					Op:     mj.ChuPai,
					Mj:     int32(mjs[0]),
				})
			}
		case def.BcOpt:
			rsp := new(pb.MjOpt)
			logdata(rsp, msg)
			if rsp.Uid == uint32(uid) {
				if rsp.Op == mj.ChuPai && rsp.Mj > 0 {
					for i, mj := range mjs {
						if int32(mj) == rsp.Mj {
							mjs = append(mjs[:i], mjs[i+1:]...)
							log.Printf("chupai mjs:%v pai:%d", mjs, rsp.Mj)
							break
						}
					}
				} else if rsp.Op == mj.MoPai && rsp.Mj > 0 {
					mjs = append(mjs, uint8(rsp.Mj))
					log.Printf("mopai mjs:%v pai:%d", mjs, rsp.Mj)
				}
			}
			if mj.HasOp(rsp.CanOp, mj.GuoPai) {
				send(def.ST_Game, def.OptGame, &pb.MjOpt{
					Uid:    uint32(uid),
					RoomId: playerData.RoomId,
					Op:     mj.GuoPai,
				})
			} else if mj.HasOp(rsp.CanOp, mj.ChuPai) {
				send(def.ST_Game, def.OptGame, &pb.MjOpt{
					Uid:    uint32(uid),
					RoomId: playerData.RoomId,
					Op:     mj.ChuPai,
					Mj:     int32(mjs[0]),
				})
				log.Printf("send chupai pai:%d", mjs[0])
			}
		case def.Reconnect:
			rsp := new(pb.DeskSnapshot)
			logdata(rsp, msg)
			playerData = &pb.StartGameRsp{
				RoomId: rsp.RoomId,
			}
			for _, info := range rsp.Info {
				if info.Uid == uint32(uid) {
					mjs = append(mjs, info.Hands...)
					log.Printf("mjs:%v", mjs)
					if mj.HasOp(info.CanOp, mj.ChuPai) {
						send(def.ST_Game, def.OptGame, &pb.MjOpt{
							Uid:    uint32(uid),
							RoomId: playerData.RoomId,
							Op:     mj.ChuPai,
							Mj:     int32(mjs[0]),
						})
					} else if mj.HasOp(info.CanOp, mj.GuoPai) {
						send(def.ST_Game, def.OptGame, &pb.MjOpt{
							Uid:    uint32(uid),
							RoomId: playerData.RoomId,
							Op:     mj.GuoPai,
						})
					}
				}
			}
		case def.GameOver:
			rsp := new(pb.GameOver)
			logdata(rsp, msg)
		case def.Logout:
			log.Printf("logout uid:%d", uid)
			send(def.ST_Gate, def.Login, &pb.LoginReq{Uid: uint32(uid), Password: "123123", From: 1, GateId: 1})
		default:
			log.Printf("recv unknown cmd:%d", msg.Cmd)
		}
	}
}

func send(svid uint8, cmd uint16, req proto.Message) {
	// bsi, _ := json.MarshalIndent(req, "", "  ")
	// log.Printf("send:%s", string(bsi))
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
		// log.Printf("send packetIn cmd:%d, svid:%d", packet.Cmd, packet.Svid)
	}

	if err = conn.SetWriteDeadline(time.Now().Add(3 * time.Second)); err != nil {
		log.Printf("set write deadline error:%s", err.Error())
		errch <- err
		return
	}

	if err = codec.WsWriteBySide(conn, ws.StateClientSide, msg); err != nil {
		log.Printf("send error:%s", err.Error())
		errch <- err
	}
}
