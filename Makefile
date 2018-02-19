GOARCH ?= amd64
GO ?= env GOOS=linux GOARCH=$(GOARCH) go

simple-deps: simple-deps.go
	$(GO) build -o simple-deps simple-deps.go

install: simple-deps
	cp simple-deps ~/tools/bin
