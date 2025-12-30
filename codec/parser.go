package codec

import (
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"google.golang.org/protobuf/proto"
)

// 协议相关常量
const (
	// 最大消息长度 16MB
	MaxMessageSize = 65535

	// 默认心跳间隔
	DefaultHeartBeatInterval = 30 * time.Second
)

// Parser 消息解析器
var seq uint32

// WsRead WebSocket消息读取
func WsRead(r io.ReadWriter) (*Message, error) {
	all, opCode, err := wsutil.ReadData(r, ws.StateClientSide)
	if err != nil {
		return nil, fmt.Errorf("websocket read error: %w", err)
	}

	if opCode == ws.OpClose {
		return nil, fmt.Errorf("websocket connection closed")
	}

	if len(all) < HeaderLen {
		return nil, fmt.Errorf("message too short: %d < %d", len(all), HeaderLen)
	}

	msg := AcquireMessage()

	// 解析头部
	msg.MagicNumber = all[OffsetMagic]
	if msg.MagicNumber != magicNumber {
		ReleaseMessage(msg)
		return nil, fmt.Errorf("invalid magic number: %02X", msg.MagicNumber)
	}

	msg.Version = all[OffsetVersion]
	msg.Flags = all[OffsetFlags]
	msg.Seq = byteOrder.Uint16(all[OffsetSeq:])
	msg.Timestamp = byteOrder.Uint32(all[OffsetTimestamp:])
	msg.Len = byteOrder.Uint16(all[OffsetLen:])
	// 检查消息长度
	if msg.Len > MaxMessageSize {
		ReleaseMessage(msg)
		return nil, fmt.Errorf("message too large: %d > %d", msg.Len, MaxMessageSize)
	}

	msg.CheckSum = byteOrder.Uint16(all[OffsetCheckSum:])

	// 解析命令字和负载
	if len(all) > HeaderLen {
		msg.Cmd = byteOrder.Uint16(all[HeaderLen:])
		if len(all) > HeaderLen+2 {
			msg.Payload = all[HeaderLen+2:]
		}
	}

	// 验证校验和
	expectedCheckSum := msg.calculateCheckSum()
	if msg.CheckSum != expectedCheckSum {
		ReleaseMessage(msg)
		return nil, fmt.Errorf("checksum mismatch: expected %04X, got %04X", expectedCheckSum, msg.CheckSum)
	}

	msg.All = all
	return msg, nil
}

// WsWrite WebSocket消息写入
func WsWrite(r io.Writer, msg *Message) error {
	if msg == nil {
		return fmt.Errorf("nil message")
	}
	data := msg.Pack()
	return wsutil.WriteServerBinary(r, data)
}

// TcpRead TCP消息读取
func TcpRead(r io.Reader) (*Message, error) {
	// 先读取头部
	headerBuf := make([]byte, HeaderLen)
	if _, err := io.ReadFull(r, headerBuf); err != nil {
		return nil, fmt.Errorf("read header failed: %w", err)
	}

	msg := AcquireMessage()

	// 解析头部
	msg.MagicNumber = headerBuf[OffsetMagic]
	if msg.MagicNumber != magicNumber {
		return nil, fmt.Errorf("invalid magic number: %02X", msg.MagicNumber)
	}

	msg.Version = headerBuf[OffsetVersion]
	msg.Flags = headerBuf[OffsetFlags]
	msg.Seq = byteOrder.Uint16(headerBuf[OffsetSeq:])
	msg.Timestamp = byteOrder.Uint32(headerBuf[OffsetTimestamp:])
	msg.Len = byteOrder.Uint16(headerBuf[OffsetLen:])
	// 检查消息长度
	if msg.Len > MaxMessageSize {
		ReleaseMessage(msg)
		return nil, fmt.Errorf("message too large: %d > %d", msg.Len, MaxMessageSize)
	}
	msg.CheckSum = byteOrder.Uint16(headerBuf[OffsetCheckSum:])

	// 读取命令字
	cmdBuf := make([]byte, 2)
	if _, err := io.ReadFull(r, cmdBuf); err != nil {
		ReleaseMessage(msg)
		return nil, fmt.Errorf("read cmd failed: %w", err)
	}
	msg.Cmd = byteOrder.Uint16(cmdBuf)

	// 读取负载
	if msg.Len > 0 {
		payloadBuf := make([]byte, msg.Len)
		if _, err := io.ReadFull(r, payloadBuf); err != nil {
			ReleaseMessage(msg)
			return nil, fmt.Errorf("read payload failed: %w", err)
		}
		msg.Payload = payloadBuf
	}

	// 验证校验和
	expectedCheckSum := msg.calculateCheckSum()
	if msg.CheckSum != expectedCheckSum {
		ReleaseMessage(msg)
		return nil, fmt.Errorf("checksum mismatch: expected %04X, got %04X", expectedCheckSum, msg.CheckSum)
	}

	return msg, nil
}

// TcpWrite TCP消息写入
func TcpWrite(r io.Writer, msg *Message) error {
	if msg == nil {
		return fmt.Errorf("nil message")
	}

	data := msg.Pack()
	if _, err := r.Write(data); err != nil {
		return fmt.Errorf("tcp write failed: %w", err)
	}

	return nil
}

// Pack 快速打包消息
func Pack(cmd uint16, pro proto.Message) ([]byte, error) {
	body, err := proto.Marshal(pro)
	if err != nil {
		return nil, fmt.Errorf("protobuf marshal failed: %w", err)
	}

	msg := AcquireMessage()
	defer ReleaseMessage(msg)

	msg.MagicNumber = magicNumber
	msg.Version = 1
	msg.Seq = nextSeq()
	msg.Timestamp = uint32(time.Now().Unix())
	msg.Cmd = cmd
	msg.Payload = body

	return msg.Pack(), nil
}

// PackFast 使用对象池快速打包
func PackFast(dest uint8, cmd uint16, pro proto.Message) ([]byte, error) {
	body, err := proto.Marshal(pro)
	if err != nil {
		return nil, fmt.Errorf("protobuf marshal failed: %w", err)
	}

	msg := AcquireMessage()
	msg.MagicNumber = magicNumber
	msg.Version = 1
	msg.Seq = nextSeq()
	msg.Timestamp = uint32(time.Now().Unix())
	msg.Cmd = cmd
	msg.Payload = body

	data := msg.Pack()
	ReleaseMessage(msg)

	return data, nil
}

// nextSeq 生成下一个序列号
func nextSeq() uint16 {
	return uint16(atomic.AddUint32(&seq, 1))
}
