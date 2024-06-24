package test

import (
	"fmt"
	"frbg/def"
	"frbg/network"
	"testing"
)

func TestEtcd(t *testing.T) {
	cli := network.NewEtcd(&network.ServerConfig{
		Addr:       "127.0.0.1:8888",
		ServerType: def.ST_Game,
		ServerId:   1,
		Subs:       []def.ServerType{def.ST_Broadcast},
	}, func(i interface{}) {
		fmt.Println(i)
	})
	cli.Init()
	// cli.Close()
}
