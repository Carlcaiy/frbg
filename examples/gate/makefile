.PHONY: all

all: build run

build:
	go build -o gate_server . 

run:
	- killall gate_server
	./gate_server -p 6666 -sid 1 > ./log/gate_server6666.log 2>&1 &
	ps -ef | grep gate_server | grep -v grep 
