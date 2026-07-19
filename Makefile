.PHONY: test build build-mcp test-integration test-browser vet race

build:
	go build -o bin/tuile ./cmd/tuile

build-mcp:
	go build -o bin/tuile-mcp ./cmd/tuile-mcp

test:
	go test ./...

vet:
	go vet ./...

race:
	go test -race ./...

test-integration:
	go test -tags=integration ./test/integration/...

test-browser: test-integration
