package core

import (
	"strconv"
	"strings"
)

type ServerConfig struct {
	Addr       string // 服务地址
	WsAddr     string // websocket地址
	ServerType uint8  // 服务类型
	ServerId   uint8  // 服务ID
}

func (s *ServerConfig) IP() []byte {
	strs := strings.Split(s.Addr, ":")
	return []byte(strs[0])
}

func (s *ServerConfig) Port() int {
	strs := strings.Split(s.Addr, ":")
	port, _ := strconv.Atoi(strs[1])
	return port
}

func (s *ServerConfig) Svid() uint16 {
	return uint16(s.ServerType)*100 + uint16(s.ServerId)
}

func (s *ServerConfig) Equal(other *ServerConfig) bool {
	return s.ServerType == other.ServerType && s.ServerId == other.ServerId
}

func Svid(serverType uint8, serverId uint8) uint16 {
	return uint16(serverType)*100 + uint16(serverId)
}

func ServerType(svid uint16) uint8 {
	return uint8(svid / 100)
}
func ServerId(svid uint16) uint8 {
	return uint8(svid % 100)
}
