SHELL = /bin/bash

# Build settings
BIN_DIR = ./bin
BUILD_DIR = ./build
PACKAGES = restclam clamctl
GO_ASMFLAGS =
GO_GCFLAGS =
GO_BUILD_ARGS = $(GO_GCFLAGS) $(GO_ASMFLAGS) -trimpath

# Lint settings
GOLANGCI_LINT_VERSION = v1.64.5
GOVULNCHECK_VERSION = latest

export GO111MODULE = on
export CGO_ENABLED = 0
export PATH := $(PWD)/$(BUILD_DIR):$(PWD)/$(BIN_DIR):$(PATH)

.PHONY: all
all: build test lint vulncheck


##@ Development

.PHONY: fix
fix: ## Fixup files in the repo.
	go mod tidy
	go fmt ./...
	./$(BIN_DIR)/golangci-lint run --fix

.PHONY: setup-tools
setup-tools:
	@mkdir -p $(BIN_DIR)
	@if [ ! -f $(BIN_DIR)/golangci-lint ]; then\
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s $(GOLANGCI_LINT_VERSION);\
	fi
	@if [ ! -f $(BIN_DIR)/govulncheck ]; then\
		GOBIN=$(PWD)/$(BIN_DIR) go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION);\
	fi

.PHONY: lint
lint: ## Run the lint check
	$(BIN_DIR)/golangci-lint run

.PHONY: vulncheck
vulncheck:
	$(BIN_DIR)/govulncheck ./...

PHONY: clean
clean: ## Cleanup build artifacts and tool binaries.
	rm -rvf $(BUILD_DIR) $(BIN_DIR)

##@ Build

.PHONY: install
install:
	go install $(GO_BUILD_ARGS)

build: $(PACKAGES)

$(PACKAGES):
	go build $(GO_BUILD_ARGS) -o $(BUILD_DIR)/ ./cmd/$@/


##@ Test

.PHONY: test
test: # Run regular unit tests
	go test -cover ./...

# run integration tests
.PHONY: integration-test
integration-test:
	go test -v -count=1 -cover -tags "integration" ./...

.PHONY: cover-report
cover-report:
	go test -cover -coverprofile coverage.out ./...
	go tool cover -func=coverage.out
	go tool cover -html=coverage.out
