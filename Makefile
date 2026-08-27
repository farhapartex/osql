BINARY  := osql
BIN_DIR := bin
DIST_DIR := dist

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT)
RELEASE_LDFLAGS := -s -w $(LDFLAGS)

PLATFORMS := darwin/arm64 darwin/amd64 linux/amd64 linux/arm64

.PHONY: all build install test e2e bench vet fmt fmt-check cross dist clean

all: fmt-check vet test build e2e

build:
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) .

install:
	go install -ldflags "$(LDFLAGS)" .

test:
	go test ./... -race

e2e: build
	bash scripts/e2e.sh

bench:
	go test -bench=. -benchmem ./...

vet:
	go vet ./...

fmt:
	go fmt ./...

fmt-check:
	@unformatted="$$(gofmt -l .)"; \
	if [ -n "$$unformatted" ]; then \
		echo "these files need gofmt:"; echo "$$unformatted"; exit 1; \
	fi

cross:
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; arch=$${platform#*/}; \
		echo "building $$os/$$arch"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch \
			go build -trimpath -ldflags "$(RELEASE_LDFLAGS)" -o /dev/null . || exit 1; \
	done

dist: clean
	@mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; arch=$${platform#*/}; \
		stage="$(DIST_DIR)/$(BINARY)_$(VERSION)_$${os}_$${arch}"; \
		mkdir -p "$$stage"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch \
			go build -trimpath -ldflags "$(RELEASE_LDFLAGS)" -o "$$stage/$(BINARY)" . || exit 1; \
		cp LICENSE README.md "$$stage/"; \
		tar -czf "$$stage.tar.gz" -C $(DIST_DIR) "$(BINARY)_$(VERSION)_$${os}_$${arch}"; \
		rm -rf "$$stage"; \
		echo "packaged $$stage.tar.gz"; \
	done
	@cd $(DIST_DIR) && shasum -a 256 *.tar.gz > checksums.txt 2>/dev/null || \
		(cd $(DIST_DIR) && sha256sum *.tar.gz > checksums.txt)
	@echo "checksums written to $(DIST_DIR)/checksums.txt"

clean:
	rm -rf $(BIN_DIR) $(DIST_DIR)
