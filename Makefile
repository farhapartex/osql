BINARY  := osql
BIN_DIR := bin

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT)

.PHONY: all build install test bench vet fmt clean

all: vet test build

build:
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) .

install:
	go install -ldflags "$(LDFLAGS)" .

test:
	go test ./... -race

bench:
	go test -bench=. -benchmem ./...

vet:
	go vet ./...

fmt:
	go fmt ./...

clean:
	rm -rf $(BIN_DIR)
