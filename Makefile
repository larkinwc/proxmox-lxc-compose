.PHONY: all build test lint clean

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
BINARY_NAME=lxc-compose
MAIN_PATH=./cmd/lxc-compose

all: lint test build

build:
	$(GOBUILD) -o $(BINARY_NAME) $(MAIN_PATH)

test:
	$(GOTEST) -v -race -cover ./...

lint:
	golangci-lint run

clean:
	rm -f $(BINARY_NAME)
	go clean -testcache

# Development helpers
.PHONY: dev
dev: lint test build

# Install development tools
.PHONY: tools
tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run the application
.PHONY: run
run: build
	./$(BINARY_NAME)

# Generate mocks (if we add them later)
.PHONY: generate
generate:
	go generate ./... 