ifeq ($(OS),Windows_NT)
VERSION ?= $(shell git describe --tags --always --dirty 2>NUL || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>NUL || echo none)
DATE ?= $(shell powershell -NoProfile -Command "[DateTime]::UtcNow.ToString('yyyy-MM-ddTHH:mm:ssZ')")
MKDIR_BIN = powershell -NoProfile -Command "New-Item -ItemType Directory -Force -Path 'bin' | Out-Null"
RM_BIN = powershell -NoProfile -Command "if (Test-Path 'bin') { Remove-Item -Recurse -Force 'bin' }"
else
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
MKDIR_BIN = mkdir -p bin
RM_BIN = rm -rf bin/
endif

.PHONY: all build test lint proto docker clean

all: lint test build

build:
	$(MKDIR_BIN)
	go build -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)" -o bin/viaduct ./cmd/viaduct

test:
	go test ./... -v -race

lint:
	golangci-lint run ./...

proto:
	@echo "protobuf generation not yet configured"

docker:
	@echo "docker build not yet configured"

clean:
	$(RM_BIN)
