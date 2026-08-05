package config

import (
	"os"
	"path/filepath"
	"testing"
)

// helper: snapshot and later restore a set of env vars.
func setEnvs(t *testing.T, kv map[string]string) {
	t.Helper()
	for k, v := range kv {
		old, ok := os.LookupEnv(k)
		t.Cleanup(func() {
			if ok {
				_ = os.Setenv(k, old)
			} else {
				_ = os.Unsetenv(k)
			}
		})
		if v == "" {
			_ = os.Unsetenv(k)
		} else {
			_ = os.Setenv(k, v)
		}
	}
}

func TestDefaultPort(t *testing.T) {
	if DefaultPort != 7420 {
		t.Errorf("DefaultPort = %d, want 7420", DefaultPort)
	}
}

func TestResolvePort_FlagWins(t *testing.T) {
	setEnvs(t, map[string]string{
		EnvPort:        "9999",
		EnvXDGDataHome: "",
	})
	got := ResolvePort(8000)
	if got != 8000 {
		t.Errorf("ResolvePort(8000) = %d, want 8000 (flag beats env)", got)
	}
}

func TestResolvePort_EnvWins(t *testing.T) {
	setEnvs(t, map[string]string{EnvPort: "9001"})
	got := ResolvePort(0)
	if got != 9001 {
		t.Errorf("ResolvePort(0) = %d, want 9001 from env", got)
	}
}

func TestResolvePort_Default(t *testing.T) {
	setEnvs(t, map[string]string{EnvPort: ""})
	got := ResolvePort(0)
	if got != DefaultPort {
		t.Errorf("ResolvePort(0) = %d, want DefaultPort %d", got, DefaultPort)
	}
}

func TestResolvePort_MalformedEnvFallsBack(t *testing.T) {
	setEnvs(t, map[string]string{EnvPort: "not-a-number"})
	got := ResolvePort(0)
	if got != DefaultPort {
		t.Errorf("ResolvePort(0) with malformed env = %d, want DefaultPort %d", got, DefaultPort)
	}
}

func TestResolvePort_NonPositiveEnvFallsBack(t *testing.T) {
	setEnvs(t, map[string]string{EnvPort: "0"})
	got := ResolvePort(0)
	if got != DefaultPort {
		t.Errorf("ResolvePort(0) with env=0 = %d, want DefaultPort %d", got, DefaultPort)
	}
}

func TestResolvePort_NonPositiveFlagIgnored(t *testing.T) {
	setEnvs(t, map[string]string{EnvPort: ""})
	// flag <= 0 is treated as "unset", so default applies
	got := ResolvePort(-1)
	if got != DefaultPort {
		t.Errorf("ResolvePort(-1) = %d, want DefaultPort %d", got, DefaultPort)
	}
}

func TestResolveDataDir_FlagWins(t *testing.T) {
	tmp := t.TempDir()
	setEnvs(t, map[string]string{
		EnvDataDir:     "/should/be/ignored",
		EnvXDGDataHome: "/also/ignored",
	})
	got, err := ResolveDataDir(tmp)
	if err != nil {
		t.Fatalf("ResolveDataDir(flag): %v", err)
	}
	want := filepath.Clean(tmp)
	if got != want {
		t.Errorf("ResolveDataDir(flag) = %q, want %q", got, want)
	}
}

func TestResolveDataDir_EnvOverrideWins(t *testing.T) {
	tmp := t.TempDir()
	setEnvs(t, map[string]string{
		EnvDataDir:     tmp,
		EnvXDGDataHome: "/should/be/ignored",
	})
	got, err := ResolveDataDir("")
	if err != nil {
		t.Fatalf("ResolveDataDir(env): %v", err)
	}
	if got != filepath.Clean(tmp) {
		t.Errorf("ResolveDataDir(env) = %q, want %q", got, tmp)
	}
}

