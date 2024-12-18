PWD=$(shell pwd)
BUILD=$(PWD)/build
VERSION=$(shell git describe --tags --dirty)

GO=go
GOFLAGS=-N -l
LDFLAGS=-X main.Version=$(VERSION)

TARGET=$(BUILD)/bin/shac

$(TARGET): $(BUILD)/bin shac.go
	$(GO) build -gcflags="$(GOFLAGS)" -ldflags="$(LDFLAGS)" -o $@

$(BUILD)/bin:
	mkdir -p $@

.PHONY: clean
clean:
	rm -rf $(BUILD)

.PHONY: format
format:
	gofmt -w shac.go
