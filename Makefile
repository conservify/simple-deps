GOARCH ?= amd64
GO ?= env GOOS=linux GOARCH=$(GOARCH) go
BUILD ?= build

$(BUILD)/simple-deps: *.go
	$(GO) build -o $@ $^

$(BUILD):
	mkdir -p $(BUILD)

clean:
	rm -rf $(BUILD)

install: $(BUILD)/simple-deps
	cp $(BUILD)/simple-deps ~/tools/bin
