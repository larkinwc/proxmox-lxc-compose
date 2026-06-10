.PHONY: all build install uninstall test lint clean

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
BINARY_NAME=lxc-compose
MAIN_PATH=./cmd/lxc-compose
PREFIX?=/usr/local

# Embed build metadata so `lxc-compose version` is meaningful for source builds.
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE?=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS=-s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.date=$(DATE)

all: lint test build

build:
	$(GOBUILD) -trimpath -ldflags '$(LDFLAGS)' -o $(BINARY_NAME) $(MAIN_PATH)

# Install the built binary to $(PREFIX)/bin (default /usr/local/bin).
install: build
	install -d $(DESTDIR)$(PREFIX)/bin
	install -m 0755 $(BINARY_NAME) $(DESTDIR)$(PREFIX)/bin/$(BINARY_NAME)
	@echo "Installed $(BINARY_NAME) to $(DESTDIR)$(PREFIX)/bin"

uninstall:
	rm -f $(DESTDIR)$(PREFIX)/bin/$(BINARY_NAME)

test:
	$(GOTEST) -v -race -cover ./...

test-debug:
	@$(GOTEST) -v ./... 2>&1 | awk '/=== RUN/{p=1}p' | awk '/ FAIL/{if(!f)print;f=1}!/FAIL/{print}' | grep -v "coverage:"

test-fails:
	@$(GOTEST) -v ./... 2>&1 | grep ": unexpected\|: expected" || true

test-short:
	@$(GOTEST) ./... -short

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

# Run a specific test: make test-one TEST=TestName [PKG=./path/to/package]
test-one:
	@$(GOTEST) -v $(if $(PKG),$(PKG),./...) $(if $(TEST),-run "$(TEST)",)

# Generate mocks (if we add them later)
.PHONY: generate
generate:
	go generate ./...

# Release helpers
.PHONY: release-major release-minor release-patch release

# Get current version
CURRENT_VERSION=$(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")

release-major: ## Create a new major release
	@echo "Current version: $(CURRENT_VERSION)"
	@NEW_VERSION=$$(echo "$(CURRENT_VERSION)" | awk -F. '{gsub("v",""); printf "v%d.0.0", $$1+1}') && \
	echo "Creating major release: $$NEW_VERSION" && \
	git tag -a $$NEW_VERSION -m "Major release $$NEW_VERSION" && \
	git push origin $$NEW_VERSION

release-minor: ## Create a new minor release
	@echo "Current version: $(CURRENT_VERSION)"
	@NEW_VERSION=$$(echo "$(CURRENT_VERSION)" | awk -F. '{gsub("v",""); printf "v%d.%d.0", $$1, $$2+1}') && \
	echo "Creating minor release: $$NEW_VERSION" && \
	git tag -a $$NEW_VERSION -m "Minor release $$NEW_VERSION" && \
	git push origin $$NEW_VERSION

release-patch: ## Create a new patch release
	@echo "Current version: $(CURRENT_VERSION)"
	@NEW_VERSION=$$(echo "$(CURRENT_VERSION)" | awk -F. '{gsub("v",""); printf "v%d.%d.%d", $$1, $$2, $$3+1}') && \
	echo "Creating patch release: $$NEW_VERSION" && \
	git tag -a $$NEW_VERSION -m "Patch release $$NEW_VERSION" && \
	git push origin $$NEW_VERSION

# Helper to show current version
version:
	@echo $(CURRENT_VERSION)

# Dry run a release to test configuration
release-dry-run:
	goreleaser release --snapshot --clean --skip-publish

# Check release configuration
release-check:
	goreleaser check 