VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.Version=$(VERSION)

.DEFAULT_GOAL := build

## Build the databricks-claude binary (and the credential-helper symlink that
## the Claude Desktop mobileconfig field expects).
build:
	go build -ldflags="$(LDFLAGS)" -o databricks-claude .
	ln -sf databricks-claude databricks-claude-credential-helper

## Install to GOPATH/bin (also drops the credential-helper symlink so Claude
## Desktop's inferenceCredentialHelper can target a stable path).
install:
	go install -ldflags="$(LDFLAGS)" .
	ln -sf databricks-claude "$$(go env GOPATH)/bin/databricks-claude-credential-helper"

## Run tests with verbose output
test:
	go test ./... -v

## Cross-compile for linux/darwin/windows amd64 + arm64. Symlinks for the
## credential-helper alias are NOT generated here — packagers (brew, .pkg,
## .deb) are responsible for creating them at install time pointing at a
## predictable system path.
dist:
	mkdir -p dist
	GOOS=darwin  GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o dist/databricks-claude-darwin-arm64  .
	GOOS=darwin  GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/databricks-claude-darwin-amd64  .
	GOOS=linux   GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/databricks-claude-linux-amd64   .
	GOOS=linux   GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o dist/databricks-claude-linux-arm64   .
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/databricks-claude-windows-amd64.exe .
	GOOS=windows GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o dist/databricks-claude-windows-arm64.exe .

## Remove build artifacts
clean:
	rm -f databricks-claude databricks-claude-credential-helper
	rm -rf dist/

## Run go vet
lint:
	go vet ./...

.PHONY: build install test dist clean lint
