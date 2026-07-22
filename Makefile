# nRF Factory build / package / release helpers.
# Pure-Go, no cgo. See tickets/build-01, build-02, package-02.

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

PKG     := ./cmd/nrf-factory
DIST    := dist
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)
GOBUILD := CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)"

BIN_WIN     := nrf-factory-windows-amd64.exe
BIN_MAC_ARM := nrf-factory-darwin-arm64

.PHONY: all fmt vet test build dist package clean

all: test build

fmt:
	go fmt ./...

vet:
	go vet ./...

test:
	go test ./...

# Build for the host OS/arch into dist/.
build:
	mkdir -p $(DIST)
	$(GOBUILD) -o $(DIST)/nrf-factory $(PKG)

# Cross-compile every release target into dist/.
dist:
	mkdir -p $(DIST)
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(DIST)/$(BIN_WIN) $(PKG)
	GOOS=darwin  GOARCH=arm64 $(GOBUILD) -o $(DIST)/$(BIN_MAC_ARM) $(PKG)

# Zip each platform, then write SHA256SUMS. Paths are kept repo-relative so the
# extracted zip mirrors the dev tree exactly (binary under dist/, launcher and
# docs at the root) — debug and release layouts are identical on both OSes.
package: dist
	rm -f $(DIST)/*.zip $(DIST)/SHA256SUMS
	zip -q $(DIST)/nrf-factory-$(VERSION)-windows-amd64.zip $(DIST)/$(BIN_WIN) README.md docs/INSTALL-TOOLS.md
	zip -q $(DIST)/nrf-factory-$(VERSION)-darwin-arm64.zip  $(DIST)/$(BIN_MAC_ARM) README.md docs/INSTALL-TOOLS.md
	cd $(DIST) && ( command -v sha256sum >/dev/null 2>&1 && sha256sum *.zip || shasum -a 256 *.zip ) > SHA256SUMS
	@echo "packaged $(VERSION) into $(DIST)/"

clean:
	rm -rf $(DIST)
