package codec

import (
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
)

const (
	// 消息头长度：16字节
	HeaderLen = 15

	// 魔术字
	magicNumber = 0x12

	// 标志位定义
	FlagsCompress  = 0x1 // 压缩标志位
	FlagsEncrypt   = 0x2 // 加密标志位
	FlagsHeartBeat = 0x4 // 心跳标志位
)

// 消息头字段偏移常量
const (
	OffsetMagic     = 0
	OffsetVersion   = 1
	OffsetFlags     = 2
	OffsetDestType  = 3
	OffsetDestId    = 4
	OffsetSeq       = 5
	OffsetTimestamp = 7
	OffsetLen       = 11
	OffsetCheckSum  = 13
)

var byteOrder binary.ByteOrder = binary.BigEndian

// Message 消息结构体
type Message struct {
	Header
	Cmd     uint16 // 命令字
	Payload []byte // 消息体
	All     []byte // 完整消息（包含头）
}

// Header 消息头
type Header struct {
	MagicNumber uint8  // 魔术字 - 放在最前面快速识别
	Version     uint8  // 版本号
	Flags       uint8  // 标志位：压缩、加密、心跳等
	DestType    uint8  // 目标服务类型
	DestId      uint8  // 目标服务ID
	Seq         uint16 // 消息序列号
	Timestamp   uint32 // 时间戳
	Len         uint16 // 消息长度
	CheckSum    uint16 // 校验码 - 放在最后
}

// 对象池相关
var messagePool = sync.Pool{
	New: func() interface{} {
		return &Message{}
	},
}

// AcquireMessage 从对象池获取消息
func AcquireMessage() *Message {
	msg := messagePool.Get().(*Message)
	msg.Reset()
	return msg
}

// ReleaseMessage 释放消息回对象池
func ReleaseMessage(msg *Message) {
	if msg != nil {
		msg.Reset()
		messagePool.Put(msg)
	}
}

// Reset 重置消息
func (m *Message) Reset() {
	m.Header = Header{}
	m.Cmd = 0
	m.Payload = m.Payload[:0]
	m.All = m.All[:0]
}

// NewMessage 创建新消息
func NewMessage(serverType uint8, serveID uint8, cmd uint16, pro proto.Message) *Message {
	msg := AcquireMessage()
	msg.DestType = serverType
	msg.DestId = serveID
	msg.MagicNumber = magicNumber
	msg.Version = 1
	msg.Seq = 0
	msg.Timestamp = uint32(time.Now().Unix())
	msg.Cmd = cmd

	if pro != nil {
		body, err := proto.Marshal(pro)
		if err != nil {
			ReleaseMessage(msg)
			return nil
		}
		msg.Payload = body
	}
	// 更新长度和校验和
	msg.Len = uint16(len(msg.Payload))
	msg.CheckSum = msg.calculateCheckSum()

	return msg
}

// String 返回消息字符串表示
func (m *Message) String() string {
	if m == nil {
		return "nil"
	}
	return fmt.Sprintf("Magic:%02X Ver:%d Flags:%02X Dest:%d/%d Seq:%d Cmd:%d Len:%d TS:%d CheckSum:%04X",
		m.MagicNumber, m.Version, m.Flags, m.DestType, m.DestId, m.Seq, m.Cmd, m.Len, m.Timestamp, m.CheckSum)
}

// Pack 快速打包（复用缓冲区）
func (m *Message) Pack() []byte {
	if m.All != nil && cap(m.All) >= HeaderLen+2+len(m.Payload) {
		m.All = m.All[:HeaderLen+2+len(m.Payload)]
	} else {
		m.All = make([]byte, HeaderLen+2+len(m.Payload))
	}

	// 更新元数据
	m.Len = uint16(len(m.Payload))
	m.CheckSum = m.calculateCheckSum()

	// 打包数据
	m.All[OffsetMagic] = m.MagicNumber
	m.All[OffsetVersion] = m.Version
	m.All[OffsetFlags] = m.Flags
	m.All[OffsetDestType] = m.DestType
	m.All[OffsetDestId] = m.DestId
	byteOrder.PutUint16(m.All[OffsetSeq:], m.Seq)
	byteOrder.PutUint32(m.All[OffsetTimestamp:], m.Timestamp)
	byteOrder.PutUint16(m.All[OffsetLen:], m.Len)
	byteOrder.PutUint16(m.All[OffsetCheckSum:], m.CheckSum)
	byteOrder.PutUint16(m.All[HeaderLen:], m.Cmd)
	copy(m.All[HeaderLen+2:], m.Payload)

	return m.All
}

// PackWith 使用新的命令字和protobuf消息打包
func (m *Message) PackWith(cmd uint16, pro proto.Message) ([]byte, error) {
	body, err := proto.Marshal(pro)
	if err != nil {
		return nil, err
	}
	m.Cmd = cmd
	m.Payload = body
	return m.Pack(), nil
}

// Unpack 解包protobuf消息
func (m *Message) Unpack(pro proto.Message) error {
	if len(m.Payload) == 0 {
		return nil
	}
	return proto.Unmarshal(m.Payload, pro)
}

// Bytes 返回完整消息字节
func (m *Message) Bytes() []byte {
	return m.All
}

// calculateCheckSum 计算校验和
func (m *Message) calculateCheckSum() uint16 {
	var sum uint32

	// 计算头部校验和（除了校验和字段本身）
	sum += uint32(m.MagicNumber)
	sum += uint32(m.Version)
	sum += uint32(m.Flags)
	sum += uint32(m.DestType)
	sum += uint32(m.DestId)
	sum += uint32(m.Seq)
	sum += uint32(m.Timestamp >> 16)
	sum += uint32(m.Timestamp & 0xFFFF)
	sum += uint32(m.Len)
	sum += uint32(m.Cmd)

	// 计算负载校验和
	for i := 0; i < len(m.Payload)-1; i += 2 {
		sum += uint32(m.Payload[i])<<8 + uint32(m.Payload[i+1])
	}
	if len(m.Payload)%2 == 1 {
		sum += uint32(m.Payload[len(m.Payload)-1]) << 8
	}

	// 回卷求和
	for (sum >> 16) > 0 {
		sum = (sum & 0xFFFF) + (sum >> 16)
	}

	return uint16(^sum)
}

// IsHeartBeat 检查是否为心跳消息
func (m *Message) IsHeartBeat() bool {
	return m.Flags&FlagsHeartBeat != 0
}

// IsCompressed 检查是否压缩
func (m *Message) IsCompressed() bool {
	return m.Flags&FlagsCompress != 0
}

// IsEncrypted 检查是否加密
func (m *Message) IsEncrypted() bool {
	return m.Flags&FlagsEncrypt != 0
}

// SetFlags 设置消息标志位
func (m *Message) SetFlags(flags ...byte) {
	for _, f := range flags {
		m.Flags |= f
	}
}
