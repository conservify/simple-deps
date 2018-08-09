GOARCH ?= amd64
GO ?= env GOOS=linux GOARCH=$(GOARCH) go
BUILD ?= build

all: $(BUILD)/simple-deps $(BUILD)/dependencies.cmake.template

$(BUILD)/simple-deps: *.go
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
