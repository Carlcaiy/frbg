package core

import (
	"frbg/codec"
	"log"
	"time"
)

type WsConn struct {
	Conn
}

func (c *WsConn) Read() (*codec.Message, error) {
	err := c.conn.SetReadDeadline(time.Now().Add(time.Second))
	if err != nil {
		log.Printf("SetReadDeadline error: %s", err.Error())
		return nil, err
	}
	return codec.WsRead(c.conn)
}

func (c *WsConn) Write(msg *codec.Message) error {
	err := c.conn.SetWriteDeadline(time.Now().Add(time.Second))
	if err == nil {
		// 如果是用户连接，只能通过ws发送
		err = codec.WsWrite(c.conn, msg)
	}
	if err != nil {
		log.Printf("Write error: %s", err.Error())
		if c.svid == 0 {
			c.poll.Del(c.Fd)
		} else {
			serverMgr.DelServe(c.svid)
		}
	}

	return err
}

func (c *WsConn) Close() error {
	return c.conn.Close()
}
