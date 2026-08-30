GOFLAGS ?= -trimpath
CGO_ENABLED ?= 0
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
REPO ?= voronkovd/gamayun
LDFLAGS ?= -s -w -X github.com/voronkovd/gamayun/internal/version.Version=$(VERSION) -X github.com/voronkovd/gamayun/internal/version.Repo=$(REPO)

.PHONY: build build-local test fmt deb

build:
	@mkdir -p dist
	GOOS=linux GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o dist/gamayun-linux-amd64 ./cmd/gamayun
	GOOS=linux GOARCH=arm64 CGO_ENABLED=$(CGO_ENABLED) go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o dist/gamayun-linux-arm64 ./cmd/gamayun

build-local:
	@mkdir -p dist
	go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o dist/gamayun ./cmd/gamayun

deb: build
	./scripts/build-deb.sh "$(VERSION)" amd64 dist/gamayun-linux-amd64
	./scripts/build-deb.sh "$(VERSION)" arm64 dist/gamayun-linux-arm64

# darwin 27 + Go 1.22: internal linker omits LC_UUID; external linker is required to run tests.
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
TESTFLAGS ?= -ldflags=-linkmode=external
endif

test:
	go test $(TESTFLAGS) ./...

fmt:
	go fmt ./...
