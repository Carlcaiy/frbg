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

	"github.com/gobwas/ws"
	"google.golang.org/protobuf/proto"
)

var uid int = 100005
var port int = 6666
var conn net.Conn
var err error
var gameData *pb.StartGameRsp
var mjs []uint8

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
	defer func() {
		conn.Close()
	}()

	must(rpc(def.ST_Gate, def.Login, &pb.LoginReq{Uid: uint32(uid), Password: "123123", From: 1, GateId: 1}, &pb.LoginRsp{}))
	must(rpc(def.ST_Hall, def.GetGameList, &pb.GetGameListReq{Uid: uint32(uid)}, &pb.GetGameListRsp{}))
	getRoomListRsp := &pb.GetRoomListRsp{}
	must(rpc(def.ST_Hall, def.GetRoomList, &pb.GetRoomListReq{Uid: uint32(uid), GameId: def.SID_MahjongBanbisan}, getRoomListRsp))
	must(send(def.ST_Hall, def.EnterRoom, &pb.EnterRoomReq{
		Uid:    uint32(uid),
		GateId: 1,
		GameId: def.SID_MahjongBanbisan,
		RoomId: uint32(getRoomListRsp.Rooms[0].RoomId),
	}))

	for {
		msg, err := codec.WsRead(conn)
		if err != nil {
			log.Println(err)
			break
		}

		switch msg.Cmd {
		case def.StartGame:
			rsp := new(pb.StartGameRsp)
			logdata(rsp, msg)
			gameData = rsp
			send(def.ST_Game, def.SyncStatus, &pb.SyncStatus{
				Uid:    uint32(uid),
				RoomId: gameData.RoomId,
				Cmd:    def.StartGame,
			})
		case def.GameFaPai:
			rsp := new(pb.FaPai)
			logdata(rsp, msg)
			for i := range rsp.Fapai {
				if rsp.Fapai[i].Uid == uint32(uid) {
					mjs = append(mjs, uint8(rsp.Fapai[i].MjVal))
				}
			}
			send(def.ST_Game, def.SyncStatus, &pb.SyncStatus{
				Uid:    uint32(uid),
				RoomId: gameData.RoomId,
				Cmd:    def.GameFaPai,
			})
		case def.NotifyChuPai:
			rsp := new(pb.MjOpt)
			logdata(rsp, msg)
			send(def.ST_Game, def.OptGame, &pb.MjOpt{
				Uid:    uint32(uid),
				RoomId: gameData.RoomId,
				Op:     mj.ChuPai,
				Mj:     int32(mjs[0]),
			})
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
					RoomId: gameData.RoomId,
					Op:     mj.GuoPai,
				})
			} else if rsp.CanOp&mj.ChuPai == mj.ChuPai {
				send(def.ST_Game, def.OptGame, &pb.MjOpt{
					Uid:    uint32(uid),
					RoomId: gameData.RoomId,
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

func must(e error) {
	if e != nil {
		log.Fatalln(e)
	}
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

func rpc(svid uint8, cmd uint16, req proto.Message, rsp proto.Message) error {
	var msg *codec.Message
	if svid == def.ST_Gate {
		msg = codec.NewMessage(cmd, req)
	} else {
		bs, _ := proto.Marshal(req)
		msg = codec.NewMessage(def.PacketIn, &pb.PacketIn{
			Svid:    uint32(core.Svid(svid, 1)),
			Cmd:     uint32(cmd),
			Payload: bs,
		})
	}
	log.Printf("send scmd:%d", cmd)
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

func send(svid uint8, cmd uint16, req proto.Message) error {
	var msg *codec.Message
	if svid == def.ST_Gate {
		msg = codec.NewMessage(cmd, req)
	} else {
		bs, _ := proto.Marshal(req)
		msg = codec.NewMessage(def.PacketIn, &pb.PacketIn{
			Svid:    uint32(core.Svid(svid, def.SID_MahjongBanbisan)),
			Cmd:     uint32(cmd),
			Payload: bs,
		})
	}
	if err = codec.WsWrite(conn, msg); err != nil {
		return err
	}
	return nil
}
