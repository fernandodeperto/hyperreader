# binary-distribution Specification

## Purpose

Defines how HyperReader is built for release, published, and installed on an end-user machine, and how the resulting binary is registered with an MCP client without the user having to locate it themselves.

## Requirements

### Requirement: Release artifact set

A release SHALL publish one standalone executable per supported platform, plus a single checksum manifest covering all of them. Supported platforms SHALL be `darwin/arm64`, `darwin/amd64`, `linux/amd64`, and `linux/arm64`.

Artifacts SHALL be published as raw executables, not archives, so that installation requires no extraction step.

Each executable SHALL be named `hyperreader-<os>-<arch>` using the Go `GOOS` and `GOARCH` values for the platform. The checksum manifest SHALL be named `SHA256SUMS` and SHALL list a SHA-256 digest for every published executable.

Release executables SHALL be built with symbol and debug information stripped and with build paths trimmed. Development builds SHALL NOT be stripped, so that panic output retains symbol names.

#### Scenario: Maintainer produces release artifacts
- **WHEN** the maintainer runs the release build
- **THEN** an executable is produced for each of the four supported platforms, named `hyperreader-<os>-<arch>`
- **AND** a `SHA256SUMS` manifest is produced listing a digest for each of those executables
- **AND** each executable is stripped of symbol and debug information

#### Scenario: Development build is not stripped
- **WHEN** the maintainer runs the ordinary development build
- **THEN** the resulting binary retains symbol information for readable panic traces

### Requirement: Source installation via the canonical module path

The Go module path declared by the project SHALL match the canonical repository location, so that the published module resolves through the Go module proxy.

Installing from source using the Go toolchain SHALL produce a working `hyperreader` executable with no additional build steps.

#### Scenario: Developer installs from source
- **WHEN** a developer with a Go toolchain installs the module at its canonical path
- **THEN** the Go module proxy resolves the module
- **AND** a working `hyperreader` executable is placed in the developer's Go binary directory
- **AND** running it reports the usual `serve` and `mcp` subcommand usage

### Requirement: Scripted installation from the latest release

The project SHALL provide an install script, retrievable and executable in a single command, that installs the newest published release without the user selecting a version, platform, or URL.

The script SHALL detect the host operating system and architecture, download the matching executable from the latest release, and install it to `~/.local/bin/hyperreader`, creating that directory if it does not exist.

The script SHALL depend only on tooling present by default on a supported platform. It SHALL NOT require a Go toolchain, Node.js, an archive extractor, a JSON parser, or an authentication token.

Re-running the script SHALL replace any previously installed executable at that location and SHALL succeed.

#### Scenario: User installs on a supported platform
- **WHEN** a user runs the install script on a supported platform
- **THEN** the executable matching that platform is downloaded from the most recent release
- **AND** it is installed to `~/.local/bin/hyperreader` with the executable permission set
- **AND** the script reports where the executable was installed

#### Scenario: User reinstalls over an existing installation
- **GIVEN** `~/.local/bin/hyperreader` already exists from a previous run
- **WHEN** the user runs the install script again
- **THEN** the existing executable is replaced with the newest release
- **AND** the script completes successfully

### Requirement: Downloaded executable integrity verification

The install script SHALL verify the downloaded executable against the release's published `SHA256SUMS` manifest before installing it.

If the computed digest does not match the published digest, the script SHALL abort with a non-zero exit status and an error naming the mismatch, and SHALL NOT place any executable at the install location.

#### Scenario: Checksum matches
- **WHEN** the downloaded executable's digest matches the published manifest entry
- **THEN** installation proceeds

#### Scenario: Checksum does not match
- **WHEN** the downloaded executable's digest does not match the published manifest entry
- **THEN** the script reports the mismatch as an error
- **AND** exits with a non-zero status
- **AND** leaves no executable at the install location

### Requirement: Unsupported platform rejection

The install script SHALL detect a host platform outside the supported set and SHALL fail immediately with a message naming the detected platform and listing the supported platforms.

The script SHALL NOT download an executable, create the install directory, or leave partial state when the platform is unsupported.

#### Scenario: User runs the script on an unsupported platform
- **WHEN** the install script runs on a platform outside the supported set
- **THEN** it reports the detected operating system and architecture
- **AND** it names the supported platforms
- **AND** it exits with a non-zero status without downloading anything

### Requirement: PATH guidance after installation

The install directory is not on the default command search path on macOS. After a successful install, the script SHALL determine whether the install directory is present on the user's current `PATH`.

When the install directory is absent from `PATH`, the script SHALL say so and SHALL print the shell configuration line needed to add it. When it is already present, the script SHALL NOT print that guidance.

#### Scenario: Install directory is not on PATH
- **GIVEN** `~/.local/bin` is not on the user's `PATH`
- **WHEN** installation completes
- **THEN** the script reports that the directory is not on `PATH`
- **AND** prints the shell configuration line that adds it

#### Scenario: Install directory is already on PATH
- **GIVEN** `~/.local/bin` is already on the user's `PATH`
- **WHEN** installation completes
- **THEN** the script does not print PATH configuration guidance

### Requirement: MCP client registration guidance

An MCP client is configured with the absolute path of the executable to launch. After a successful install, the script SHALL print a ready-to-use MCP client configuration entry for HyperReader containing the fully resolved absolute path of the executable it just installed, with no placeholder for the user to substitute.

Project documentation SHALL likewise show a concrete resolved path in its MCP configuration example rather than a placeholder.

#### Scenario: Installer emits MCP configuration
- **WHEN** installation completes successfully
- **THEN** the script prints an MCP client configuration entry for HyperReader
- **AND** the configured command is the absolute path of the installed executable
- **AND** the configuration invokes the `mcp` subcommand
- **AND** the printed path contains no placeholder text

### Requirement: Installed executable runs without additional platform approval

An executable installed by the script SHALL be directly runnable on a supported platform with no further user action: no manual attribute removal, no security override, and no separate signing step.

On macOS this requires that the published executable carry a valid signature and that the install path does not mark the file as quarantined.

#### Scenario: User runs the installed executable on macOS
- **GIVEN** a user has installed HyperReader with the install script on macOS
- **WHEN** the user runs the installed executable
- **THEN** it starts normally
- **AND** the operating system does not block it or require the user to approve it
