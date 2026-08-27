BINARY  := osql
BIN_DIR := bin
DIST_DIR := dist

FILE_VERSION := $(shell cat VERSION 2>/dev/null)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo v$(FILE_VERSION))
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT)
RELEASE_LDFLAGS := -s -w $(LDFLAGS)

PLATFORMS := darwin/arm64 darwin/amd64 linux/amd64 linux/arm64

.PHONY: all build install test e2e bench vet fmt fmt-check cross dist version version-check release-check clean

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

version:
	@echo v$(FILE_VERSION)

version-check:
	@if [ -z "$(FILE_VERSION)" ]; then echo "VERSION file is missing or empty"; exit 1; fi
	@echo "$(FILE_VERSION)" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$$' || \
		{ echo "VERSION must be a bare semver number like 0.2.0, got \"$(FILE_VERSION)\""; exit 1; }
	@echo "VERSION is v$(FILE_VERSION)"

release-check: version-check
	@git fetch --tags --quiet 2>/dev/null || true
	@if git rev-parse "v$(FILE_VERSION)" >/dev/null 2>&1; then \
		echo "tag v$(FILE_VERSION) already exists — bump VERSION before tagging"; exit 1; \
	fi
	@echo "tag v$(FILE_VERSION) is free, ready to tag"

clean:
	rm -rf $(BIN_DIR) $(DIST_DIR)
