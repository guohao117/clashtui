.PHONY: build run clean debug fmt vet lint

BINARY := clashtui
BUILD_DIR := bin

build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/

run: build
	$(BUILD_DIR)/$(BINARY)

debug:
	go run cmd/debug.go

clean:
	rm -rf $(BUILD_DIR)
	go clean

fmt:
	go fmt ./...

vet:
	go vet ./...

lint: fmt vet

test:
	go test ./...
