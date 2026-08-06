## Context

See proposal.md for motivation. The constraints that shaped this design, all verified against the working tree and the maintainer's machine rather than assumed:

- **Cross-compilation is free.** Storage is `modernc.org/sqlite`, a pure-Go SQLite with no cgo. Building all four targets from darwin/arm64 took 39 seconds cold, with no failures.
- **`-trimpath -ldflags="-s -w"` cuts the binary from 18 MB to roughly 12 MB.** The current `make build` uses neither flag.
- **Go's linker ad-hoc signs darwin binaries.** `codesign -dv` on a cross-built `darwin/arm64` artifact reports `flags=0x20002(adhoc,linker-signed)`, `Signature=adhoc`. No signing identity, certificate, or notarization step is needed for the binary to be runnable.
- **`curl` does not quarantine.** A file fetched with `curl` carries only `com.apple.provenance`, not `com.apple.quarantine`. A browser download of the same file would carry the latter, and macOS would refuse to execute it.
- **Network reachability is asymmetric on the maintainer's machine.** `github.com`, `raw.githubusercontent.com`, and `proxy.golang.org` all respond. `registry.npmjs.org` fails TLS outright, and `npm` is pinned to a corporate Artifactory mirror.
- **`gh` 2.96.0 is installed. GoReleaser is not.**
- **The module path is wrong and cheap to fix.** `go.mod` declares `github.com/fmendonca/hyperreader`; the repository is `github.com/fernandodeperto/HyperReader`. 13 import lines across 8 files reference it.
- **There are no tags and no CI.** This is the first release of any kind.

## Goals / Non-Goals

**Goals:**

- One command a user can paste to get a working `hyperreader` on macOS or Linux.
- Two commands for the maintainer to cut a release, with no hand-assembled artifact list.
- No new tooling to install, learn, or keep configured.
- Remove the placeholder path from the README's MCP configuration example.

**Non-Goals:**

- Reproducible or third-party-verifiable builds. Artifacts come off the maintainer's laptop.
- Version pinning for end users. The install script always fetches the newest release.
- Automated release publishing. Everything here is run by hand, deliberately.

See proposal.md for product-level exclusions (Windows, npm, Homebrew, `--version`).

## Decisions

### Install script as the primary channel, not the Releases page

A GitHub Releases page alone is the only candidate channel that is broken on the project's primary platform. Downloading through a browser applies `com.apple.quarantine` and macOS blocks execution with "the developer cannot be verified", recoverable only by a manual `xattr -d` or a right-click override. Fetching the same bytes with `curl` does not apply the attribute.

The install script is therefore not convenience layered on top of the release. It is the mechanism that makes the release usable without code signing and notarization, which would require a paid Apple Developer account.

*Alternatives considered:* publishing archives and documenting the manual download; rejected because it makes the documented happy path the broken one. Signing and notarizing; rejected as disproportionate for a personal tool.

### Raw executables, not `.tar.gz`

Archiving would cut transfer from about 12 MB to about 5 MB per artifact, and costs an extraction step in every consumer, including the install script. For a tool of this size and audience the bandwidth is irrelevant and the removed step is worth more than the saving.

*Alternative considered:* `.tar.gz` per platform, the ecosystem convention. Rejected on simplicity grounds, which is the stated priority for this change.

### Install to `~/.local/bin`

`/usr/local/bin` requires `sudo`, and a `curl | sh` one-liner that escalates privileges is the most common reason people decline to run one. `~/.local/bin` needs no elevation and mirrors the convention the project already follows for its data directory (`~/.local/share/hyperreader`, per `internal/config`).

The cost is that `~/.local/bin` is not on the default `PATH` on macOS. This is handled explicitly rather than ignored: the script checks the live `PATH` after installing and prints the shell line to add the directory only when it is missing. This follows the pattern `rustup` uses.

*Alternative considered:* `/usr/local/bin` with `sudo`. Rejected. Note that the `PATH` gap does not affect MCP clients, which are configured with an absolute path.

### Fetch through the `releases/latest/download/` redirect

`https://github.com/<owner>/<repo>/releases/latest/download/<file>` serves a redirect to the corresponding asset of the newest release. Using it means the script needs no GitHub API call, no `jq`, no token, and no version-string parsing, which is what keeps its dependency list down to `curl`, `uname`, and a checksum tool.

*Alternative considered:* querying `/releases/latest` via the API and parsing `assets[].browser_download_url`. Rejected: it adds a JSON parser dependency and consumes unauthenticated rate limit for no gain.

