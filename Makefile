SHELL = /bin/bash

PROJECT_ROOT = $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

# Setting GOBIN and PATH ensures two things:
# - All 'go install' commands we run
#   only affect the current directory.
# - All installed tools are available on PATH
#   for commands like go generate.
export GOBIN = $(PROJECT_ROOT)/bin
export PATH := $(GOBIN):$(PATH)

STITCHMD = bin/stitchmd
STITCHMD_ARGS = -o README.md -preface doc/preface.txt doc/README.md
TEST_FLAGS ?= -v -race

# Non-test Go files.
GO_SRC_FILES = $(shell find . \
	   -path '*/.*' -prune -o \
	   '(' -type f -a -name '*.go' -a -not -name '*_test.go' ')' -print)


.PHONY: all
all: build lint test

.PHONY: build
build: $(STITCHMD)

.PHONY: lint
lint: golangci-lint tidy-lint readme-lint

.PHONY: test
test:
	go test $(TEST_FLAGS) ./...

.PHONY: cover
cover:
	go test $(TEST_FLAGS) -coverprofile=cover.out -coverpkg=./... ./...
	go tool cover -html=cover.out -o cover.html

.PHONY: readme
readme: README.md

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: readme-lint
readme-lint:
	@DIFF=$$($(STITCHMD) -color -d $(STITCHMD_ARGS)); \
	if [[ -n "$$DIFF" ]]; then \
	        echo "README.md is out of date:"; \
	        echo "$$DIFF"; \
	        false; \
	fi

.PHONY: golangci-lint
golangci-lint:
	golangci-lint run

.PHONY: tidy-lint
tidy-lint:
	@echo "[lint] go mod tidy"
	@go mod tidy && \
		git diff --exit-code -- go.mod go.sum || \
		(echo "'go mod tidy' changed files" && false)

README.md: $(wildcard doc/*) $(STITCHMD)
	$(STITCHMD) $(STITCHMD_ARGS)

$(STITCHMD): $(GO_SRC_FILES)
	go install go.abhg.dev/stitchmd
