ifeq ($(OS),Windows_NT)
VERSION ?= $(shell git describe --tags --always --dirty 2>NUL || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>NUL || echo none)
DATE ?= $(shell powershell -NoProfile -Command "[DateTime]::UtcNow.ToString('yyyy-MM-ddTHH:mm:ssZ')")
MKDIR_BIN = powershell -NoProfile -Command "New-Item -ItemType Directory -Force -Path 'bin' | Out-Null"
MKDIR_DIST = powershell -NoProfile -Command "New-Item -ItemType Directory -Force -Path 'dist' | Out-Null"
RM_BIN = powershell -NoProfile -Command "if (Test-Path 'bin') { Remove-Item -Recurse -Force 'bin' }"
RM_DIST = powershell -NoProfile -Command "if (Test-Path 'dist') { Remove-Item -Recurse -Force 'dist' }"
RM_COVER = powershell -NoProfile -Command "$$paths = @('coverage','coverage.out','coverage.out;','coverage-fresh','coverage-fresh.out'); foreach ($$path in $$paths) { if (Test-Path $$path) { Remove-Item -Force $$path } }"
BIN_TARGET = bin/viaduct.exe
GO_RUN = powershell -NoProfile -ExecutionPolicy Bypass -File scripts/go_run_windows.ps1
RUN_BIN = $(GO_RUN) -Ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)" ./cmd/viaduct
WEB_INSTALL = cd web && npm ci
COVERPROFILE_ARG = "-coverprofile=coverage.out"
COVERFUNC_ARG = "-func=coverage.out"
GO_BUILD_LINUX_AMD64 = powershell -NoProfile -Command "$$env:CGO_ENABLED='0'; $$env:GOOS='linux'; $$env:GOARCH='amd64'; go build -ldflags '-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)' -o '$(LINUX_AMD64_BINARY)' ./cmd/viaduct"
GO_BUILD_LINUX_ARM64 = powershell -NoProfile -Command "$$env:CGO_ENABLED='0'; $$env:GOOS='linux'; $$env:GOARCH='arm64'; go build -ldflags '-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)' -o '$(LINUX_ARM64_BINARY)' ./cmd/viaduct"
GO_BUILD_DARWIN_ARM64 = powershell -NoProfile -Command "$$env:CGO_ENABLED='0'; $$env:GOOS='darwin'; $$env:GOARCH='arm64'; go build -ldflags '-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)' -o '$(DARWIN_ARM64_BINARY)' ./cmd/viaduct"
GO_BUILD_WINDOWS = powershell -NoProfile -Command "$$env:CGO_ENABLED='0'; $$env:GOOS='windows'; $$env:GOARCH='amd64'; go build -ldflags '-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)' -o '$(WINDOWS_BINARY)' ./cmd/viaduct"
GO_TEST_RACE = powershell -NoProfile -ExecutionPolicy Bypass -File scripts/go_test_race_windows.ps1 ./... -v -race
GO_TEST_RACE_COUNT = powershell -NoProfile -ExecutionPolicy Bypass -File scripts/go_test_race_windows.ps1 ./... -v -race -count=1
GO_TEST_COVER = powershell -NoProfile -ExecutionPolicy Bypass -File scripts/go_test_windows.ps1 ./... $(COVERPROFILE_ARG)
else
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
MKDIR_BIN = mkdir -p bin
MKDIR_DIST = mkdir -p dist
RM_BIN = rm -rf bin/
RM_DIST = rm -rf dist/
RM_COVER = rm -f coverage coverage.out
BIN_TARGET = bin/viaduct
RUN_BIN = ./$(BIN_TARGET)
GO_RUN = go run
WEB_INSTALL = cd web && npm ci
COVERPROFILE_ARG = -coverprofile=coverage.out
COVERFUNC_ARG = -func=coverage.out
GO_BUILD_LINUX_AMD64 = CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)" -o $(LINUX_AMD64_BINARY) ./cmd/viaduct
GO_BUILD_LINUX_ARM64 = CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)" -o $(LINUX_ARM64_BINARY) ./cmd/viaduct
GO_BUILD_DARWIN_ARM64 = CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)" -o $(DARWIN_ARM64_BINARY) ./cmd/viaduct
GO_BUILD_WINDOWS = CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)" -o $(WINDOWS_BINARY) ./cmd/viaduct
GO_TEST_RACE = go test ./... -v -race
GO_TEST_RACE_COUNT = go test ./... -v -race -count=1
GO_TEST_COVER = go test ./... $(COVERPROFILE_ARG)
endif

COVER_MIN ?= 50.0
PLUGIN_MANIFEST ?= examples/plugin-example/plugin.json
PACKAGE_VERSION ?= $(patsubst v%,%,$(VERSION))
LINUX_AMD64_BINARY = bin/viaduct-linux-amd64
LINUX_ARM64_BINARY = bin/viaduct-linux-arm64
DARWIN_ARM64_BINARY = bin/viaduct-darwin-arm64
WINDOWS_BINARY = bin/viaduct.exe

.PHONY: all build build-linux build-linux-amd64 build-linux-arm64 build-darwin-arm64 build-windows test lint proto docker dashboard serve web-build web-verify package-release package-release-host-bundle package-release-linux package-release-linux-amd64 package-release-linux-amd64-bundle package-release-linux-arm64 package-release-linux-arm64-bundle package-release-darwin-arm64 package-release-darwin-arm64-bundle package-release-windows package-release-windows-bundle package-release-matrix certification-test soak-test plugin-check openapi-generate contract-check release-gate clean

all: lint test build

build:
	$(MKDIR_BIN)
	go build -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)" -o $(BIN_TARGET) ./cmd/viaduct

