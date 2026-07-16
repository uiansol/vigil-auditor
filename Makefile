.PHONY: all generate lint test build

# Run the entire verification pipeline
all: generate lint test build

# Compile SQL queries into type-safe Go code using sqlc
generate:
	@echo "==> Generating type-safe Go code from SQL..."
	@sqlc generate

# Run advanced static analysis and formatting
lint:
	@echo "==> Formatting imports..."
	@goimports -w .
	@echo "==> Running golangci-lint..."
	@golangci-lint run ./...

# Run the test suite
test:
	@echo "==> Running test suite..."
	@go test -v -race ./...

# Attempt a dry-run compile to guarantee the entire project builds
build:
	@echo "==> Verifying project compilation..."
	@go build -o /dev/null ./cmd/gateway/...
