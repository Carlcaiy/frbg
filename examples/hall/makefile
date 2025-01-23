.PHONY: all

all: build run

build:
	go build -o hall_server . 

run:
	- killall hall_server
	./hall_server -p 6676 -sid 1 > /mnt/d/golang/frbg/log/hall_server6676.log 2>&1 &
	ps -ef | grep hall_server | grep -v grep 
