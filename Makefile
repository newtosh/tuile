.PHONY: test build build-mcp test-integration test-browser test-web vet race viewer-dev

build:
	go build -o bin/tuile ./cmd/tuile

build-mcp:
	go build -o bin/tuile-mcp ./cmd/tuile-mcp

viewer-dev:
	@chmod +x scripts/viewer-dev-restart.sh scripts/viewer-demo-sessions.sh
	@./scripts/viewer-dev-restart.sh

test:
	go test ./...

test-web:
	cd web && node --test *.test.js

vet:
	go vet ./...

race:
	go test -race ./...

test-integration:
	go test -tags=integration ./test/integration/...

test-browser: test-integration
