# Set shell to bash
SHELL := /bin/bash

# Current version of the project.x`
VERSION ?= v0.0.3

# This repo's root import path (under GOPATH).
ROOT := github.com/cd1989/harbor-cleaner

# A list of all packages.
PKGS := $(shell go list ./... | grep -v /vendor | grep -v /test)

# Git commit sha.
COMMIT := $(shell git rev-parse --short HEAD)

# Project main package location (can be multiple ones).
CMD_DIR := ./cmd

# Project output directory.
OUTPUT_DIR := ./bin

# Build direcotory.
BUILD_DIR := ./build

# Golang standard bin directory.
BIN_DIR := $(GOPATH)/bin
GOMETALINTER := $(BIN_DIR)/gometalinter

UNAME := $(shell uname)

# All targets.
.PHONY: lint test build

lint: $(GOMETALINTER)
	golangci-lint run --disable=gosimple --deadline=300s ./pkg/... ./cmd/...

$(GOMETALINTER):
	go get -u github.com/alecthomas/gometalinter
	gometalinter --install &> /dev/null

test:
	go test $(PKGS)

build-local:
	CGO_ENABLED=0 GOARCH=amd64 go build -i -v -o ./bin/harbor-cleaner -ldflags "-s -w" ./cmd

build:
	docker run --rm                                                                \
	  -v $(PWD):/go/src/$(ROOT)                                                    \
	  -w /go/src/$(ROOT)                                                           \
	  -e GOOS=linux                                                                \
	  -e GOARCH=amd64                                                              \
	  -e GOPATH=/go                                                                \
	  -e CGO_ENABLED=0                                                             \
	    golang:1.10-alpine3.8                                                      \
	      go build -i -v -o $(OUTPUT_DIR)/cleaner                                  \
	        -ldflags "-s -w -X $(ROOT)/pkg/version.VERSION=$(VERSION)              \
	        -X $(ROOT)/pkg/version.COMMIT=$(COMMIT)                                \
	        -X $(ROOT)/pkg/version.REPOROOT=$(ROOT)"                               \
	        $(CMD_DIR);

image: build
	docker build -t k8sdevops/harbor-cleaner:$(VERSION) -f ./build/Dockerfile .

.PHONY: clean
clean:
	-rm -vrf ${OUTPUT_DIR}