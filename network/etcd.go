package network

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/namespace"
)

var etcd *Etcd

type Etcd struct {
	cli         *clientv3.Client
	conf        *ServerConfig
	serverConfs map[int32]string
	handle      Handler
}

func NewEtcd(s *ServerConfig, handle Handler) *Etcd {
	ret := new(Etcd)
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:         []string{"127.0.0.1:2379"},
		AutoSyncInterval:  time.Second,
		DialTimeout:       time.Second * 3,
		DialKeepAliveTime: time.Second * 5,
		// Username:          "cyf",
		// Password:          "cyf123",
	})
	if err != nil {
		panic(err)
	}
	cli.KV = namespace.NewKV(cli.KV, "cyf/")
	cli.Watcher = namespace.NewWatcher(cli.Watcher, "cyf/")
	cli.Lease = namespace.NewLease(cli.Lease, "cyf/")
	ret.cli = cli
	ret.conf = s
	ret.serverConfs = make(map[int32]string)
	return ret
}

func (p *Etcd) Init() {
	p.Put()
	// 如果有订阅需求的才去获取，正常只需要上报就行
	p.Get()
	go p.Watch()
}

func (p *Etcd) key() string {
	return fmt.Sprintf("server/%d/%d", p.conf.ServerType, p.conf.ServerId)
}

func (p *Etcd) parseKey(key string) (uint8, uint32) {
	strs := strings.Split(key, "/")
	if len(strs) != 3 {
		log.Println("key struct wrong", key)
		return 0, 0
	}
	if strs[0] != "server" {
		log.Println("without server prefix", strs[0])
		return 0, 0
	}
	serverType, err := strconv.Atoi(strs[1])
	if err != nil {
		log.Println("parse server type error", strs[1])
		return 0, 0
	}
	serverID, err := strconv.Atoi(strs[2])
	if err != nil {
		log.Println("parse server id error", strs[2])
		return 0, 0
	}

	return uint8(serverType), uint32(serverID)
}

func (p *Etcd) parseValue(value []byte) string {
	addr := string(value)
	return addr
}

func (p *Etcd) Put() {
	p.cli.Put(context.TODO(), p.key(), p.conf.Addr)
}

func (p *Etcd) Get() {
	res, err := p.cli.Get(context.TODO(), "server/", clientv3.WithPrefix())
	if err != nil {
		return
	}
	for _, kv := range res.Kvs {
		serverType, serverId := p.parseKey(string(kv.Key))
		addr := p.parseValue(kv.Value)
		p.handle.OnEtcd(&ServerConfig{
			ServerType: serverType,
			ServerId:   serverId,
			Addr:       addr,
		})
	}
}

func (p *Etcd) Del() {
	p.cli.Delete(context.TODO(), p.key())
}

func (p *Etcd) Watch() {
	wg.Add(1)
	defer func() {
		wg.Done()
	}()
	watchCh := p.cli.Watch(context.TODO(), "server", clientv3.WithPrefix())
	for {
		select {
		// case <-closech:
		// 	return
		case <-p.cli.Ctx().Done():
			log.Println("ectd done")
			return
		case watch := <-watchCh:
			for _, event := range watch.Events {
				switch event.Type {
				case clientv3.EventTypePut:
					serverType, serverId := p.parseKey(string(event.Kv.Key))
					addr := p.parseValue(event.Kv.Value)
					log.Printf("etcd event put %d:%d:%s", serverType, serverId, addr)
					p.handle.OnEtcd(&ServerConfig{
						ServerType: serverType,
						ServerId:   serverId,
						Addr:       addr,
					})
				case clientv3.EventTypeDelete:
					serverType, serverId := p.parseKey(string(event.Kv.Key))
					p.handle.OnEtcd(&ServerConfig{
						ServerType: serverType,
						ServerId:   serverId,
					})
					log.Printf("etcd event delete %d:%d", serverType, serverId)
				}
			}
		}
	}
}

func (e *Etcd) Close() {
	e.Del()
	e.cli.Close()
}
