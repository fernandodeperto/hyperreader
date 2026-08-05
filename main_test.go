package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/fmendonca/hyperreader/internal/config"
)

// setEnv sets (or unsets, for "") an env var and restores its prior value
// after the test, mirroring internal/config's test helper so port
// resolution priority (flag > env > default) is exercised the same way at
// the CLI layer as it is in internal/config.
func setEnv(t *testing.T, key, val string) {
	t.Helper()
	old, ok := os.LookupEnv(key)
	t.Cleanup(func() {
		if ok {
			_ = os.Setenv(key, old)
		} else {
			_ = os.Unsetenv(key)
		}
	})
	if val == "" {
		_ = os.Unsetenv(key)
	} else {
		_ = os.Setenv(key, val)
	}
}

// TestParseMCPFlags proves the mcp subcommand's flag parsing and port
// resolution (flag > HYPERREADER_PORT env > DefaultPort), and that a bad flag
// fails fast with an error rather than falling through to mcp.Run (which
// would block reading stdin as the JSON-RPC transport).
func TestParseMCPFlags(t *testing.T) {
	t.Run("flag wins over env", func(t *testing.T) {
		setEnv(t, config.EnvPort, "9999")
		got, err := parseMCPFlags([]string{"--port", "8000"})
		if err != nil {
			t.Fatalf("parseMCPFlags: unexpected error: %v", err)
		}
		if got != 8000 {
			t.Errorf("port = %d, want 8000 (flag beats env)", got)
		}
	})

	t.Run("env wins when no flag given", func(t *testing.T) {
		setEnv(t, config.EnvPort, "9001")
		got, err := parseMCPFlags(nil)
		if err != nil {
			t.Fatalf("parseMCPFlags: unexpected error: %v", err)
		}
		if got != 9001 {
			t.Errorf("port = %d, want 9001 from env", got)
		}
	})

	t.Run("falls back to DefaultPort", func(t *testing.T) {
		setEnv(t, config.EnvPort, "")
		got, err := parseMCPFlags(nil)
		if err != nil {
			t.Fatalf("parseMCPFlags: unexpected error: %v", err)
		}
		if got != config.DefaultPort {
			t.Errorf("port = %d, want DefaultPort %d", got, config.DefaultPort)
		}
	})

	t.Run("unknown flag fails fast with an error", func(t *testing.T) {
		_, err := parseMCPFlags([]string{"--bogus"})
		if err == nil {
			t.Fatal("parseMCPFlags: expected an error for an unrecognized flag, got nil")
		}
	})
}

// TestRunMCP_StdoutIsClean guards R005/the stdio-framing contract at the
// dispatch layer: runMCP must never write to stdout, since stdout carries
// JSON-RPC framing for the mcp subcommand and a stray byte corrupts the
// client's session. It exercises the flag-error path (which returns before
// mcp.Run would block on stdin) and asserts stdout stayed untouched.
func TestRunMCP_StdoutIsClean(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	origStdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = origStdout }()

	runErr := runMCP([]string{"--bogus"})

	_ = w.Close()
	os.Stdout = origStdout

	buf := make([]byte, 1)
	n, _ := r.Read(buf)
	_ = r.Close()

	if runErr == nil {
		t.Fatal("runMCP([--bogus]): expected an error, got nil")
	}
	if n != 0 {
		t.Fatalf("runMCP wrote %d byte(s) to stdout; stdout must stay JSON-RPC-pure", n)
	}
}

// TestRun_NoSubcommand proves the zero-args dispatch path: run([]) must
// fail with a usage error naming the two known subcommands, not panic on
// an out-of-range args[0] access.
func TestRun_NoSubcommand(t *testing.T) {
	err := run(nil)
	if err == nil {
		t.Fatal("run(nil): expected a usage error, got nil")
	}
	if !strings.Contains(err.Error(), "no subcommand") {
		t.Errorf("run(nil) error = %q, want it to mention 'no subcommand'", err.Error())
	}
}

