# configuration-identity Specification

## Purpose

Align HyperReader's local storage and configuration namespace with its public identity while making the breaking migration boundary explicit.

## Requirements

### Requirement: HyperReader configuration namespace

The system SHALL use `HYPERREADER_DATA_DIR` and `HYPERREADER_PORT` as its supported environment-variable overrides. It SHALL resolve its default application data directory under `hyperreader`.

#### Scenario: User configures the data directory
- **WHEN** a user sets `HYPERREADER_DATA_DIR`
- **THEN** HyperReader stores and reads documents from the configured directory

#### Scenario: User configures the listen port
- **WHEN** a user sets `HYPERREADER_PORT` and does not pass an explicit port flag
- **THEN** HyperReader listens on the configured port

### Requirement: Legacy configuration cutoff

The system SHALL NOT use `HTML_MCP_DATA_DIR`, `HTML_MCP_PORT`, or the prior default `html-mcp` application data directory as runtime configuration sources.

#### Scenario: Legacy environment variables are set
- **WHEN** only `HTML_MCP_DATA_DIR` or `HTML_MCP_PORT` is set
- **THEN** HyperReader resolves its data directory and port without using those variables

#### Scenario: Existing documents require migration
- **WHEN** documents exist only in the prior `html-mcp` application data directory
- **THEN** HyperReader does not automatically read or modify that directory
- **AND** migration documentation instructs the user to move the documents before starting HyperReader
