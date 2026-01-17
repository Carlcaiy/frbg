package core

import (
	"fmt"
	"frbg/codec"
	"log"
	"net"
	"sync/atomic"
	"time"

	"google.golang.org/protobuf/proto"
)

type IConn interface {
	Fd() int
	Svid() uint16
	SetSvid(svid uint16)
	ActiveTime() int64
	SetActiveTime(t int64)
	Context() interface{}
	SetContext(ctx interface{})
	String() string
	Read() (*codec.Message, error)
	Write(msg *codec.Message) error
	WriteBy(cmd uint16, pro proto.Message) error
	RpcWrite(cmd uint16, req proto.Message, rsp proto.Message, timeout int) error
	RpcWriteAsync(cmd uint16, req proto.Message, callback func(*codec.Message, error)) error
	Close() error
}

var seq atomic.Uint32

type Conn struct {
	poll       *Poll
	conn       net.Conn    // 连接
	fd         int         // 文件描述符
	activeTime int64       // 活跃时间
	svid       uint16      // 服务id
	ctx        interface{} // user-defined context
}

func (c *Conn) Fd() int {
	return c.fd
}

func (c *Conn) Svid() uint16 {
	return c.svid
}

func (c *Conn) SetSvid(svid uint16) {
	c.svid = svid
}

func (c *Conn) ActiveTime() int64 {
	return c.activeTime
}

func (c *Conn) SetActiveTime(t int64) {
	// log.Printf("SetActiveTime fd:%d t:%d raddr:%s", c.fd, t, c.String())
	c.activeTime = t
}

func (c *Conn) Context() interface{} {
	return c.ctx
}

func (c *Conn) SetContext(ctx interface{}) {
	log.Printf("SetContext ctx:%v", ctx)
	c.ctx = ctx
}

func (c *Conn) Close() error {
	return c.conn.Close()
}

func (c *Conn) String() string {
	return c.conn.RemoteAddr().String()
}

func (c *Conn) Read() (*codec.Message, error) {
	now := time.Now()
	c.conn.SetReadDeadline(now.Add(time.Second))
	msg, err := codec.TcpRead(c.conn)
	if err != nil {
		log.Printf("Read error: %s", err.Error())
		return nil, err
	}
	// 更新活跃时间
	c.SetActiveTime(now.Unix())
	return msg, nil
}

// 复用Message
func (c *Conn) Write(msg *codec.Message) error {
	now := time.Now()
	err := c.conn.SetWriteDeadline(now.Add(time.Second))
	if err == nil {
		err = codec.TcpWrite(c.conn, msg)
	}
	if err != nil {
		log.Printf("Write error: %s", err.Error())
		c.poll.Del(c.Fd())
		return err
	}

	// 更新活跃时间
	c.SetActiveTime(now.Unix())
	return err
}

func (c *Conn) WriteBy(cmd uint16, pro proto.Message) error {
	payload, err := proto.Marshal(pro)
	if err != nil {
		return fmt.Errorf("marshal proto message failed: %w", err)
	}
	msg := codec.AcquireMessage()
	msg.Cmd = cmd
	msg.Payload = payload
	return c.Write(msg)
}

// RpcWrite 发送RPC消息并等待响应
// 参数:
//   - cmd: 请求命令字
//   - req: 请求消息体(protobuf)
//   - rsp: 返回消息体(protobuf)
//   - timeout: 超时时间(毫秒)
func (c *Conn) RpcWrite(cmd uint16, req proto.Message, rsp proto.Message, timeout int) error {
	// 1. 序列化请求消息
	payload, err := proto.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal proto message failed: %w", err)
	}

	// 2. 生成唯一序列号用于匹配响应
	seq := uint16(seq.Add(1))
	msg := codec.AcquireMessage()
	msg.Cmd = cmd
	msg.Payload = payload
	msg.Seq = seq

	// 3. 创建响应等待通道
	respChan := make(chan *codec.Message, 1)
	rpcMgr.RegisterRpc(seq, respChan)
	defer rpcMgr.UnregisterRpc(seq)

	// 4. 发送请求
	if err := c.Write(msg); err != nil {
		return fmt.Errorf("send request failed: %w", err)
	}
	log.Printf("RpcWrite seq:%d, req:%v p.rpcResponses:%v", seq, req, rpcMgr.rpcResponses)

	// 5. 等待响应或超时
	select {
	case msg := <-respChan:
		return msg.Unpack(rsp)
	case <-time.After(time.Duration(timeout) * time.Millisecond):
		return fmt.Errorf("rpc timeout after %d ms", timeout)
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
func (c *Conn) RpcWriteAsync(cmd uint16, req proto.Message, callback func(*codec.Message, error)) error {
	// 1. 序列化请求消息
	payload, err := proto.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal proto message failed: %w", err)
	}

	// 2. 生成唯一序列号用于匹配响应
	seq := uint16(seq.Add(1))
	msg := codec.AcquireMessage()
	msg.Cmd = cmd
	msg.Payload = payload
	msg.Seq = seq

	// 3. 注册异步回调
	rpcMgr.RegisterRpcCallback(seq, callback)

	// 4. 发送请求
	if err := c.Write(msg); err != nil {
		return fmt.Errorf("send request failed: %w", err)
	}
	return nil
}

type WsConn struct {
	Conn
}

func (c *WsConn) Read() (*codec.Message, error) {
	now := time.Now()
	c.conn.SetReadDeadline(now.Add(time.Second))
	if msg, err := codec.WsRead(c.conn); err == nil {
		c.SetActiveTime(now.Unix())
		return msg, nil
	} else {
		log.Printf("WsRead error: %s", err.Error())
		return nil, err
	}
}

func (c *WsConn) Write(msg *codec.Message) error {
	// if !msg.IsHeartBeat() {
	// 	log.Printf("send ws msg:%s", msg.String())
	// }
	now := time.Now()
	err := c.conn.SetWriteDeadline(now.Add(time.Second))
	if err == nil {
		// 如果是用户连接，只能通过ws发送
		err = codec.WsWrite(c.conn, msg)
	}
	if err != nil {
		c.poll.Del(c.Fd())
		log.Printf("Write error: %s", err.Error())
		return err
	}
	c.SetActiveTime(now.Unix())
	return err
}
