.PHONY: all build cli server install run-server

all: build

build: cli server

cli:
	go build -o bin/kitin ./cmd/dig

server:
	go build -o bin/kitin-server ./cmd/server

install: cli
	@mkdir -p $(HOME)/go/bin
	install -m 755 bin/kitin $(HOME)/go/bin/kitin
	@echo "installed $(HOME)/go/bin/kitin — ensure ~/go/bin is on your PATH"

run-server:
	go run ./cmd/server
