.PHONY: all

all: build run

build:
	go build -o game_server . 

run:
	- killall game_server
	./game_server >> ./log/game_server.log 2>&1 &
	ps -ef | grep game_server | grep -v grep 