func TestResolveDataDir_XDGDataHome(t *testing.T) {
	xdg := t.TempDir()
	setEnvs(t, map[string]string{
		EnvDataDir:     "",
		EnvXDGDataHome: xdg,
	})
	got, err := ResolveDataDir("")
	if err != nil {
		t.Fatalf("ResolveDataDir(xdg): %v", err)
	}
	want := filepath.Join(filepath.Clean(xdg), AppDirName)
	if got != want {
		t.Errorf("ResolveDataDir(xdg) = %q, want %q", got, want)
	}
}

func TestResolveDataDir_HomeFallback(t *testing.T) {
	setEnvs(t, map[string]string{
		EnvDataDir:     "",
		EnvXDGDataHome: "",
	})
	got, err := ResolveDataDir("")
	if err != nil {
		t.Fatalf("ResolveDataDir(home): %v", err)
	}
	// must be absolute, cleaned, and end with the app dir under .local/share
	if !filepath.IsAbs(got) {
		t.Errorf("ResolveDataDir(home) = %q, want absolute", got)
	}
	wantSuffix := filepath.Join(".local", "share", AppDirName)
	if filepath.Base(got) != AppDirName {
		t.Errorf("ResolveDataDir(home) base = %q, want %q", filepath.Base(got), AppDirName)
	}
	if !endsWith(got, wantSuffix) {
		t.Errorf("ResolveDataDir(home) = %q, want suffix %q", got, wantSuffix)
	}
}

