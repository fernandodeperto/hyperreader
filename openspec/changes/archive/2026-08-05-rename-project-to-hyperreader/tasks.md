## 1. Canonical project identity

- [x] 1.1 Rename the Go module to `github.com/fmendonca/hyperreader` and update every internal import.
- [x] 1.2 Rename the built executable, CLI usage, help text, diagnostics, server output, and build metadata to `hyperreader`.
- [x] 1.3 Replace runtime configuration identifiers with the `HYPERREADER_*` namespace and change the default application data directory to `hyperreader` without legacy fallback.

## 2. Reader and MCP identity

- [x] 2.1 Rename MCP server metadata, tool descriptions, and unreachable-server guidance to HyperReader while preserving the `send_html` tool contract.
- [x] 2.2 Rebrand the embedded reader title, visible labels, client-side diagnostics, and browser theme-storage key as HyperReader.

## 3. Documentation and migration guidance

- [x] 3.1 Update the README, Makefile, package metadata, and all example invocations and MCP-client configurations to use HyperReader.
- [x] 3.2 Document the breaking configuration and data-directory migration, including the manual move required to retain documents from `html-mcp`.

## 4. Tests and verification

- [x] 4.1 Update unit, integration, and subprocess-MCP expectations for the HyperReader command, module, MCP identity, configuration namespace, and legacy cutoff.
- [x] 4.2 Update browser tests for the HyperReader reader identity while preserving coverage of browsing, search, and live updates.
- [x] 4.3 Run `go test ./...`, `go vet ./...`, and the Playwright e2e suite to verify the renamed public surfaces and unchanged behavior.
