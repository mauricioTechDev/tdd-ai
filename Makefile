BINARY_NAME=tdd-ai
VERSION?=0.4.1
BUILD_DIR=bin

.PHONY: build test lint clean install

build:
	go build -ldflags "-X github.com/macosta/tdd-ai/cmd.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) .

test:
	go test ./... -v

test-short:
	go test ./... -short

lint:
	go vet ./...

clean:
	rm -rf $(BUILD_DIR)

install: build
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME) 2>/dev/null || \
	cp $(BUILD_DIR)/$(BINARY_NAME) ~/go/bin/$(BINARY_NAME)
