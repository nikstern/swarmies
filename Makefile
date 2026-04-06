GOCACHE := $(CURDIR)/.cache/go-build
GOMODCACHE := $(CURDIR)/.cache/go-mod

.PHONY: test build fmt

test:
	@mkdir -p "$(GOCACHE)" "$(GOMODCACHE)"
	GOCACHE="$(GOCACHE)" GOMODCACHE="$(GOMODCACHE)" go test ./...

build:
	@mkdir -p "$(GOCACHE)" "$(GOMODCACHE)"
	GOCACHE="$(GOCACHE)" GOMODCACHE="$(GOMODCACHE)" go build ./...

fmt:
	gofmt -w $(shell rg --files -g '*.go')
