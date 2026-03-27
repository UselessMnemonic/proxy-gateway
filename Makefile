APP_NAME := proxy-gateway
BUILD_DIR := build
LINUX_GOOS := linux
LINUX_GOARCH ?= arm64
PLUGIN_DIR := plugins
LINUX_CC ?= aarch64-linux-gnu-gcc
LDFLAGS := -linkmode=external -extldflags=-fuse-ld=bfd

.PHONY: build-linux build-plugins-linux build-all-linux

build-linux:
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 GOOS=$(LINUX_GOOS) GOARCH=$(LINUX_GOARCH) CC=$(LINUX_CC) go build -trimpath -ldflags='$(LDFLAGS)' -o $(BUILD_DIR)/$(APP_NAME) .

build-plugins-linux:
	mkdir -p $(BUILD_DIR)
	cd $(PLUGIN_DIR)/ec2              && CGO_ENABLED=1 GOOS=$(LINUX_GOOS) GOARCH=$(LINUX_GOARCH) CC=$(LINUX_CC) go build -trimpath -buildmode=plugin -ldflags='$(LDFLAGS)' -o $(CURDIR)/$(BUILD_DIR)/ec2.so .
	cd $(PLUGIN_DIR)/minecraft-server && CGO_ENABLED=1 GOOS=$(LINUX_GOOS) GOARCH=$(LINUX_GOARCH) CC=$(LINUX_CC) go build -trimpath -buildmode=plugin -ldflags='$(LDFLAGS)' -o $(CURDIR)/$(BUILD_DIR)/minecraft-server.so .

build-all-linux: build-linux build-plugins-linux
