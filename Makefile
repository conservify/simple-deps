GOARCH ?= amd64
GO ?= env GOOS=linux GOARCH=$(GOARCH) go
BUILD ?= build

$(BUILD)/simple-deps: simple-deps.go
	$(GO) build -o $(BUILD)/simple-deps simple-deps.go

$(BUILD):
	mkdir -p $(BUILD)

clean:
	rm -rf $(BUILD)

install: $(BUILD)/simple-deps
	cp $(BUILD)/simple-deps ~/tools/bin
