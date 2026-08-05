## Purpose

Give the HTML artifact reader a single public identity that describes its role as a reader of agent-produced hypertext documents.

## ADDED Requirements

### Requirement: HyperReader command identity

The system SHALL identify its executable, command usage, help text, diagnostics, build output, and supported invocation examples as `hyperreader`. It SHALL retain the existing `serve` and `mcp` subcommands and their behavior.

#### Scenario: User requests command help
- **WHEN** a user runs `hyperreader --help`
- **THEN** the usage banner and subcommand guidance identify the command as `hyperreader`
- **AND** the help describes the existing `serve` and `mcp` subcommands

### Requirement: HyperReader reader identity

The system SHALL identify its browser reader, server messages, and user-visible documentation as HyperReader. The browser reader SHALL preserve its existing document browsing, search, and live-update behavior.

#### Scenario: User opens the reader
- **WHEN** a user opens the served root page
- **THEN** the document title and primary reader label identify the application as HyperReader
- **AND** the document list and search interface remain available

### Requirement: HyperReader MCP identity

The system SHALL identify its MCP server and MCP-client registration guidance as HyperReader while preserving the `send_html` tool name, arguments, and result contract.

#### Scenario: MCP client initializes the server
- **WHEN** an MCP client initializes the `hyperreader mcp` process
- **THEN** the server identifies itself as HyperReader
- **AND** the client can invoke `send_html` with the existing request and response behavior
