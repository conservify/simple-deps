GOARCH ?= amd64
GOOS ?= linux
GO ?= env GOOS=$(GOOS) GOARCH=$(GOARCH) go
BUILD ?= build

all: $(BUILD)/simple-deps $(BUILD)/dependencies.cmake.template

$(BUILD)/simple-deps: *.go
	$(GO) get gopkg.in/src-d/go-git.v4
	$(GO) build -o $@ $^

$(BUILD)/dependencies.cmake.template: dependencies.cmake.template
	cp dependencies.cmake.template $(BUILD)

$(BUILD):
	mkdir -p $(BUILD)

clean:
	rm -rf $(BUILD)

install: $(BUILD)/simple-deps
	cp $(BUILD)/simple-deps ~/tools/bin
	cp *.template ~/tools/bin
