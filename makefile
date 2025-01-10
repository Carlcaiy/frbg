.PHONY: game

all: redis etcd gate hall game

redis:
	redis-server config/redis.conf

etcd:
	etcd > log/etcd.log 2>&1 &

client:
	go run ./examples/client/main.go &

gate:
	go build -o gateway ./examples/gateway/main.go
	- killall gateway
	./gateway -p 6666 -wp 6667 -sid 1 > log/gateway6666.log 2>&1 &
	ps -ef | grep ./gateway | grep -v grep 

gate1:
	go build -o gateway ./examples/gateway/main.go
	- killall gateway
	./gateway -p 16666 -wp 16667 -sid 1 > log/gateway6676.log 2>&1 &
	ps -ef | grep ./gateway | grep -v grep 

hall:
	go build -o hall ./examples/hall/main.go
	- killall hall
	./hall -p 6676 -sid 1 > log/hall6676.log 2>&1 &
	ps -ef | grep ./hall | grep -v grep 

hall1:
	go run ./examples/hall/main.go -p 16676 -sid 2 > log/hall16676.log 2>&1 &

game:
	go build -o game ./examples/game/main.go
	- killall game
	./game > log/game.log 2>&1 &
	ps -ef | grep ./game | grep -v grep 

login:
	go build -o login ./examples/login/main.go
	killall login
	./login > log/login.log 2>&1 &
	ps -ef | grep ./login | grep -v grep

proto:
	cd examples/proto/ && sh proto.sh