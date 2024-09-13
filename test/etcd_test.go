package test

import (
	"frbg/def"
	"frbg/network"
	"testing"
)

func TestEtcd(t *testing.T) {
	cli := network.NewEtcd(&network.ServerConfig{
		Addr:       "127.0.0.1:8888",
		ServerType: def.ST_Game,
		ServerId:   1,
	})
	cli.Init()
	// cli.Close()
}