// TestRun_UnknownSubcommand proves an unrecognized first argument fails
// fast with a message naming the offending subcommand, rather than falling
// through to serve/mcp or silently doing nothing.
func TestRun_UnknownSubcommand(t *testing.T) {
	err := run([]string{"frobnicate"})
	if err == nil {
		t.Fatal("run([frobnicate]): expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "frobnicate") {
		t.Errorf("run([frobnicate]) error = %q, want it to name the unknown subcommand", err.Error())
	}
	if !strings.Contains(err.Error(), "serve") || !strings.Contains(err.Error(), "mcp") {
		t.Errorf("run([frobnicate]) error = %q, want it to mention both known subcommands", err.Error())
	}
}

// TestRun_Help proves each of the three help spellings (-h, --help, help)
// dispatches to printUsage and returns nil, and that the usage text lands
// on stdout (a real user reading `hyperreader --help` expects it there, not
// on stderr where run's own top-level error path writes).
func TestRun_Help(t *testing.T) {
	for _, spelling := range []string{"-h", "--help", "help"} {
		t.Run(spelling, func(t *testing.T) {
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("os.Pipe: %v", err)
			}
			origStdout := os.Stdout
			os.Stdout = w

			runErr := run([]string{spelling})

			_ = w.Close()
			os.Stdout = origStdout

			var buf bytes.Buffer
			_, _ = buf.ReadFrom(r)
			_ = r.Close()

			if runErr != nil {
				t.Fatalf("run([%s]) = %v, want nil", spelling, runErr)
			}
			if !strings.Contains(buf.String(), "Usage: hyperreader") {
				t.Errorf("run([%s]) stdout = %q, want it to contain the usage banner", spelling, buf.String())
			}
		})
	}
}

// TestRun_ServeBadFlag proves the serve subcommand's own flag parsing
// fails fast on an unrecognized flag: run() must surface that error
// without ever reaching server.Run (which would bind a port and block
// forever), keeping run(["serve", ...]) safe to exercise in a unit test.
func TestRun_ServeBadFlag(t *testing.T) {
	err := run([]string{"serve", "--not-a-real-flag"})
	if err == nil {
		t.Fatal("run([serve --not-a-real-flag]): expected a flag-parse error, got nil")
	}
}

// TestRun_MCPBadFlag proves the mcp subcommand is reachable through run()'s
// dispatch (not just via a direct runMCP call) and that a bad flag there
// also fails fast without blocking on stdin (mcp.Run would block reading
// stdin as the JSON-RPC transport, which a bad flag must never reach).
func TestRun_MCPBadFlag(t *testing.T) {
	err := run([]string{"mcp", "--not-a-real-flag"})
	if err == nil {
		t.Fatal("run([mcp --not-a-real-flag]): expected a flag-parse error, got nil")
	}
}

// TestUsageError proves usageError's two message shapes: the no-args case
// names no subcommand and just states one is required, while the
// unknown-subcommand case echoes back exactly what was typed so a typo is
// obvious to the user.
func TestUsageError(t *testing.T) {
	if err := usageError(""); err == nil || !strings.Contains(err.Error(), "no subcommand") {
		t.Errorf("usageError(\"\") = %v, want an error mentioning 'no subcommand'", err)
	}
	if err := usageError("bogus"); err == nil || !strings.Contains(err.Error(), `"bogus"`) {
		t.Errorf("usageError(\"bogus\") = %v, want it to quote the unknown subcommand", err)
	}
}

// TestPrintUsage proves the usage banner documents both subcommands, the
// serve-specific flags, and the three config env vars — the surface a user
// running `hyperreader --help` actually needs, not just a banner that prints
// something.
func TestPrintUsage(t *testing.T) {
	var buf bytes.Buffer
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	printUsage(w)
	_ = w.Close()
	_, _ = buf.ReadFrom(r)
	_ = r.Close()

	out := buf.String()
	for _, want := range []string{
		"serve",
		"mcp",
		"--data-dir",
		"--port",
		config.EnvDataDir,
		config.EnvPort,
		config.EnvXDGDataHome,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("printUsage output missing %q; got:\n%s", want, out)
		}
	}
}
