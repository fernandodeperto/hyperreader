## Context

The current identity appears in the Go module path, executable and CLI text, the MCP initialization name and tool description, runtime configuration, embedded reader assets, tests, and developer documentation. The server's HTTP routes, `send_html` tool schema, storage format, and default port are already public behavior that does not need to change. See proposal.md for motivation.

## Goals / Non-Goals

**Goals:**
- Make every supported public name consistently identify the project as HyperReader.
- Preserve the existing reader and MCP behavior under the new identity.
- Establish an unambiguous configuration and data-directory cutover.

**Non-Goals:**
- Changing HTTP routes, the MCP tool contract, storage schema, document-file format, or the default port.
- Automatically importing, copying, or deleting data from the prior application directory.
- Providing executable, environment-variable, module-path, or runtime compatibility aliases.

## Decisions

### Use `hyperreader` as the canonical machine-readable name

The binary, MCP server name, command usage, errors, build metadata, data directory, browser storage key, and documentation examples will use `hyperreader`. User-facing prose and the web reader will use the display form `HyperReader`. The Go module will move from `github.com/fmendonca/html-mcp` to `github.com/fmendonca/hyperreader` and every internal import will move with it.

A clean cutover avoids a permanent second vocabulary. Keeping an `html-mcp` binary, server registration, or environment namespace would preserve migration convenience at the cost of ambiguity and ongoing support burden.

### Preserve behavioral contracts beneath the identity layer

`serve`, `mcp`, `send_html`, the localhost HTTP API, the storage schema and files, and port `7420` remain unchanged. This makes the rename testable as a public-surface migration rather than a functional rewrite.

### Rename configuration and data locations without runtime fallback

`HYPERREADER_DATA_DIR` and `HYPERREADER_PORT` replace the existing environment variables. The default application directory becomes `hyperreader`. Legacy variables and the old default directory are ignored at runtime. Documentation will provide a one-time manual move instruction for users who want to retain existing documents.

This makes prior configuration and storage a deliberate breaking boundary, rather than allowing hidden precedence between two names.

### Verify named surfaces directly

Existing focused unit, integration, subprocess-MCP, and browser tests will be updated to assert the new command, server identity, configuration names, data-directory defaults, and reader labels. Existing behavior tests will continue to protect the unchanged API and ingestion path.

## Risks / Trade-offs

- Existing scripts, installed binary paths, MCP-client registrations, environment variables, and Go imports break until users update them. This is the intended clean cutover and must be called out in documentation.
- Existing documents remain in the old data directory until manually moved. Automatic migration would add filesystem mutation and legacy fallback behavior outside this rename's scope.
- The new module path assumes `github.com/fmendonca/hyperreader` is the intended canonical repository location. Repository hosting and package availability must be confirmed before release.
