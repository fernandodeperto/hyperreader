## MODIFIED Requirements

### Requirement: HyperReader MCP identity

The system SHALL identify its MCP server and MCP-client registration guidance as HyperReader while preserving the `send_html` tool name. The tool's arguments and result contract are governed by HyperReader's page model, not frozen by this identity requirement, and MAY change when that model changes, provided the tool remains named `send_html` and the server continues to identify itself as HyperReader.

#### Scenario: MCP client initializes the server
- **WHEN** an MCP client initializes the `hyperreader mcp` process
- **THEN** the server identifies itself as HyperReader
- **AND** the client can invoke `send_html` under its current, documented argument and result contract

#### Scenario: The page model changes what `send_html` accepts or returns
- **WHEN** a change to HyperReader's page model adds, removes, or renames one of `send_html`'s arguments or result fields
- **THEN** the tool remains named `send_html`
- **AND** the server still identifies itself as HyperReader to the initializing client
