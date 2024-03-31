#!/bin/bash

client:
	go run ./client/main.go

server:
	go run ./server/main.go

proto:
	protoc --proto_path=./proto --go_out=./proto proto/chat.proto