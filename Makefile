BINARY     := mp2rss
PKG        := github.com/areyoubugcoder/mp2rss-cli
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS    := -s -w -X $(PKG)/internal/version.Version=$(VERSION)
BUILD_DIR  := dist
GOFLAGS    := -trimpath -ldflags "$(LDFLAGS)"

.PHONY: build build-all clean test lint install help

help:
	@echo "Targets:"
	@echo "  build        — build current host binary (-trimpath -s -w)"
	@echo "  build-all    — cross-compile darwin/linux/windows (amd64+arm64)"
	@echo "  test         — go test ./..."
	@echo "  lint         — go vet + golangci-lint (if installed)"
	@echo "  install      — build and install to ~/.local/bin (override INSTALL_DIR)"
	@echo "  clean        — remove build artifacts"

build:
	go build $(GOFLAGS) -o $(BINARY) .

build-all:
	mkdir -p $(BUILD_DIR)
	GOOS=darwin  GOARCH=amd64  go build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY)_darwin_amd64  .
	GOOS=darwin  GOARCH=arm64  go build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY)_darwin_arm64  .
	GOOS=linux   GOARCH=amd64  go build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY)_linux_amd64   .
	GOOS=linux   GOARCH=arm64  go build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY)_linux_arm64   .
	GOOS=windows GOARCH=amd64  go build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY)_windows_amd64.exe .

clean:
	rm -f $(BINARY)
	rm -rf $(BUILD_DIR)

test:
	go test ./...

lint:
	go vet ./...
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./... ; \
	else \
		echo "[lint] golangci-lint 未安装，跳过（CI 中会安装）" ; \
	fi

INSTALL_DIR ?= $(HOME)/.local/bin

install: build
	@mkdir -p $(INSTALL_DIR)
	install -m 755 $(BINARY) $(INSTALL_DIR)/$(BINARY)
	@echo "已安装到 $(INSTALL_DIR)/$(BINARY)"
