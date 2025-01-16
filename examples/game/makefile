.PHONY: all

all: build run

build:
	go build -o game . 

run:
	- killall game
	./game > /mnt/d/golang/frbg/log/game.log 2>&1 &
	ps -ef | grep game | grep -v grep 
