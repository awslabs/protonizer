PACKAGES := $(shell go list ./...)
BUILD_VERSION := $(shell git describe --tags)

all: help

.PHONY: help
help: Makefile
	@echo
	@echo " Choose a make command to run"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
	@echo

## precommit: run all pre-commit hooks
.PHONY: precommit
precommit:
	pre-commit run --all-files

## vet: vet code
.PHONY: vet
vet:
	go vet $(PACKAGES)

## test: run unit tests
.PHONY: test
test: vet
	go test -race -cover $(PACKAGES)

## build: build a binary
.PHONY: build
build:
	echo building ${BUILD_VERSION}
	go build -ldflags "-X main.version=${BUILD_VERSION}" -o ./app -v

## autobuild: auto build when source files change
.PHONY: autobuild
autobuild:
	# curl -sf https://gobinaries.com/cespare/reflex | sh
	reflex -g '*.go' -- sh -c 'echo "\n\n\n\n\n\n" && make build'

## start: build and run local project
.PHONY: start
start: build
	clear
	@echo ""
	./app

## xplat: multiplatform build
.PHONY: xplat
xplat: build
	GOOS=darwin GOARCH=amd64 go build -v -o ./dist/protonizer-darwin-amd64 -ldflags "-X main.version=${BUILD_VERSION}"
	GOOS=darwin GOARCH=arm64 go build -v -o ./dist/protonizer-darwin-arm64 -ldflags "-X main.version=${BUILD_VERSION}"
	GOOS=windows GOARCH=amd64 go build -v -o ./dist/protonizer-windows-amd64 -ldflags "-X main.version=${BUILD_VERSION}"
