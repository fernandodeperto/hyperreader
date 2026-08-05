// Package config resolves runtime configuration for the html-mcp binary:
// the XDG-style data directory where the SQLite DB and HTML files live,
// and the HTTP port the serve process listens on.
//
// Both values honor a strict override priority:
//
//	data-dir: --data-dir flag  >  HTML_MCP_DATA_DIR env  >  XDG_DATA_HOME  >  ~/.local/share/html-mcp
//	port:     --port flag (>0) >  HTML_MCP_PORT env       >  DefaultPort
//
// The env/flag overrides are tested in config_test.go.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

const (
	// DefaultPort is the TCP port the serve process listens on when no
	// override is supplied. It is stable across restarts (R009).
	DefaultPort = 7420

	// AppDirName is the per-app directory created under the XDG data dir.
	AppDirName = "html-mcp"

	// EnvDataDir overrides the resolved data directory.
	EnvDataDir = "HTML_MCP_DATA_DIR"
	// EnvPort overrides the resolved port.
	EnvPort = "HTML_MCP_PORT"
	// EnvXDGDataHome is the standard XDG data-home variable, honored on
	// all platforms when set (macOS/Windows fall back to os.UserHomeDir).
	EnvXDGDataHome = "XDG_DATA_HOME"
)

// Config holds the fully-resolved runtime configuration.
type Config struct {
	// DataDir is the absolute path to the app data directory (contains
	// docs.db and the files/ subtree).
	DataDir string
	// Port is the TCP port the serve process listens on.
	Port int
}

// Config holds the fully-resolved runtime configuration.
//
// Resolve builds a Config by applying the override priority documented at
// package level. A non-empty dataDirFlag and/or a portFlag > 0 take
// precedence over the corresponding environment variables, which in turn
// take precedence over the defaults.
//
// Resolve never creates directories; callers that need the directory to
// exist (e.g. the serve process at startup) should call EnsureDataDir.
func Resolve(dataDirFlag string, portFlag int) (*Config, error) {
	dataDir, err := ResolveDataDir(dataDirFlag)
	if err != nil {
		return nil, err
	}
	return &Config{
		DataDir: dataDir,
		Port:    ResolvePort(portFlag),
	}, nil
}

// ResolveDataDir resolves the XDG-style data directory honoring overrides.
//
// Priority:
//  1. dataDirFlag (non-empty) — explicit --data-dir flag
//  2. HTML_MCP_DATA_DIR env — programmatic override
//  3. XDG_DATA_HOME/html-mcp — standard XDG, when the var is set
//  4. ~/.local/share/html-mcp — os.UserHomeDir fallback (macOS/Windows)
//
// The returned path is absolute and cleaned.
func ResolveDataDir(dataDirFlag string) (string, error) {
	// 1. explicit flag
	if dataDirFlag != "" {
		return absClean(dataDirFlag)
	}
	// 2. programmatic env override
	if v := os.Getenv(EnvDataDir); v != "" {
		return absClean(v)
	}
	// 3. standard XDG_DATA_HOME
	if v := os.Getenv(EnvXDGDataHome); v != "" {
		return absClean(filepath.Join(v, AppDirName))
	}
	// 4. os.UserHomeDir fallback
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve data dir: %w", err)
	}
	return absClean(filepath.Join(home, ".local", "share", AppDirName))
}

// ResolvePort resolves the listen port honoring overrides.
//
// Priority:
//  1. portFlag > 0 — explicit --port flag
//  2. HTML_MCP_PORT env — programmatic override (invalid value falls back to default)
//  3. DefaultPort
func ResolvePort(portFlag int) int {
	if portFlag > 0 {
		return portFlag
	}
	if v := os.Getenv(EnvPort); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			return p
		}
		// malformed env falls back to the default rather than failing;
		// a clear error is only useful when the user explicitly asked
		// for a value via the flag, which is handled above.
	}
	return DefaultPort
}

// EnsureDataDir creates the data directory (and parents) if it does not
// already exist, returning the absolute path. It is idempotent.
func EnsureDataDir(dataDir string) (string, error) {
	abs, err := absClean(dataDir)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return "", fmt.Errorf("create data dir %s: %w", abs, err)
	}
	return abs, nil
}

// DBPath returns the path to the SQLite database file inside the data dir.
func (c *Config) DBPath() string {
	return filepath.Join(c.DataDir, "docs.db")
}

// FilesDir returns the path to the HTML files subtree inside the data dir.
func (c *Config) FilesDir() string {
	return filepath.Join(c.DataDir, "files")
}

// absClean converts a possibly-relative path to an absolute, cleaned one.
func absClean(p string) (string, error) {
	a, err := filepath.Abs(p)
	if err != nil {
		return "", fmt.Errorf("resolve absolute path for %q: %w", p, err)
	}
	return filepath.Clean(a), nil
}
