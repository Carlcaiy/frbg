package network

import (
	"frbg/codec"
	"frbg/def"
	"log"
	"net"
	"time"
)

type Conn struct {
	poll       *Poll
	conn       *net.TCPConn // 连接
	Fd         int          // 文件描述符
	ActiveTime int64        // 活跃时间
	Protocol   byte         // 协议 0:tcp 1:ws
	svid       uint16       // 服务id
	uid        uint32       // 用户id
}

func (c *Conn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *Conn) Uid() uint32 {
	return c.uid
}

func (c *Conn) SetUid(uid uint32) {
	c.uid = uid
}

func (c *Conn) Svid() uint16 {
	return c.svid
}

func (c *Conn) SetSvid(svid uint16) {
	c.svid = svid
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
		if c.svid == 0 {
			c.poll.Del(c.Fd)
		} else {
			innerServerMgr.DelServe(c.svid)
		}
	}

	return err
}

func (c *Conn) Close() error {
	return c.conn.Close()
}

type IConn interface {
	Write(msg []byte) error
	Close() error
}
