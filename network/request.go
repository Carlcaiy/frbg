package network

import (
	"frbg/codec"
	"log"
	"net"
	"time"

	"github.com/gogo/protobuf/proto"
)

type Conn struct {
	poll       *Poll
	conf       *ServerConfig // 信息
	conn       *net.TCPConn  // 连接
	Fd         int           // 文件描述符
	ActiveTime int64         // 活跃时间
	Uid        uint32        // 玩家
	payload    []byte        // 数据包
}

func (c *Conn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *Conn) Read() (*codec.Message, error) {
	err := c.conn.SetReadDeadline(time.Now().Add(time.Second))
	if err != nil {
		log.Printf("SetReadDeadline error: %s", err.Error())
		c.poll.Del(c.Fd)
		return nil, err
	}
	if isWebSocket {
		return codec.WsRead(c.conn)
	}

	return codec.TcpRead(c.conn)
}

func (c *Conn) Write(msg []byte) error {
	err := c.conn.SetWriteDeadline(time.Now().Add(time.Second))
	if err == nil {
		_, err = c.conn.Write(msg)
	}
	if err != nil {
		log.Printf("Write error: %s", err.Error())
		c.poll.Del(c.Fd)
	}

	return err
}

func (c *Conn) Send(uid uint32, dest uint8, cmd uint16, pro proto.Message) error {
	body, err := proto.Marshal(pro)
	if err != nil {
		return err
	}
	msg := &codec.Message{
		ServeType: dest,
		Cmd:       cmd,
		Body:      body,
	}

	return c.Write(msg.Pack())
}

func (c *Conn) Close() error {
	return c.conn.Close()
}

type IConn interface {
	Write(msg []byte) error
	Close() error
}

func NewMessage(poll *Poll, conn *Conn, msg *codec.Message) *Message {
	return &Message{
		poll:    poll,
		conn:    conn,
		Message: msg,
	}
}

type Message struct {
	poll *Poll
	conn *Conn
	*codec.Message
}

func (r *Message) GetClient() *Conn {
	return r.conn
}

func (r *Message) Response(dest uint8, cmd uint16, pro proto.Message) error {
	body, err := proto.Marshal(pro)
	if err != nil {
		return err
	}
	msg := &codec.Message{
		ServeType: dest,
		Cmd:       cmd,
		Body:      body,
	}
	return r.conn.Write(msg.Pack())
}
