SHELL = /bin/bash

# Setting GOBIN and PATH ensures two things:
# - All 'go install' commands we run
#   only affect the current directory.
# - All installed tools are available on PATH
#   for commands like go generate.
export GOBIN ?= $(dir $(abspath $(lastword $(MAKEFILE_LIST))))/bin
export PATH := $(GOBIN):$(PATH)

TEST_FLAGS ?= -race

STITCHMD = bin/stitchmd
STITCHMD_ARGS = -o README.md -preface doc/preface.txt doc/README.md

# Non-test Go files.
GO_SRC_FILES = $(shell find . \
	   -path '*/.*' -prune -o \
	   '(' -type f -a -name '*.go' -a -not -name '*_test.go' ')' -print)

.PHONY: all
all: lint build test

.PHONY: build
build: $(STITCHMD)

.PHONY: lint
lint: tidy-lint readme-lint golangci-lint

.PHONY: test
test:
	go test $(TEST_FLAGS) ./...

.PHONY: cover
cover:
	go test $(TEST_FLAGS) -coverprofile=cover.out -coverpkg=./... ./...
	go tool cover -html=cover.out -o cover.html

.PHONY: fmt
fmt:
	gofumpt -w -s .

.PHONY: readme
readme: README.md

.PHONY: readme-lint
readme-lint: $(STITCHMD)
	@echo "[lint] Checking README.md."
	@DIFF=$$($(STITCHMD) -color -d $(STITCHMD_ARGS)); \
	if [[ -n "$$DIFF" ]]; then \
		echo "README.md is out of date:"; \
		echo "$$DIFF"; \
		false; \
	fi

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: tidy-lint
tidy-lint:
	@echo "[lint] Checking go.mod files."
	@go mod tidy -v && \
	if ! git diff --exit-code go.mod go.sum; then \
		echo "go.mod files are out of date" && \
		false; \
	fi

.PHONY: golangci-lint
golangci-lint:
	@echo "[lint] Running golangci-lint."
	@golangci-lint run

README.md: $(wildcard doc/*) $(STITCHMD)
	$(STITCHMD) $(STITCHMD_ARGS)

$(STITCHMD): $(GO_SRC_FILES)
	go build -o $(STITCHMD)
