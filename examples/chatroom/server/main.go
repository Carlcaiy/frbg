package main

import (
	"fmt"
	"frbg/examples/chatroom/cmd"
	"frbg/examples/chatroom/proto"
	"frbg/network"
	"frbg/parser"
)

type Server struct {
	conns          map[int32]*network.Conn
	history        map[string]uint32
	map_room_users map[int32][]int32
	roomList       []int32
}

// Handler的初始化
func (c *Server) Init() {
	c.conns = make(map[int32]*network.Conn)
	c.history = make(map[string]uint32)
	c.roomList = []int32{1, 2, 3, 4, 5, 6, 7, 8}
	c.map_room_users = make(map[int32][]int32)
}

// 消息路由
func (c *Server) Route(conn *network.Conn, msg *parser.Message) error {
	switch msg.Cmd() {
	case cmd.ReqRoomList:
		if buf, err := parser.Pack2(msg.UserID(), cmd.RespRoomList, &proto.RespRoomList{RoomIds: c.roomList}); err != nil {
			fmt.Println(err)
			return err
		} else {
			conn.Write(buf)
		}
	case cmd.ReqJoinRoom:
		data := &proto.ReqJoinRoom{}
		msg.UnPack(data)
		fmt.Println(data)
		c.conns[int32(msg.UserID())] = conn
		c.map_room_users[data.RoomId] = append(c.map_room_users[data.RoomId], int32(msg.UserID()))
		if buf, err := parser.Pack2(msg.UserID(), cmd.RespJoinRoom, &proto.RespJoinRoom{
			RoomId:  data.RoomId,
			UserIds: c.map_room_users[data.RoomId],
		}); err != nil {
			fmt.Println(err)
			return err
		} else {
			conn.Write(buf)
		}
	case cmd.SendMsg:
		data := &proto.Send{}
		if err := msg.UnPack(data); err != nil {
			return err
		}
		fmt.Println(data)
		for _, uid := range c.map_room_users[data.Room] {
			if conn, ok := c.conns[uid]; ok {
				buf, _ := parser.Pack2(uint32(uid), cmd.PushMsg, data)
				conn.Write(buf)
			}
		}
	}
	return nil
}

// 连接关闭的回调
func (c *Server) Close(conn *network.Conn) {

}

// 连接成功的回调
func (c *Server) OnConnect(conn *network.Conn) {

}

// 新连接的回调
func (c *Server) OnAccept(conn *network.Conn) {
	fmt.Println(conn.TCPConn.RemoteAddr().String())
	c.history[conn.TCPConn.RemoteAddr().String()]++
}

// 心跳
func (c *Server) Tick() {

}

func main() {
	server := &Server{}
	sconf := &network.ServerConfig{
		Addr: ":8080",
	}
	pconf := &network.PollConfig{
		HeartBeat: 1,
		MaxConn:   1000,
	}
	network.Serve(sconf, pconf, server)
}
