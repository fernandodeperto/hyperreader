## 1. Identity

- [x] 1.1 Rename the GitHub repository from `HyperReader` to lowercase `hyperreader` in repository settings; confirm the old URL redirects
- [ ] 1.2 Update the local `origin` remote to the new URL with `git remote set-url`; leave the `gitlab` remote untouched; confirm `git fetch origin` succeeds
- [x] 1.3 Rewrite the module path to `github.com/fernandodeperto/hyperreader` with `go mod edit -module`, and update the 13 import lines across the 8 `.go` files that reference the old path
- [x] 1.4 Confirm `go build ./...`, `go vet ./...`, `gofmt -l .`, and `go test ./...` all pass after the path change

## 2. Release build

- [x] 2.1 Add a `release` target to the Makefile that builds `hyperreader-<os>-<arch>` into `dist/` for `darwin/arm64`, `darwin/amd64`, `linux/amd64`, and `linux/arm64`, using `-trimpath -ldflags="-s -w"`
- [x] 2.2 Extend the `release` target to generate `dist/SHA256SUMS` covering all four executables, using the checksum tool available on the build host
- [x] 2.3 Add a `dist-clean` target that removes `dist/`, and wire `dist/` into the existing `clean` target
- [x] 2.4 Add `dist/` to `.gitignore`
- [x] 2.5 Leave the existing `build` target unstripped; add a comment recording that this is deliberate so panic traces keep symbol names
- [x] 2.6 Add the new targets to the Makefile's `help` output with `##` descriptions, matching the existing convention
- [x] 2.7 Run `make release` and confirm four executables plus `SHA256SUMS` appear in `dist/`, each executable roughly 12 MB
- [x] 2.8 Confirm `codesign -dv dist/hyperreader-darwin-arm64` reports an ad-hoc linker signature

## 3. Install script

- [x] 3.1 Create `install.sh` at the repository root that detects the platform via `uname -sm` and maps it to a `<os>-<arch>` artifact suffix
- [x] 3.2 Fail on an unsupported platform: report the detected OS and architecture, list the four supported platforms, exit non-zero, and download nothing
- [x] 3.3 Download the matching executable and `SHA256SUMS` from `https://github.com/fernandodeperto/hyperreader/releases/latest/download/` into a temporary directory
- [x] 3.4 Detect the available checksum tool (`shasum -a 256` on macOS, `sha256sum` on Linux), extract the single relevant line from `SHA256SUMS`, and compare digests; on mismatch, report it, exit non-zero, and leave nothing at the install location
- [x] 3.5 Create `~/.local/bin` if absent, install the verified executable there as `hyperreader` with the executable bit set, replacing any existing file, and clean up the temporary directory on both success and failure
- [x] 3.6 Report the absolute install path on success
- [x] 3.7 Check whether the install directory is on the live `PATH`; when absent, print the shell configuration line that adds it; when present, print nothing about `PATH`
- [x] 3.8 Print a ready-to-paste MCP client configuration entry whose `command` is the resolved absolute install path and whose `args` are `["mcp"]`, with no placeholder text
- [x] 3.9 Confirm the script passes `shellcheck` and runs under `sh`, not just `bash`

## 4. First release

- [x] 4.1 Tag `v0.1.0` and push the tag
- [x] 4.2 Publish the release with `gh release create v0.1.0 dist/*`, uploading the whole `dist/` directory rather than a hand-typed file list, so `SHA256SUMS` cannot be omitted
- [x] 4.3 Confirm `https://github.com/fernandodeperto/hyperreader/releases/latest/download/SHA256SUMS` resolves through the redirect

## 5. Verification

`git push` over SSH is broken on this machine (`Permission denied (publickey)`
for a key that is registered on the account), so the tag was created server-side
by `gh release create --target a1e65c3` rather than by `git push --tags`. The
resulting `refs/tags/v0.1.0` points at the same commit either route would have
produced.

5.1-5.4, 5.7 and 5.8 were run against the published v0.1.0 release. 5.5 needs a
corrupted manifest, which no real release can supply, so it used a local mirror
of the identical artifacts; only `BASE_URL` differed from the shipped script.
5.6 ran the shipped script unmodified with a stubbed `uname`.

- [x] 5.1 Run the install script end to end on macOS with no prior installation; confirm the binary lands in `~/.local/bin`, runs, and prints the expected usage
- [x] 5.2 Confirm the installed binary carries no `com.apple.quarantine` attribute and runs without a Gatekeeper prompt
- [x] 5.3 Re-run the install script over the existing installation and confirm it replaces the binary and exits zero
- [x] 5.4 Run the script in a shell where `~/.local/bin` is absent from `PATH` and confirm the guidance is printed; run it where the directory is on `PATH` and confirm it is not
- [x] 5.5 Corrupt a downloaded artifact or its manifest entry and confirm the script aborts non-zero with nothing installed
- [x] 5.6 Force an unsupported platform value and confirm the script fails before downloading
- [x] 5.7 Paste the emitted MCP configuration into a real MCP client, start `hyperreader serve`, and confirm a `send_html` call succeeds end to end
- [x] 5.8 Run `go install github.com/fernandodeperto/hyperreader@v0.1.0` and confirm the proxy resolves the module and the resulting binary runs

## 6. Documentation

- [x] 6.1 Add an Install section to the README covering the `curl | sh` one-liner, the download-inspect-run alternative for readers who will not pipe to a shell, and `go install`
- [x] 6.2 Replace `/absolute/path/to/hyperreader` in the README's MCP configuration example with a concrete resolved path, and note that the installer prints this entry
- [x] 6.3 Record in the README that supported platforms are macOS and Linux on x86-64 and arm64, and that Windows is unsupported
- [x] 6.4 Update the README's Requirements section: Go is now needed only for building from source, not for installing
- [x] 6.5 Add `install.sh` and `dist/` to the README's project layout listing
- [x] 6.6 Document the release procedure for the maintainer: `make release` then `gh release create`, with the tagging convention
- [x] 6.7 Confirm every command shown in the README's new sections was actually executed during task group 5
