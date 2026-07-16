.PHONY: all generate lint test build modernize

# Run the entire verification pipeline including modernizers
all: generate modernize lint test build

# Compile SQL queries into type-safe Go code using sqlc
generate:
	@echo "==> Generating type-safe Go code from SQL..."
	@sqlc generate

# Use Go 1.26's new modernizer tools to automatically update idioms
modernize:
	@echo "==> Running Go 1.26 modernizers..."
	@go fix ./...

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

# Verify compilation
build:
	@echo "==> Verifying project compilation..."
	@go build -o /dev/null ./cmd/gateway/...
