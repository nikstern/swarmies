GOCACHE := $(CURDIR)/.cache/go-build
GOMODCACHE := $(CURDIR)/.cache/go-mod
GOSUMDB ?= off
GOENV := GOCACHE="$(GOCACHE)" GOMODCACHE="$(GOMODCACHE)" GOSUMDB="$(GOSUMDB)"

.PHONY: test build fmt

test:
	@mkdir -p "$(GOCACHE)" "$(GOMODCACHE)"
	$(GOENV) go test ./...

build:
	@mkdir -p "$(GOCACHE)" "$(GOMODCACHE)"
	$(GOENV) go build ./...

fmt:
	gofmt -w $(shell rg --files -g '*.go')
