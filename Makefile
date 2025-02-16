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
	@NEW_VERSION=$$(semver bump major $(CURRENT_VERSION)) && \
	echo "Creating major release: $$NEW_VERSION" && \
	git tag -a $$NEW_VERSION -m "Major release $$NEW_VERSION" && \
	git push origin $$NEW_VERSION

release-minor: ## Create a new minor release
	@echo "Current version: $(CURRENT_VERSION)"
	@NEW_VERSION=$$(semver bump minor $(CURRENT_VERSION)) && \
	echo "Creating minor release: $$NEW_VERSION" && \
	git tag -a $$NEW_VERSION -m "Minor release $$NEW_VERSION" && \
	git push origin $$NEW_VERSION

release-patch: ## Create a new patch release
	@echo "Current version: $(CURRENT_VERSION)"
	@NEW_VERSION=$$(semver bump patch $(CURRENT_VERSION)) && \
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