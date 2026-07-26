BIN_DIR := bin
BINARY := $(BIN_DIR)/batz
BUILD_FLAGS ?=
LDFLAGS ?=

GO_BUILD_ARGS := $(BUILD_FLAGS)
ifneq ($(strip $(LDFLAGS)),)
GO_BUILD_ARGS += -ldflags="$(LDFLAGS)"
endif

.PHONY: all build check clean fmt-check install test test-race vet

all: build

build:
	mkdir -p $(BIN_DIR)
	go build $(GO_BUILD_ARGS) -o $(BINARY) ./cmd/batz

fmt-check:
	test -z "$$(gofmt -l cmd internal)"

vet:
	go vet ./...

test:
	go test ./...

test-race:
	go test -race ./...

check: fmt-check vet test

install:
	go install ./cmd/batz

clean:
	rm -f $(BINARY)
	rmdir $(BIN_DIR) 2>/dev/null || true
