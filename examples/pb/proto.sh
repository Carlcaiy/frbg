#!bin/bash

protoc --proto_path=. --go_out=. test.proto
protoc --proto_path=. --go_out=. enum.proto
protoc --go_out=. --go-grpc_out=. game.proto
