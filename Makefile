BINARY_NAME=tdd-ai
VERSION?=0.4.1
BUILD_DIR=bin

.PHONY: build test test-short lint coverage clean install

build:
	go build -ldflags "-X github.com/macosta/tdd-ai/cmd.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) .

test:
	go test ./... -v

test-short:
	go test ./... -short

lint:
	golangci-lint run

coverage:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out

install: build
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME) 2>/dev/null || \
	cp $(BUILD_DIR)/$(BINARY_NAME) ~/go/bin/$(BINARY_NAME)
