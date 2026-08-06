## Why

HyperReader has no distribution method. The only documented install is `git clone` plus `go build`, which requires the Go toolchain and leaves the binary at an unpredictable path that every MCP client config has to hardcode. The README's `"command": "/absolute/path/to/hyperreader"` is the visible symptom.

`go install` is not a workaround today: `go.mod` declares `github.com/fmendonca/hyperreader`, but the repository is `github.com/fernandodeperto/HyperReader`, so the module path resolves to nothing.

Publishing binaries alone does not fix it either. A binary downloaded from a GitHub Releases page through a browser is tagged `com.apple.quarantine` and macOS refuses to run it. An install script fetched with `curl` is not tagged, so the script is what makes the release usable on the project's primary platform, not optional polish on top of it.

## What Changes

- **BREAKING (internal only)**: the Go module path changes from `github.com/fmendonca/hyperreader` to `github.com/fernandodeperto/hyperreader`, and the GitHub repository is renamed from `HyperReader` to lowercase `hyperreader`. 13 import lines across 8 files update. Nothing external imports this module, so there is no downstream impact.
- New `make release` target cross-compiles four targets (`darwin/arm64`, `darwin/amd64`, `linux/amd64`, `linux/arm64`) with `-trimpath -ldflags="-s -w"` into `dist/`, alongside a `SHA256SUMS` file.
- Release artifacts are **raw binaries, not archives**. No extraction step for consumers.
- New `install.sh` at the repository root: detects platform via `uname -sm`, downloads the matching binary through GitHub's `releases/latest/download/` redirect, verifies it against `SHA256SUMS`, installs to `~/.local/bin`, warns when that directory is not on `PATH`, and prints a ready-to-paste MCP client config containing the resolved absolute path.
- README gains an Install section covering the `curl | sh` one-liner and `go install`, and its MCP configuration example uses a concrete resolved path instead of a placeholder.
- No Windows target. No npm package. No Homebrew tap. No CI.

## Capabilities

### New Capabilities

- `binary-distribution`: how HyperReader is built for release, published, installed on a user machine, and how an installed binary is registered with an MCP client.

### Modified Capabilities

None. The command remains `hyperreader` with unchanged `serve` and `mcp` subcommands, unchanged MCP server identity, and unchanged `send_html` contract, so `project-identity` is unaffected. The Go module path is an implementation detail with no spec-level behavior.

## Impact

- `go.mod` module declaration, plus 13 import lines across 8 `.go` files.
- `Makefile`: new `release` and `dist-clean` targets. Existing `build` target is left unstripped so development panics keep their symbol names.
- New `install.sh` at the repository root.
- New `dist/` output directory, gitignored.
- `README.md`: new Install section, revised MCP configuration example, revised project layout listing.
- GitHub repository setting: rename to lowercase. GitHub serves a redirect from the old name, and the existing `origin` remote keeps working, but local remotes should be updated.
- Release publishing depends on the `gh` CLI, already present on the maintainer's machine.

## Non-Goals

- **`--version` flag and ldflags version injection.** Considered and deliberately excluded. Its main justification was detecting version skew between a long-lived `serve` and an independently updated `mcp`, and that risk was assessed as not applicable: both processes are developer-run, short-lived, and the ingest API is not expected to change. Revisit only if bug reports make build identification necessary.
- Windows support. `internal/config` falls back to `~/.local/share/hyperreader` on Windows, which is not the platform convention, and nothing in the repository has ever exercised it.
- npm distribution. Rejected: it requires publishing multiple platform packages atomically, it puts a resident Node process in the stdio path of the MCP server, and the public npm registry is unreachable from the maintainer's network.
- Homebrew tap, GoReleaser, and release CI. All are natural successors once the manual process proves itself, and none block a first release.
