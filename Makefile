BINARY_NAME=tdd-ai
VERSION?=0.4.1
BUILD_DIR=bin

.PHONY: build test test-short test-race lint coverage ci clean install

build:
	go build -ldflags "-X github.com/macosta/tdd-ai/cmd.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) .

test:
	go test ./... -v

test-short:
	go test ./... -short

test-race:
	go test -race ./...

lint:
	golangci-lint run

ci: lint test-race build
	@echo "CI passed."

coverage:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out

install: build
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME) 2>/dev/null || \
	cp $(BUILD_DIR)/$(BINARY_NAME) ~/go/bin/$(BINARY_NAME)
