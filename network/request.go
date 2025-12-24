package network

import (
	"fmt"
	"frbg/codec"
	"log"
	"net"
	"sync/atomic"
	"time"

	"google.golang.org/protobuf/proto"
)

var seq atomic.Uint32

type Conn struct {
	poll       *Poll
	conn       *net.TCPConn // 连接
	Fd         int          // 文件描述符
	ActiveTime int64        // 活跃时间
	isWs       bool         // 是否为websocket连接
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
	if c.isWs {
		return codec.WsRead(c.conn)
	}

	return codec.TcpRead(c.conn)
}

func (c *Conn) Write(msg *codec.Message) error {
	err := c.conn.SetWriteDeadline(time.Now().Add(time.Second))
	if err == nil {
		// 如果是用户连接，只能通过ws发送
		if c.isWs {
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

// RpcWrite 发送RPC消息并等待响应
// 参数:
//   - reqCmd: 请求命令字
//   - reqMsg: 请求消息体(protobuf)
//   - timeout: 超时时间(毫秒)
//
// 返回:
//   - *codec.Message: 响应消息
//   - error: 错误信息
func (c *Conn) RpcWrite(reqCmd uint16, reqMsg proto.Message, timeout int) (*codec.Message, error) {
	// 1. 创建请求消息
	req := codec.NewMessage(reqCmd, reqMsg)
	if req == nil {
		return nil, fmt.Errorf("create request message failed")
	}

	// 2. 生成唯一序列号用于匹配响应
	seq := uint16(seq.Add(1))
	req.Seq = seq

	// 3. 创建响应等待通道
	respChan := make(chan *codec.Message, 1)
	c.poll.RegisterRpc(seq, respChan)
	defer c.poll.UnregisterRpc(seq)

	// 4. 发送请求
	if err := c.Write(req); err != nil {
		return nil, fmt.Errorf("send request failed: %w", err)
	}

	log.Printf("RpcWrite seq:%d, req:%v p.rpcResponses:%v", seq, req, c.poll.rpcResponses)
	// 5. 等待响应或超时
	select {
	case resp := <-respChan:
		return resp, nil
	case <-time.After(time.Duration(timeout) * time.Millisecond):
		return nil, fmt.Errorf("rpc timeout after %d ms", timeout)
	}
}

// RpcWriteAsync 异步发送RPC消息
// 参数:
//   - reqCmd: 请求命令字
//   - reqMsg: 请求消息体(protobuf)
//   - callback: 响应回调函数
//
// 返回:
//   - error: 错误信息(仅发送错误)
func (c *Conn) RpcWriteAsync(reqCmd uint16, reqMsg proto.Message, callback func(*codec.Message, error)) error {
	// 1. 创建请求消息
	req := codec.NewMessage(reqCmd, reqMsg)
	if req == nil {
		return fmt.Errorf("create request message failed")
	}

	// 2. 生成唯一序列号
	seq := uint16(time.Now().UnixNano() & 0xFFFF)
	req.Seq = seq

	// 3. 注册异步回调
	c.poll.RegisterRpcCallback(seq, callback)

	// 4. 发送请求
	if err := c.Write(req); err != nil {
		c.poll.UnregisterRpc(seq)
		return fmt.Errorf("send request failed: %w", err)
	}

	return nil
}

func (c *Conn) Close() error {
	return c.conn.Close()
}

type IConn interface {
	Write(msg []byte) error
	Close() error
}
