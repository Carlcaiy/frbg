.PHONY: game

all: gate hall game

data: redis etcd

redis:
	redis-server config/redis.conf &

etcd:
	etcd > log/etcd.log 2>&1 &

client:
	go run ./examples/client/main.go &

gate:
	cd examples/gate && make all

hall:
	cd examples/hall && make all

game:
	cd examples/game && make all

proto:
	cd examples/proto/ && sh proto.sh