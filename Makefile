.PHONY: test race coverage fmt vet build clean hooks

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

# Build CLI binary
build:
	go build -o repogov ./cmd/repogov/

# Clean build artifacts
clean:
	rm -f repogov

# Install pre-commit hook
hooks:
	cp scripts/hooks/pre-commit .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit
