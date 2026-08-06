#!/bin/sh
# HyperReader installer.
#
#   curl -fsSL https://raw.githubusercontent.com/fernandodeperto/hyperreader/main/install.sh | sh
#
# Downloads the newest published release for this platform, verifies it
# against that release's SHA256SUMS manifest, and installs it to
# ~/.local/bin/hyperreader.
#
# Fetching with curl rather than a browser is deliberate: a browser download
# sets com.apple.quarantine and macOS then refuses to run the binary without
# a manual override. Go's linker ad-hoc signs darwin binaries, so no separate
# signing or notarization step is involved.
#
# Dependencies are limited to what a supported platform ships by default:
# curl, uname, and either sha256sum or shasum. No Go toolchain, no Node, no
# archive extractor, no JSON parser, no token.

set -eu

REPO="fernandodeperto/hyperreader"
BASE_URL="https://github.com/${REPO}/releases/latest/download"
BINARY="hyperreader"
INSTALL_DIR="${HOME}/.local/bin"
INSTALL_PATH="${INSTALL_DIR}/${BINARY}"
SUPPORTED="darwin-arm64, darwin-amd64, linux-amd64, linux-arm64"

die() {
	echo "install.sh: $*" >&2
	exit 1
}

# Platform detection runs before anything is downloaded and before the
# install directory is created, so an unsupported host leaves no state.
os=$(uname -s)
arch=$(uname -m)

case "$os" in
Darwin) goos=darwin ;;
Linux) goos=linux ;;
*) goos="" ;;
esac

case "$arch" in
arm64 | aarch64) goarch=arm64 ;;
x86_64 | amd64) goarch=amd64 ;;
*) goarch="" ;;
esac

if [ -z "$goos" ] || [ -z "$goarch" ]; then
	echo "install.sh: unsupported platform: ${os} ${arch}" >&2
	echo "Supported platforms: ${SUPPORTED}" >&2
	exit 1
fi

asset="${BINARY}-${goos}-${goarch}"

command -v curl >/dev/null 2>&1 || die "curl is required but was not found"

# Checksum tooling differs by platform: sha256sum on Linux, shasum on macOS.
if command -v sha256sum >/dev/null 2>&1; then
	sha256() { sha256sum "$1"; }
elif command -v shasum >/dev/null 2>&1; then
	sha256() { shasum -a 256 "$1"; }
else
	die "no SHA-256 tool found (looked for sha256sum and shasum)"
fi

tmpdir=$(mktemp -d)
# Cleared on every exit path, verified or not.
trap 'rm -rf "$tmpdir"' EXIT

# releases/latest/download redirects to the newest release's asset, which is
# what keeps this script free of a GitHub API call, a JSON parser, and a token.
echo "Downloading ${asset} from the latest release..."
curl -fsSL -o "${tmpdir}/${asset}" "${BASE_URL}/${asset}" ||
	die "failed to download ${BASE_URL}/${asset}"
curl -fsSL -o "${tmpdir}/SHA256SUMS" "${BASE_URL}/SHA256SUMS" ||
	die "failed to download ${BASE_URL}/SHA256SUMS"

# Only one artifact was downloaded, so `sha256sum -c SHA256SUMS` cannot be
# used: it would fail on the three absent files. Compare the one line that
# matters instead.
expected=$(awk -v name="$asset" '$2 == name { print $1 }' "${tmpdir}/SHA256SUMS")
[ -n "$expected" ] || die "no entry for ${asset} in the published SHA256SUMS"

actual=$(sha256 "${tmpdir}/${asset}" | cut -d' ' -f1)

if [ "$expected" != "$actual" ]; then
	echo "install.sh: checksum mismatch for ${asset}" >&2
	echo "  expected: ${expected}" >&2
	echo "  actual:   ${actual}" >&2
	echo "Nothing was installed." >&2
	exit 1
fi
echo "Checksum verified."

# Nothing above this line touches the install location.
mkdir -p "$INSTALL_DIR"
install -m 755 "${tmpdir}/${asset}" "$INSTALL_PATH" ||
	die "failed to install to ${INSTALL_PATH}"

echo "Installed ${BINARY} to ${INSTALL_PATH}"

# ~/.local/bin is not on the default PATH on macOS. Say so only when it is
# actually missing from the live PATH.
case ":${PATH}:" in
*":${INSTALL_DIR}:"*) ;;
*)
	echo
	echo "${INSTALL_DIR} is not on your PATH. Add it to your shell profile with:"
	echo
	echo "    export PATH=\"\$HOME/.local/bin:\$PATH\""
	;;
esac

# MCP clients launch the server by absolute path, so PATH is irrelevant to
# them. Print the entry fully resolved, with nothing left to substitute.
cat <<EOF

Add this to your MCP client configuration:

{
  "mcpServers": {
    "hyperreader": {
      "command": "${INSTALL_PATH}",
      "args": ["mcp"]
    }
  }
}

Start the reader with: ${INSTALL_PATH} serve
EOF
