package network

import (
	"strconv"
	"strings"
)

type ServerConfig struct {
	Addr       string // 服务地址
	ServerType uint8  // 服务类型
	ServerId   uint32 // 服务ID
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