func TestResolveDataDir_RelativeFlagAbsolutized(t *testing.T) {
	setEnvs(t, map[string]string{
		EnvDataDir:     "",
		EnvXDGDataHome: "",
	})
	got, err := ResolveDataDir("rel/path")
	if err != nil {
		t.Fatalf("ResolveDataDir(rel): %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("ResolveDataDir(rel) = %q, want absolute", got)
	}
	if filepath.Base(got) != "path" {
		t.Errorf("ResolveDataDir(rel) base = %q, want path", filepath.Base(got))
	}
}

func TestResolve(t *testing.T) {
	tmp := t.TempDir()
	setEnvs(t, map[string]string{
		EnvDataDir:     "",
		EnvXDGDataHome: "",
		EnvPort:        "",
	})
	cfg, err := Resolve(tmp, 8123)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if cfg.DataDir != filepath.Clean(tmp) {
		t.Errorf("cfg.DataDir = %q, want %q", cfg.DataDir, tmp)
	}
	if cfg.Port != 8123 {
		t.Errorf("cfg.Port = %d, want 8123", cfg.Port)
	}
	if cfg.DBPath() != filepath.Join(cfg.DataDir, "docs.db") {
		t.Errorf("DBPath = %q", cfg.DBPath())
	}
	if cfg.FilesDir() != filepath.Join(cfg.DataDir, "files") {
		t.Errorf("FilesDir = %q", cfg.FilesDir())
	}
}

func TestEnsureDataDir_CreatesDir(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "a", "b", "c")
	abs, err := EnsureDataDir(target)
	if err != nil {
		t.Fatalf("EnsureDataDir: %v", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("EnsureDataDir did not create a directory: %q", abs)
	}
}

func TestEnsureDataDir_Idempotent(t *testing.T) {
	target := t.TempDir()
	if _, err := EnsureDataDir(target); err != nil {
		t.Fatalf("first EnsureDataDir: %v", err)
	}
	if _, err := EnsureDataDir(target); err != nil {
		t.Fatalf("second EnsureDataDir: %v", err)
	}
}

func TestEnsureDataDir_RelativeAbsolutized(t *testing.T) {
	setEnvs(t, map[string]string{
		EnvDataDir:     "",
		EnvXDGDataHome: "",
	})
	abs, err := EnsureDataDir("rel-ensure")
	if err != nil {
		t.Fatalf("EnsureDataDir(rel): %v", err)
	}
	if !filepath.IsAbs(abs) {
		t.Errorf("EnsureDataDir(rel) = %q, want absolute", abs)
	}
	_ = os.RemoveAll(abs)
}

// TestAppDirName_IsHyperReader proves the default per-app directory name
// migrated from the legacy "html-mcp" to "hyperreader" per the clean
// cutover: a stray reference to the old name would silently resurrect the
// legacy data directory for XDG/home-fallback resolution.
func TestAppDirName_IsHyperReader(t *testing.T) {
	if AppDirName != "hyperreader" {
		t.Errorf("AppDirName = %q, want %q", AppDirName, "hyperreader")
	}
}

// TestEnvNames_AreHyperReaderNamespaced proves the override identifiers
// themselves moved to the HYPERREADER_* namespace, not just their
// resolution behavior — a docs/help-text reference to the old names would
// otherwise go undetected, since every other test exercises the symbols
// rather than their literal values.
func TestEnvNames_AreHyperReaderNamespaced(t *testing.T) {
	if EnvDataDir != "HYPERREADER_DATA_DIR" {
		t.Errorf("EnvDataDir = %q, want %q", EnvDataDir, "HYPERREADER_DATA_DIR")
	}
	if EnvPort != "HYPERREADER_PORT" {
		t.Errorf("EnvPort = %q, want %q", EnvPort, "HYPERREADER_PORT")
	}
}

// TestResolvePort_LegacyEnvIgnored proves the configuration-identity
// cutoff (spec: "Legacy configuration cutoff"): the prior HTML_MCP_PORT
// variable is not a recognized override. Even when it is the only
// port-related env var set, ResolvePort must fall back to DefaultPort
// rather than reading it.
func TestResolvePort_LegacyEnvIgnored(t *testing.T) {
	setEnvs(t, map[string]string{
		EnvPort:         "",
		"HTML_MCP_PORT": "9999",
	})
	got := ResolvePort(0)
	if got != DefaultPort {
		t.Errorf("ResolvePort(0) with only legacy HTML_MCP_PORT set = %d, want DefaultPort %d (legacy env must be ignored)", got, DefaultPort)
	}
}

// TestResolveDataDir_LegacyEnvIgnored proves the configuration-identity
// cutoff for the data directory: the prior HTML_MCP_DATA_DIR variable is
// not a recognized override. With only the legacy variable set, resolution
// must fall through to the next priority tier (XDG_DATA_HOME) rather than
// using the legacy path.
func TestResolveDataDir_LegacyEnvIgnored(t *testing.T) {
	xdg := t.TempDir()
	setEnvs(t, map[string]string{
		EnvDataDir:          "",
		"HTML_MCP_DATA_DIR": "/should/be/ignored/by/hyperreader",
		EnvXDGDataHome:      xdg,
	})
	got, err := ResolveDataDir("")
	if err != nil {
		t.Fatalf("ResolveDataDir: %v", err)
	}
	want := filepath.Join(filepath.Clean(xdg), AppDirName)
	if got != want {
		t.Errorf("ResolveDataDir() with only legacy HTML_MCP_DATA_DIR set = %q, want %q (legacy env must be ignored)", got, want)
	}
}

// TestResolveDataDir_LegacyDefaultDirNotUsed proves the home-fallback
// default directory itself moved off the legacy name: with every override
// unset, the resolved path's app-dir component must be "hyperreader", never
// the prior "html-mcp".
func TestResolveDataDir_LegacyDefaultDirNotUsed(t *testing.T) {
	setEnvs(t, map[string]string{
		EnvDataDir:     "",
		EnvXDGDataHome: "",
	})
	got, err := ResolveDataDir("")
	if err != nil {
		t.Fatalf("ResolveDataDir(home): %v", err)
	}
	if filepath.Base(got) == "html-mcp" {
		t.Fatalf("ResolveDataDir(home) = %q, resolved to the legacy html-mcp app dir", got)
	}
	if filepath.Base(got) != "hyperreader" {
		t.Errorf("ResolveDataDir(home) base = %q, want %q", filepath.Base(got), "hyperreader")
	}
}

// endsWith reports whether path ends with suffix using OS path semantics.
func endsWith(path, suffix string) bool {
	return filepath.Clean(path) == filepath.Join(filepath.Dir(filepath.Clean(path)), suffix) ||
		len(path) >= len(suffix) && path[len(path)-len(suffix):] == suffix
}
