# hyperreader: dev convenience targets.
#
# Wraps the commands documented in README.md's Build / Run the server /
# Run the MCP server / Run tests / Browser e2e smoke tests sections, plus
# the release build described in README.md's Releasing section. There is
# no CI and no Docker: `release` is run by hand on the maintainer's
# machine and the artifacts are published with `gh release create`.

BINARY := hyperreader
DIST := dist

# GOOS/GOARCH pairs published in a release, as raw executables (no archives).
PLATFORMS := darwin/arm64 darwin/amd64 linux/amd64 linux/arm64

# macOS ships `shasum`, Linux ships `sha256sum`. Pick whichever exists here.
SHA256 := $(shell command -v sha256sum >/dev/null 2>&1 && echo sha256sum || echo 'shasum -a 256')

.PHONY: all build serve mcp test vet fmt fmt-fix check e2e-install e2e release dist-clean clean help

all: build ## Build the hyperreader binary (default target)

# Deliberately unstripped: development panics keep readable symbol names.
# Release artifacts are stripped separately, by the `release` target.
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

release: dist-clean ## Cross-compile stripped release executables + SHA256SUMS into dist/
	@mkdir -p $(DIST)
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; arch=$${platform#*/}; \
		echo "  building $(BINARY)-$$os-$$arch"; \
		GOOS=$$os GOARCH=$$arch go build -trimpath -ldflags="-s -w" \
			-o $(DIST)/$(BINARY)-$$os-$$arch . || exit 1; \
	done
	@cd $(DIST) && $(SHA256) $(BINARY)-* > SHA256SUMS
	@echo "  wrote $(DIST)/SHA256SUMS"

dist-clean: ## Remove the dist/ release output directory
	rm -rf $(DIST)

clean: dist-clean ## Remove the built binary and the dist/ directory
	rm -f $(BINARY)

help: ## Show this help
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?## / {printf "  %-14s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
