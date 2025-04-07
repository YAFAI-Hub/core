all: proto-build

build:
	mkdir -p tmp
	go build -o tmp/yafai cmd/yafai/main.go

dev:
	air -c .air.toml

run:
	./tmp/yafai 

install:
	go build -o tmp/yafai main.go
	sudo cp tmp/yafai /usr/local/bin

proto-build:
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative internal/bridge/proto/link.proto

proto-clean:
	rm -rf bridge/proto/*.pb.go

.PHONY: all build dev run install proto-build proto-clean
