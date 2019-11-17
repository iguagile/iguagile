all: build

build:
	protoc --go_out=plugins=grpc:./room room.proto

install:
	go get -u google.golang.org/grpc
	go get -u github.com/golang/protobuf/protoc-gen-go
