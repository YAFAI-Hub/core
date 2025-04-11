all: proto-build

build:
	mkdir -p tmp
	go build -o tmp/yafai main.go

dev:
	air -c .air.toml

run:
	./tmp/yafai 

install:
	go build -o tmp/yafai main.go
	sudo cp tmp/yafai /usr/local/bin

proto-gen:
	@find internal/bridge/ -name "*.proto" | while read proto_file; do \
		service_dir=$$(dirname $$proto_file | sed 's|internal/bridge/||'); \
		mkdir -p internal/bridge/$$service_dir; \
		protoc --proto_path=. \
			--go_out=internal/bridge/$$service_dir \
			--go-grpc_out=internal/bridge/$$service_dir \
			$$proto_file; \
	done





proto-rm:
	rm -rf internal/bridge/gen/


.PHONY: all build dev run install proto-build proto-clean
