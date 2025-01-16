.PHONY: game

all: redis etcd gate hall game

redis:
	redis-server config/redis.conf

etcd:
	etcd > log/etcd.log 2>&1 &

client:
	go run ./examples/client/main.go &

gate:
	cd examples/gateway && make run

hall:
	cd examples/hall && make run

game:
	cd examples/game && make run

proto:
	cd examples/proto/ && sh proto.sh