## Why

`html-mcp` names the transport and document format, not the user-facing purpose: reading persistent HTML artifacts produced by agents. Rename the project to HyperReader so its command, UI, MCP identity, and configuration describe a hypertext reader.

## What Changes

- **BREAKING** Rename the public project identity from `html-mcp` to `hyperreader`, including the Go module path, built executable, CLI usage and diagnostics, Makefile targets, MCP server registration guidance, and documentation.
- **BREAKING** Rename the default application data directory and environment-variable namespace from `html-mcp`/`HTML_MCP_*` to `hyperreader`/`HYPERREADER_*`, without retaining legacy runtime aliases.
- Rebrand the embedded web UI and browser-persisted theme preference as HyperReader.
- Keep the existing `serve` and `mcp` subcommands, localhost HTTP API, `send_html` tool contract, document storage model, search, and live-update behavior unchanged.
- Update unit, integration, and browser tests to validate the renamed public surfaces.

## Capabilities

### New Capabilities
- `project-identity`: HyperReader provides a consistent public identity across its executable, MCP server, web reader, source module, and documentation.
- `configuration-identity`: HyperReader resolves its data location and configuration overrides through HyperReader-branded identifiers.

### Modified Capabilities
- None. The project has no existing OpenSpec capability specifications.

## Impact

- Affected code: `go.mod`, CLI entrypoint, configuration package, MCP server metadata, server messages, embedded web assets, and all related tests.
- Affected developer surfaces: `README.md`, `Makefile`, `package.json` metadata, test fixture references, and MCP-client configuration examples.
- User migration: installations, scripts, MCP-client registrations, and environment variables must move to the new name. Existing stored documents must be moved from the old application data directory to HyperReader's data directory before startup if they are to remain available.
