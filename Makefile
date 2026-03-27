APP_NAME := proxy-gateway
BUILD_DIR := build
LINUX_GOOS := linux
LINUX_GOARCH ?= arm64
PLUGIN_DIR := plugins

.PHONY: build-dev-linux build-prod-linux build-plugins-dev-linux build-all-dev-linux

build-dev-linux:
	mkdir -p $(BUILD_DIR)
	GOOS=$(LINUX_GOOS) GOARCH=$(LINUX_GOARCH) go build -o $(BUILD_DIR)/$(APP_NAME) .

build-prod-linux:
	mkdir -p $(BUILD_DIR)
	GOOS=$(LINUX_GOOS) GOARCH=$(LINUX_GOARCH) go build -trimpath -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME) .

build-plugins-dev-linux:
	mkdir -p $(BUILD_DIR)
	cd $(PLUGIN_DIR)/ec2-activator && CGO_ENABLED=1 GOOS=$(LINUX_GOOS) GOARCH=$(LINUX_GOARCH) go build -buildmode=plugin -o $(CURDIR)/$(BUILD_DIR)/ec2-activator.so .
	cd $(PLUGIN_DIR)/minecraft-interceptor && CGO_ENABLED=1 GOOS=$(LINUX_GOOS) GOARCH=$(LINUX_GOARCH) go build -buildmode=plugin -o $(CURDIR)/$(BUILD_DIR)/minecraft-interceptor.so .

build-all-dev-linux: build-dev-linux build-plugins-dev-linux