### Checksum verification, with an honest scope

`SHA256SUMS` is published and the script verifies against it. This costs roughly five lines in total and catches truncated or corrupted downloads.

It should be understood for what it is. The manifest is served from the same origin as the binary, so it does not defend against a compromised release or account. It defends against transport corruption, and it means the script fails loudly rather than installing a broken executable.

Only one artifact is downloaded, so `sha256sum -c SHA256SUMS` cannot be used directly (it fails on the absent files). The script extracts the single relevant line and compares digests itself.

Checksum tooling differs by platform: `shasum -a 256` on macOS, `sha256sum` on Linux. The script must detect which is available rather than assume either.

### Rename the repository to lowercase, and fix the module path in the same change

Go module paths containing uppercase letters are escaped in the module proxy: `HyperReader` becomes `!hyper!reader`. It works, but it surfaces in proxy URLs and cache paths and looks like a defect.

The module path has to change regardless, because it currently names a repository that does not exist. Nothing imports this module externally, so the change is free right now and never gets cheaper. Doing the rename separately would mean touching all 13 import sites twice.

*Alternative considered:* keep `HyperReader` and set the module path to the escaped-equivalent mixed-case path. Rejected as a permanent papercut adopted to avoid a two-minute rename.

### Strip flags on release builds only

`-s -w` removes the symbol table and DWARF data, which is what makes panic traces readable. Release artifacts get the smaller size; `make build`, the development loop, keeps its symbols.

### Makefile plus `gh`, not GoReleaser

GoReleaser would handle cross-compilation, checksums, changelog generation, the GitHub release, and a Homebrew formula from a single config file, and it is the obvious answer at larger scale. It is not installed here and it is a second mental model to maintain.

Roughly fifteen lines of Makefile that the maintainer fully understands is simpler for a solo project with no CI. The correct moment to reconsider is when release automation moves into CI, at which point the manual process will have revealed its actual shape.

*Alternative considered:* adopting GoReleaser now. Deferred, not rejected on merit.

## Risks / Trade-offs

- **`curl | sh` asks for trust.** → Inherent to the format. Mitigated by keeping the script short and readable, serving it from the canonical repository, and documenting the two-step download-then-inspect-then-run alternative alongside the one-liner.
- **A hand-run release can be published incomplete**, for example binaries uploaded without `SHA256SUMS`, which makes every install fail verification. → The release target builds artifacts and the manifest together into `dist/`, and the publish step uploads the whole directory rather than a hand-typed file list.
- **Artifacts are built on one laptop and are not independently reproducible.** → Accepted for a personal tool. `-trimpath` at least removes local filesystem paths from the binaries.
- **`latest/download` gives users no way to pin a version.** → Accepted, and consistent with the decision that version skew is not a concern for this project. Per-version asset URLs remain available for anyone who needs one.
- **Renaming the GitHub repository invalidates the current `origin` URL.** → GitHub serves redirects for renamed repositories and existing remotes continue to work, but the maintainer's local remote should be updated explicitly rather than left relying on the redirect. Note the repository has two remotes, `origin` (GitHub) and `gitlab`; only the GitHub one is affected.
- **The install script cannot be tested until a release exists.** `latest/download` returns 404 with no published release. → Reflected in the task ordering: publish `v0.1.0` first, then verify the script end to end against the real release.
- **`~/.local/bin` may shadow, or be shadowed by, another `hyperreader`**, for example one previously placed by `go install` in `~/go/bin`. → The script reports the absolute install path on completion, and the printed MCP configuration uses that absolute path, so MCP behavior is unambiguous regardless of `PATH` ordering.

## Migration Plan

Ordering matters, because the identity change has to land before anything can be published under it, and the script cannot be verified before a release exists.

1. Rename the GitHub repository to lowercase `hyperreader`; update the local `origin` remote.
2. Rewrite the module path with `go mod edit -module` and update the 13 import sites; confirm the build and the existing test suite still pass.
3. Add the `release` target, `dist/` gitignore entry, and `install.sh`.
4. Tag `v0.1.0` and publish artifacts with `gh release create`.
5. Verify the install script end to end on a clean path, on both a `PATH`-configured and a non-`PATH`-configured shell.
6. Update the README last, so its documented commands are ones that have already been executed successfully.

No rollback is required. Nothing depends on the module path, and a bad release can be deleted or superseded by a new tag.
