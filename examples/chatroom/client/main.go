package main

import (
	"flag"
	"fmt"
	"frbg/examples/chatroom/cmd"
	"frbg/examples/chatroom/proto"
	"frbg/network"
	"frbg/parser"

	pproto "google.golang.org/protobuf/proto"
)

type Client struct {
	uid    uint32
	roomid int32
	conn   *network.Conn
}

// Handler的初始化
func (c *Client) Init() {

}

func (c *Client) Parse(ccmd uint16, proto pproto.Message) {

}

// 消息路由
func (c *Client) Route(conn *network.Conn, msg *parser.Message) error {
	switch msg.Cmd() {
	case cmd.RespRoomList:
		data := &proto.RespRoomList{}
		msg.UnPack(data)
		fmt.Println(data)
		if buf, err := parser.Pack2(c.uid, cmd.ReqJoinRoom, &proto.ReqJoinRoom{RoomId: data.RoomIds[0]}); err != nil {
			fmt.Println(err)
			return err
		} else {
			conn.Write(buf)
		}
	case cmd.RespJoinRoom:
		data := &proto.RespJoinRoom{}
		msg.UnPack(data)
		c.roomid = data.RoomId
		fmt.Println(data)
		if buf, err := parser.Pack2(c.uid, cmd.SendMsg, &proto.Send{
			Uid:  int32(c.uid),
			Data: "你好啊",
			Room: data.RoomId,
		}); err != nil {
			fmt.Println(err)
			return err
		} else {
			conn.Write(buf)
		}
	case cmd.PushMsg:
		data := &proto.Send{}
		if err := msg.UnPack(data); err != nil {
			return err
		}
		fmt.Println(data)
	}
	return nil
}

func (c *Client) Send(msg string) {
	buf, _ := parser.Pack2(c.uid, cmd.SendMsg, &proto.Send{
		Uid:  int32(c.uid),
		Room: c.roomid,
		Data: msg,
	})
	c.conn.Write(buf)
}

// 连接关闭的回调
func (c *Client) Close(conn *network.Conn) {
	fmt.Println("Close")
}

// 连接成功的回调(Client)
func (c *Client) OnConnect(conn *network.Conn) {
	data, err := parser.Pack2(c.uid, cmd.ReqRoomList, &proto.Empty{})
	if err != nil {
		fmt.Println(err)
		return
	}
	c.conn = conn
	conn.Write(data)
}

// 新连接的回调(Server)
func (c *Client) OnAccept(conn *network.Conn) {

}

// 心跳
func (c *Client) Tick() {

}

func main() {
	uid := 0
	flag.IntVar(&uid, "uid", 0, "-uid 1")
	flag.Parse()

	client := &Client{uid: uint32(uid)}
	sconf := &network.ServerConfig{
		Addr: ":8080",
	}
	pconf := &network.PollConfig{
		HeartBeat: 1,
		MaxConn:   1000,
	}
	network.Client(sconf, pconf, client)
}
