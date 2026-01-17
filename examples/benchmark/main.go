package main

import (
	"fmt"
	"frbg/codec"
	"frbg/core"
	"frbg/def"
	"frbg/examples/pb"
	"io"
	"net"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
)

var wg sync.WaitGroup

func main() {
	for i := 100; i < 5000; i++ {
		wg.Add(1)
		go gate(i)
	}
	wg.Wait()
}

func gate(uid int) {
	defer wg.Done()
	conn, err := net.DialTCP("tcp", nil, &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 6666})
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()
	for i := 0; i < 1000; i++ {
		err = codec.TcpWrite(conn, codec.NewMessage(def.Login, &pb.LoginReq{
			Uid:    uint32(uid),
			GateId: 1,
		}))
		if err != nil {
			fmt.Println(err)
			return
		}
		codec.TcpRead(conn)
		err = Write(conn, def.ST_Hall, def.GetGameList, &pb.GetGameListReq{
			Uid:      uint32(uid),
			GateId:   1,
			ServerId: 1,
		})
		if err != nil {
			fmt.Println(err)
			return
		}
		msg, err := codec.TcpRead(conn)
		if err != nil {
			fmt.Println(err)
			return
		}
		rsp := new(pb.GetGameListRsp)
		if err = msg.Unpack(rsp); err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(uid, rsp.String())
	}
}

func hall() {
	defer wg.Done()
	conn, err := net.DialTCP("tcp", nil, &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 6676})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("1")
	time.Sleep(time.Second * 1)
	defer conn.Close()
	err = codec.TcpWrite(conn, codec.NewMessage(def.GetGameList, &pb.GetGameListReq{
		Uid:      10001,
		GateId:   1,
		ServerId: 1,
	}))
	if err != nil {
		fmt.Println(err)
		return
	}
	msg, err := codec.TcpRead(conn)
	if err != nil {
		fmt.Println(err)
		return
	}
	// 解析rsp
	rspMsg := new(pb.GetGameListRsp)
	Read(rspMsg, msg)
	fmt.Println(rspMsg.String())
}

func game() {
	conn, err := net.DialTCP("tcp", nil, &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8080})
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()
	err = codec.TcpWrite(conn, codec.NewMessage(def.GameStatus, &pb.GameStatusReq{
		Uid: 10001,
	}))
	if err != nil {
		fmt.Println(err)
		return
	}
	msg, err := codec.TcpRead(conn)
	if err != nil {
		fmt.Println(err)
		return
	}
	rsp := new(pb.GameStatusRsp)
	Read(rsp, msg)
	fmt.Println(rsp.String())
}

func Write(r io.Writer, st uint8, cmd uint16, pro proto.Message) error {
	bs, _ := proto.Marshal(pro)
	req := &pb.PacketIn{
		Svid:    uint32(core.Svid(st, def.SID_MahjongBanbisan)),
		Cmd:     uint32(cmd),
		Payload: bs,
	}
	if err := codec.TcpWrite(r, codec.NewMessage(def.PacketIn, req)); err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func Read(pro proto.Message, msg *codec.Message) {
	rsp := new(pb.PacketOut)
	if err := msg.Unpack(rsp); err != nil {
		fmt.Println(err)
		return
	}
	// 解析rsp
	if err := proto.Unmarshal(rsp.Payload, pro); err != nil {
		fmt.Println(err)
		return
	}
}
