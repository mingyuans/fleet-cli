BINARY_NAME := fleet
BUILD_DIR := bin

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
	go build -o $(BUILD_DIR)/$(BINARY_NAME) .

clean:
	rm -rf $(BUILD_DIR)
	go clean

lint:
	go vet ./...
