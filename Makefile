.PHONY: tidy build run test

tidy:
	go mod tidy

build:
	go build -o xcbolt ./cmd/xcbolt

run: build
	./xcbolt

test:
	go test ./...
