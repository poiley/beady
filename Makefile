MODULE   := github.com/poiley/beady
BINARY   := bdy
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT   := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE     := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS  := -s -w \
	-X 'main.Version=$(VERSION)' \
	-X 'main.Commit=$(COMMIT)' \
	-X 'main.Date=$(DATE)'

.PHONY: build install clean test vet lint release-dry

## build: Build the binary
build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

## install: Build and install to GOPATH/bin
install:
	go install -ldflags "$(LDFLAGS)" .

## clean: Remove build artifacts
clean:
	rm -f $(BINARY)
	rm -rf dist/

## test: Run tests
test:
	go test ./...

## vet: Run go vet
vet:
	go vet ./...

## lint: Run golangci-lint
lint:
	golangci-lint run ./...

## release-dry: Preview goreleaser build (no publish)
release-dry:
	goreleaser release --snapshot --clean

## version: Print the current version
version:
	@echo $(VERSION)

## help: Show this help
help:
	@echo "Usage: make [target]"
	@echo ""
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':'
