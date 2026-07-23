.PHONY: test build build-mcp test-integration test-browser test-web test-export-background vet race viewer-dev

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

test-export-background:
	@chmod +x scripts/verify-export-custom-background.sh
	@./scripts/verify-export-custom-background.sh

test-export-svg:
	@chmod +x scripts/verify-export-svg.sh scripts/audit-export-svg.py
	@./scripts/verify-export-svg.sh

capture-readme-screenshots:
	@./scripts/capture-readme-screenshots.sh

vet:
	go vet ./...

race:
	go test -race ./...

test-integration:
	go test -tags=integration ./test/integration/...

test-control-input:
	@chmod +x scripts/test-control-input.sh scripts/test-control-input-playwright.cjs
	@./scripts/test-control-input.sh

test-browser: test-integration
