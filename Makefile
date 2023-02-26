SHELL = /bin/bash

# Setting GOBIN and PATH ensures two things:
# - All 'go install' commands we run
#   only affect the current directory.
# - All installed tools are available on PATH
#   for commands like go generate.
export GOBIN ?= $(dir $(abspath $(lastword $(MAKEFILE_LIST))))/bin
export PATH := $(GOBIN):$(PATH)

MODULES ?= . ./tools
TEST_FLAGS ?= -race

STATICCHECK = bin/staticcheck
REVIVE = bin/revive
STITCHMD = bin/stitchmd

# All known Go files.
GO_FILES = $(shell find . \
	   -path '*/.*' -prune -o \
	   '(' -type f -a -name '*.go' ')' -print)

# Non-test Go files.
GO_SRC_FILES = $(shell find . \
	   -path '*/.*' -prune -o \
	   '(' -type f -a -name '*.go' -a -not -name '*_test.go' ')' -print)

# All known go.mod and go.sum files.
GO_MOD_FILES = \
	$(patsubst %,%/go.mod,$(MODULES)) \
	$(patsubst %,%/go.sum,$(MODULES))


.PHONY: all
all: lint build test

.PHONY: build
build: $(STITCHMD)

.PHONY: lint
lint: fmtcheck tidycheck staticcheck readmecheck revive

.PHONY: test
test:
	go test $(TEST_FLAGS) ./...

.PHONY: cover
cover:
	go test $(TEST_FLAGS) -coverprofile=cover.out -coverpkg=./... ./...
	go tool cover -html=cover.out -o cover.html

.PHONY: fmt
fmt:
	gofmt -w -s $(GO_FILES)

.PHONY: readme
readme: README.md

.PHONY: tidy
tidy:
	$(foreach dir,$(MODULES),(cd $(dir) && go mod tidy) &&) true

.PHONY: fmtcheck
fmtcheck:
	@DIFF=$$(gofmt -d -s $(GO_FILES)); \
	if [[ -n "$$DIFF" ]]; then \
		echo "gofmt would cause changes:"; \
		echo "$$DIFF"; \
		false; \
	fi

.PHONY: readmecheck
readmecheck:
	make readme
	@DIFF=$$(git diff README.md); \
	if [[ -n "$$DIFF" ]]; then \
		echo "README.md is out of date:"; \
		echo "$$DIFF"; \
		false; \
	fi

.PHONY: staticcheck
staticcheck: $(STATICCHECK)
	staticcheck ./...

.PHONY: revive
revive: $(REVIVE)
	revive -set_exit_status ./...

.PHONY: tidycheck
tidycheck:
	make tidy
	@if ! git diff --quiet $(GO_MOD_FILES); then \
		echo "go mod tidy changed files:" && \
		git status --porcelain $(GO_MOD_FILES) && \
		false; \
	fi

README.md: $(wildcard doc/*.md) $(STITCHMD)
	$(STITCHMD) -o README.md doc/README.md

$(STITCHMD): $(GO_SRC_FILES)
	go build -o $(STITCHMD)

$(STATICCHECK): tools/go.mod
	cd tools && go install honnef.co/go/tools/cmd/staticcheck

$(REVIVE): tools/go.mod
	cd tools && go install github.com/mgechev/revive
