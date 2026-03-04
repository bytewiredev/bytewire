.PHONY: build test lint clean wasm example-counter

# Build the bytewire CLI
build:
	go build -o bin/bytewire ./cmd/bytewire

# Compile the WASM client module
wasm:
	GOOS=js GOARCH=wasm go build -o dist/bytewire.wasm ./pkg/wasm

# Run all tests
test:
	go test ./...

# Run linter
lint:
	go vet ./...

# Remove build artifacts
clean:
	rm -rf bin/ dist/

# Run the counter example (in-tree)
example-counter:
	go run ./examples/counter

# Install the CLI to $GOPATH/bin
install:
	go install ./cmd/bytewire
