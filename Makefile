.PHONY: test race coverage fmt vet lint build clean hooks

# Run all tests
test:
	go test ./...

# Run tests with race detector
race:
	go test ./... -race

# Run with coverage
coverage:
	go test ./... -cover

# Format code
fmt:
	gofmt -w .

# Vet
vet:
	go vet ./...

# Lint (requires golangci-lint on PATH)
lint:
	golangci-lint run ./...

# Build CLI binary
build:
	go build -o repogov ./cmd/repogov/

# Clean build artifacts (cross-platform: rm -f is Unix-only and breaks
# under cmd.exe; Make execs `go` directly so tools/xplat has no shell.)
clean:
	@go run ./tools/xplat rmrf repogov

# Install pre-commit hook (cross-platform: cp/chmod are Unix-only;
# tools/xplat cp also marks the hook executable for POSIX git.)
hooks:
	@go run ./tools/xplat cp scripts/hooks/pre-commit .git/hooks/pre-commit
