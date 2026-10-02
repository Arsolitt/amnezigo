BIN_DIR ?= bin
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
LDFLAGS ?= -s -w -X github.com/Arsolitt/amnezigo/internal/buildinfo.Version=$(VERSION) -X github.com/Arsolitt/amnezigo/internal/buildinfo.Commit=$(COMMIT)
export CGO_ENABLED ?= 0

.DEFAULT_GOAL := build
.PHONY: build test test-race test-e2e lint fmt notice snapshot image clean

build:
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/amnezigo ./cmd/amnezigo

test:
	go test ./...

test-race:
	go test -race ./...

test-e2e:
	go test -tags=e2e ./e2e/...

lint:
	golangci-lint run

fmt:
	golangci-lint fmt

notice:
	go run ./hack/noticegen

snapshot:
	goreleaser release --snapshot --clean

# The runtime base (amneziavpn/amneziawg-go) is amd64-only, matching the
# release matrix; the explicit platform keeps the build working on arm64 hosts
# (emulated through Rosetta/containerd).
image:
	docker build --platform linux/amd64 -t amnezigo .

clean:
	rm -rf $(BIN_DIR) dist
