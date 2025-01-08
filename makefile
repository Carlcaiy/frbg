.PHONY: game

all: redis etcd gate hall game login

redis:
	redis-server config/redis.conf

etcd:
	etcd > log/etcd.log 2>&1 &

client:
	go run ./examples/client/main.go &

gate:
	go run ./examples/gateway/main.go -p 6666 -wp 6667 -sid 1 > log/gate6666.log 2>&1 &

gate1:
	go run ./examples/gateway/main.go -p 16666 -wp 16667 -sid 2 > log/gate16666.log 2>&1 &

hall:
	go run ./examples/hall/main.go -p 6676 -sid 1 > log/hall6676.log 2>&1 &

hall1:
	go run ./examples/hall/main.go -p 16676 -sid 2 > log/hall16676.log 2>&1 &

game:
	go run ./examples/game/main.go > log/game.log 2>&1 &

login:
	go run ./examples/login/main.go > log/login.log 2>&1 &