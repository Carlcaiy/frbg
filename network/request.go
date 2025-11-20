package network

import (
	"frbg/codec"
	"frbg/def"
	"log"
	"net"
	"time"

	"google.golang.org/protobuf/proto"
)

type Conn struct {
	poll       *Poll
	conf       *ServerConfig // 信息
	conn       *net.TCPConn  // 连接
	Fd         int           // 文件描述符
	ActiveTime int64         // 活跃时间
	Uid        uint32        // 玩家
	Protocol   byte          // 协议 0:tcp 1:ws
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
	if c.Protocol == def.ProtocolWs {
		return codec.WsRead(c.conn)
	}

	return codec.TcpRead(c.conn)
}

func (c *Conn) Write(msg *codec.Message) error {
	err := c.conn.SetWriteDeadline(time.Now().Add(time.Second))
	if err == nil {
		// 如果是用户连接，只能通过ws发送
		if c.Protocol == def.ProtocolWs {
			err = codec.WsWrite(c.conn, msg)
		} else {
			err = codec.TcpWrite(c.conn, msg)
		}
	}
	if err != nil {
		log.Printf("Write error: %s", err.Error())
		err = c.poll.Del(c.Fd)
	}

	return err
}

func (c *Conn) Send(dest uint8, cmd uint16, pro proto.Message) error {
	msg := codec.NewMessage(dest, 0, cmd, pro)
	return c.Write(msg)
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

// 原路返回
func (r *Message) Response(cmd uint16, pro proto.Message) error {
	body, err := proto.Marshal(pro)
	if err != nil {
		return err
	}
	msg := &codec.Message{
		Header:  r.Header,
		Cmd:     cmd,
		Payload: body,
	}
	return r.conn.Write(msg)
}
