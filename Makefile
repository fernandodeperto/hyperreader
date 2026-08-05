.PHONY: build serve mcp test vet fmt fmt-fix e2e-install e2e check clean

BINARY := html-mcp

## build: compile the html-mcp binary (go build ./...)
build:
	go build -o $(BINARY) .

## serve: run the long-lived HTTP server (go run . serve)
serve:
	go run . serve

## mcp: run the stdio MCP server (go run . mcp) — normally launched by an MCP client, not by hand
mcp:
	go run . mcp

## test: run Go unit + integration tests (go test ./...)
test:
	go test ./...

## vet: run go vet ./...
vet:
	go vet ./...

## fmt: check formatting without modifying files (gofmt -l .)
fmt:
	gofmt -l .

## fmt-fix: reformat all Go files in place (gofmt -w .)
fmt-fix:
	gofmt -w .

## e2e-install: install npm deps + Playwright's chromium browser (first run only)
e2e-install:
	npm install
	npx playwright install chromium

## e2e: run the Playwright browser smoke suite against the real serve binary (npm run test:e2e)
e2e:
	npm run test:e2e

## check: build, vet, fmt-check, and unit tests — the fast local verification loop
check: build vet fmt test

## clean: remove the built binary
clean:
	rm -f $(BINARY)
