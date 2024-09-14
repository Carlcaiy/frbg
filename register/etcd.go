package register

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	etcd "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/namespace"
)

var client *etcd.Client
var serverMap map[string]string

func init_client() {
	if client != nil {
		return
	}
	cli, err := etcd.New(etcd.Config{
		Endpoints:            []string{"127.0.0.1:2379"},
		AutoSyncInterval:     time.Second,
		DialTimeout:          time.Second * 3,
		DialKeepAliveTime:    time.Second * 5,
		DialKeepAliveTimeout: time.Hour,
		// Username:          "cyf",
		// Password:          "cyf123",
	})
	if err != nil {
		log.Fatalf("init etcd error:%s", err.Error())
		return
	}
	cli.KV = namespace.NewKV(cli.KV, "cyf/")
	cli.Watcher = namespace.NewWatcher(cli.Watcher, "cyf/")
	cli.Lease = namespace.NewLease(cli.Lease, "cyf/")
	client = cli
	serverMap = make(map[string]string)
	get_configs()
	go watch()
}

func Get(serverType uint8, serverId uint8) string {
	init_client()
	if client == nil {
		return ""
	}
	key := fmt.Sprintf("server/%d/%d", serverType, serverId)
	if addr, ok := serverMap[key]; ok {
		return addr
	}
	return ""
}

func Put(serverType uint8, serverId uint8, addr string) error {
	init_client()
	if client == nil {
		return fmt.Errorf("etcd init failed")
	}
	key := fmt.Sprintf("server/%d/%d", serverType, serverId)
	_, err := client.Put(context.TODO(), key, addr)
	return err
}

func parseKey(key string) (uint8, uint8) {
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

	return uint8(serverType), uint8(serverID)
}

func get_configs() {
	if client == nil {
		return
	}
	res, err := client.Get(context.TODO(), "server/", etcd.WithPrefix())
	if err != nil {
		return
	}
	for _, kv := range res.Kvs {
		serverMap[string(kv.Key)] = string(kv.Value)
		log.Printf("get_configs:%s:%s", string(kv.Key), string(kv.Value))
	}
}

func Del(serverType uint8, serverId uint8) {
	if client != nil {
		key := fmt.Sprintf("server/%d/%d", serverType, serverId)
		client.Delete(context.TODO(), key)
	}
}

func watch() {
	log.Println("start etcd watch coroutine")
	watchCh := client.Watch(context.TODO(), "server", etcd.WithPrefix())
	for watch := range watchCh {
		for _, event := range watch.Events {
			switch event.Type {
			case etcd.EventTypePut:
				key := string(event.Kv.Key)
				addr := string(event.Kv.Value)
				serverMap[key] = addr
				log.Printf("etcd event put %s:%s", key, addr)
			case etcd.EventTypeDelete:
				key := string(event.Kv.Key)
				addr := serverMap[key]
				delete(serverMap, key)
				log.Printf("etcd event delete %s:%s", key, addr)
			}
		}
	}
}
