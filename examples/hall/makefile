.PHONY: all

all: build run

build:
	go build -o hall . 

run:
	- killall hall
	./hall -p 6676 -sid 1 > /mnt/d/golang/frbg/log/hall6676.log 2>&1 &
	ps -ef | grep hall | grep -v grep 
