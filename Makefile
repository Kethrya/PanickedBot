.PHONY: all generate build clean test vet install-tools help check-sqlc

# Default target
all: generate build

## help: Display this help message
help:
	@echo "Available targets:"
	@echo "  all           - Generate sqlc code and build (default)"
	@echo "  generate      - Generate sqlc code from SQL queries"
	@echo "  build         - Build the binary"
	@echo "  clean         - Remove generated files and binaries"
	@echo "  test          - Run tests"
	@echo "  vet           - Run go vet"
	@echo "  install-tools - Install required tools (sqlc)"
	@echo "  help          - Display this help message"

## check-sqlc: Check if sqlc is installed
check-sqlc:
	@which sqlc > /dev/null || (echo "sqlc is not installed. Run 'make install-tools' or install manually with 'go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest'" && exit 1)

## generate: Generate sqlc code from SQL queries
generate: check-sqlc
	@echo "Generating sqlc code..."
	@sqlc generate

## build: Build the binary
build:
	@echo "Building..."
	@go build -v -o PanickedBot .

## clean: Remove generated files and binaries
clean:
	@echo "Cleaning..."
	@rm -rf internal/db/sqlc
	@rm -f PanickedBot

## test: Run tests
test:
	@echo "Running tests..."
	@go test -v -race ./...

## vet: Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...

## install-tools: Install required tools (sqlc)
install-tools:
	@echo "Installing sqlc..."
	@go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	@echo "sqlc installed successfully"
	@echo "Make sure $(shell go env GOPATH)/bin is in your PATH"
