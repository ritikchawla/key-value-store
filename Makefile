.PHONY: proto build run

proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/raft.proto

build: proto
	go build -o bin/kvstore ./cmd/kvstore

run: build
	./bin/kvstore