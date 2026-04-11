BINARY_NAME := fleet
BUILD_DIR := bin

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    := $(shell date -u +%Y-%m-%d)
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

.PHONY: dev install test build clean lint

dev: build
	./$(BUILD_DIR)/$(BINARY_NAME)

install: build
	rm -f $(shell go env GOPATH)/bin/$(BINARY_NAME)
	cp $(BUILD_DIR)/$(BINARY_NAME) $(shell go env GOPATH)/bin/$(BINARY_NAME)

test:
	go test ./... -v -race

build:
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) .

clean:
	rm -rf $(BUILD_DIR)
	go clean

lint:
	go vet ./...
