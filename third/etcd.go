package third

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
var serverMap map[uint16]string

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
	serverMap = make(map[uint16]string)
	get_configs()
	go watch()
}

func Get(svid uint16) string {
	init_client()
	if client == nil {
		return ""
	}
	if addr, ok := serverMap[svid]; ok {
		return addr
	}
	return ""
}

func Gets(serverType uint8) []uint8 {
	init_client()
	if client == nil {
		return nil
	}
	var listId []uint8
	for svid := range serverMap {
		st := uint8(svid / 100)
		if st == serverType {
			listId = append(listId, uint8(svid-(svid/100)*100))
		}
	}
	return listId
}

func Put(svid uint16, addr string) error {
	init_client()
	if client == nil {
		return fmt.Errorf("etcd init failed")
	}
	key := fmt.Sprintf("server/%d", svid)
	_, err := client.Put(context.TODO(), key, addr)
	return err
}

func parseKey(key []byte) uint16 {
	strs := strings.Split(string(key), "/")
	if len(strs) != 2 {
		log.Println("key struct wrong", string(key))
		client.Delete(context.TODO(), string(key))
		return 0
	}
	if strs[0] != "server" {
		log.Println("without server prefix", strs[0])
		return 0
	}
	svid, err := strconv.Atoi(strs[1])
	if err != nil {
		log.Println("parse server type error", strs[1])
		return 0
	}
	return uint16(svid)
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
		if key := parseKey(kv.Key); key > 0 {
			serverMap[key] = string(kv.Value)
			log.Printf("get_configs:%d:%s", key, string(kv.Value))
		}
	}
}

func Del(svid uint16) error {
	if client != nil {
		key := fmt.Sprintf("server/%d", svid)
		if _, err := client.Delete(context.TODO(), key); err != nil {
			return err
		}
	}
	return nil
}

func watch() {
	log.Println("start etcd watch coroutine")
	watchCh := client.Watch(context.TODO(), "server", etcd.WithPrefix())
	for watch := range watchCh {
		for _, event := range watch.Events {
			switch event.Type {
			case etcd.EventTypePut:
				key := parseKey(event.Kv.Key)
				addr := string(event.Kv.Value)
				serverMap[key] = addr
				log.Printf("etcd event put %d:%s", key, addr)
			case etcd.EventTypeDelete:
				key := parseKey(event.Kv.Key)
				addr := serverMap[key]
				delete(serverMap, key)
				log.Printf("etcd event delete %d:%s", key, addr)
			}
		}
	}
}
