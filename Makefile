SHELL := /bin/bash

MODULE  := github.com/victorseara/aipim
BINARY  := aipim
DIST    := dist
PKG     := $(MODULE)/cmd

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X $(PKG).Version=$(VERSION) \
	-X $(PKG).Commit=$(COMMIT) \
	-X $(PKG).Date=$(DATE)

.PHONY: all build install test test-race lint completions clean help

all: build

help:
	@echo "Targets:"
	@echo "  build        Build $(BINARY) into $(DIST)/$(BINARY)"
	@echo "  install      go install with version metadata"
	@echo "  test         Run unit tests"
	@echo "  test-race    Run unit tests with -race"
	@echo "  lint         Run golangci-lint (requires golangci-lint in PATH)"
	@echo "  completions  Generate shell completion scripts under $(DIST)/completions/"
	@echo "  clean        Remove the $(DIST) directory"

build:
	@mkdir -p $(DIST)
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY) .

install:
	go install -trimpath -ldflags "$(LDFLAGS)" .

test:
	go test ./...

test-race:
	go test -race ./...

lint:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not installed"; exit 1; }
	golangci-lint run ./...

completions: build
	@mkdir -p $(DIST)/completions
	$(DIST)/$(BINARY) completion bash       > $(DIST)/completions/$(BINARY).bash
	$(DIST)/$(BINARY) completion zsh        > $(DIST)/completions/_$(BINARY)
	$(DIST)/$(BINARY) completion fish       > $(DIST)/completions/$(BINARY).fish
	$(DIST)/$(BINARY) completion powershell > $(DIST)/completions/$(BINARY).ps1
	@echo "Wrote completions to $(DIST)/completions/"

clean:
	rm -rf $(DIST)
