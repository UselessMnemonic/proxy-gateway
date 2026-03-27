APP_NAME := proxy-gateway
BUILD_DIR := build
LINUX_GOOS := linux
LINUX_GOARCH ?= arm64

.PHONY: build-dev-linux build-prod-linux

build-dev-linux:
	mkdir -p $(BUILD_DIR)
	GOOS=$(LINUX_GOOS) GOARCH=$(LINUX_GOARCH) go build -o $(BUILD_DIR)/$(APP_NAME)-dev .

build-prod-linux:
	mkdir -p $(BUILD_DIR)
	GOOS=$(LINUX_GOOS) GOARCH=$(LINUX_GOARCH) go build -trimpath -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME) .
