.PHONY: all

all: build run

build:
	go build -o gateway . 

run:
	- killall gateway
	./gateway -p 6666 -wp 6667 -sid 1 > /mnt/d/golang/frbg/log/gateway6666.log 2>&1 &
	ps -ef | grep gateway | grep -v grep 