build-linux:
	$(MAKE) build-linux-amd64

build-linux-amd64:
	$(MKDIR_BIN)
	$(GO_BUILD_LINUX_AMD64)

build-linux-arm64:
	$(MKDIR_BIN)
	$(GO_BUILD_LINUX_ARM64)

build-darwin-arm64:
	$(MKDIR_BIN)
	$(GO_BUILD_DARWIN_ARM64)

build-windows:
	$(MKDIR_BIN)
	$(GO_BUILD_WINDOWS)

test:
	$(GO_TEST_RACE)

lint:
	golangci-lint run ./...

proto:
	protoc --go_out=. --go-grpc_out=. api/proto/plugin.proto

docker:
	docker build -t viaduct:$(VERSION) .

dashboard:
	$(MAKE) web-build

serve:
	$(WEB_INSTALL)
	cd web && npm run dev:full

web-build:
	$(WEB_INSTALL)
	cd web && npm run build

web-verify:
	$(WEB_INSTALL)
	cd web && npm run lint -- --max-warnings=0
	cd web && npm run format
	cd web && npm run test -- --run
	cd web && npm run build

package-release:
	$(RM_DIST)
	$(MKDIR_DIST)
	$(MAKE) web-build
	$(MAKE) package-release-host-bundle

package-release-host-bundle:
	$(MKDIR_DIST)
	$(MAKE) build
	$(GO_RUN) ./scripts/package_release -workspace . -version $(PACKAGE_VERSION) -commit $(COMMIT) -date $(DATE) -binary $(BIN_TARGET) -web-dir web/dist -output-dir dist

package-release-linux:
	$(MAKE) package-release-linux-amd64

package-release-linux-amd64:
	$(MAKE) web-build
	$(MAKE) package-release-linux-amd64-bundle

package-release-linux-amd64-bundle:
	$(MKDIR_DIST)
	$(MAKE) build-linux-amd64
	$(GO_RUN) ./scripts/package_release -workspace . -version $(PACKAGE_VERSION) -commit $(COMMIT) -date $(DATE) -binary $(LINUX_AMD64_BINARY) -web-dir web/dist -output-dir dist -bundle-goos linux -bundle-goarch amd64

package-release-linux-arm64:
	$(MAKE) web-build
	$(MAKE) package-release-linux-arm64-bundle

package-release-linux-arm64-bundle:
	$(MKDIR_DIST)
	$(MAKE) build-linux-arm64
	$(GO_RUN) ./scripts/package_release -workspace . -version $(PACKAGE_VERSION) -commit $(COMMIT) -date $(DATE) -binary $(LINUX_ARM64_BINARY) -web-dir web/dist -output-dir dist -bundle-goos linux -bundle-goarch arm64

package-release-darwin-arm64:
	$(MAKE) web-build
	$(MAKE) package-release-darwin-arm64-bundle

package-release-darwin-arm64-bundle:
	$(MKDIR_DIST)
	$(MAKE) build-darwin-arm64
	$(GO_RUN) ./scripts/package_release -workspace . -version $(PACKAGE_VERSION) -commit $(COMMIT) -date $(DATE) -binary $(DARWIN_ARM64_BINARY) -web-dir web/dist -output-dir dist -bundle-goos darwin -bundle-goarch arm64

package-release-windows:
	$(MAKE) web-build
	$(MAKE) package-release-windows-bundle

package-release-windows-bundle:
	$(MKDIR_DIST)
	$(MAKE) build-windows
	$(GO_RUN) ./scripts/package_release -workspace . -version $(PACKAGE_VERSION) -commit $(COMMIT) -date $(DATE) -binary $(WINDOWS_BINARY) -web-dir web/dist -output-dir dist -bundle-goos windows -bundle-goarch amd64

package-release-matrix:
	$(RM_DIST)
	$(MKDIR_DIST)
	$(MAKE) web-build
	$(MAKE) package-release-linux-amd64-bundle
	$(MAKE) package-release-linux-arm64-bundle
	$(MAKE) package-release-darwin-arm64-bundle
	$(MAKE) package-release-windows-bundle

certification-test:
	go test ./tests/certification/... -v

soak-test:
	go test -tags soak ./tests/soak/... -count=1

plugin-check:
	$(GO_RUN) ./scripts/plugin_manifest_check -manifest $(PLUGIN_MANIFEST) -host-version $(VERSION)

openapi-generate:
	$(GO_RUN) ./scripts/openapi_generate

contract-check:
	$(MAKE) openapi-generate
	go test ./tests/integration/... -run TestOpenAPISpec_ -count=1

release-gate:
	$(RM_DIST)
	$(RM_COVER)
	go mod tidy
	go build ./...
	go vet ./...
	golangci-lint run ./...
	$(GO_TEST_RACE_COUNT)
	$(MAKE) certification-test
	$(MAKE) soak-test
	$(MAKE) plugin-check
	$(MAKE) contract-check
	$(MAKE) build
	$(RUN_BIN) --help
	$(RUN_BIN) version
	$(RUN_BIN) start --help
	$(RUN_BIN) doctor --help
	$(RUN_BIN) status --runtime
	$(RUN_BIN) stop --help
	$(RUN_BIN) plan --help
	$(RUN_BIN) migrate --help
	$(MAKE) web-verify
	$(GO_TEST_COVER)
	go tool cover $(COVERFUNC_ARG)
	$(GO_RUN) ./scripts/coverage_gate.go coverage.out $(COVER_MIN)
	$(MAKE) package-release-matrix

clean:
	$(RM_BIN)
	$(RM_DIST)
	$(RM_COVER)
