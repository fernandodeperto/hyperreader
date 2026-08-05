# hyperreader: dev convenience targets.
#
# Wraps the commands documented in README.md's Build / Run the server /
# Run the MCP server / Run tests / Browser e2e smoke tests sections. This
# is a local-dev convenience layer only — no CI, Docker, or release/
# cross-compile targets exist here because none of that exists in the
# project yet.

BINARY := hyperreader

.PHONY: all build serve mcp test vet fmt fmt-fix check e2e-install e2e clean help

all: build ## Build the hyperreader binary (default target)

build: ## go build ./... -> ./hyperreader
	go build -o $(BINARY) .

serve: ## go run . serve (long-lived HTTP server, default port/data-dir)
	go run . serve

mcp: ## go run . mcp (stdio MCP server; normally launched by an MCP client, not by hand)
	go run . mcp

test: ## go test ./... (unit + integration, includes real MCP subprocess handshake)
	go test ./...

vet: ## go vet ./...
	go vet ./...

fmt: ## gofmt -l . (list files needing formatting; does not modify anything)
	gofmt -l .

fmt-fix: ## gofmt -w . (rewrite files in place to match gofmt)
	gofmt -w .

check: vet fmt test ## Run vet + fmt + test together (pre-commit convenience)

e2e-install: ## npm install + Playwright chromium browser (first run only)
	npm install
	npx playwright install chromium

e2e: ## npm run test:e2e (Playwright smoke against the real serve binary)
	npm run test:e2e

clean: ## Remove the built binary
	rm -f $(BINARY)

help: ## Show this help
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?## / {printf "  %-14s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
