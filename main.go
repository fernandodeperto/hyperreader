// Command hyperreader is the single binary behind HyperReader, the "always-open
// HTML reader for agent output". It exposes two subcommands:
//
//	hyperreader serve   long-lived HTTP server (storage + ingest/list/get API)
//	hyperreader mcp     stdio MCP server forwarding to the serve process
//
// Both subcommands share the same config resolution (internal/config): the
// XDG-style data directory and the listen port. This file wires the CLI
// flags to that config and dispatches to per-subcommand handlers.
//
// Both subcommands are fully wired: `serve` (internal/server) binds the
// port, opens storage, mounts the API router, and serves until interrupted;
// `mcp` (internal/mcp) runs a stdio MCP server that forwards send_html to a
// running serve process. mcp's stdout is reserved for JSON-RPC framing, so
// runMCP writes nothing to stdout — any stray byte would corrupt the
// protocol stream.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/fernandodeperto/hyperreader/internal/config"
	"github.com/fernandodeperto/hyperreader/internal/mcp"
	"github.com/fernandodeperto/hyperreader/internal/server"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "hyperreader:", err)
		os.Exit(1)
	}
}

// run dispatches on the first positional argument (the subcommand). It is
// split out from main so it can be tested/inspected without os.Exit.
func run(args []string) error {
	if len(args) == 0 {
		return usageError("")
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "serve":
		return runServe(rest)
	case "mcp":
		return runMCP(rest)
	case "-h", "--help", "help":
		printUsage(os.Stdout)
		return nil
	default:
		return usageError(sub)
	}
}

// runServe parses the serve subcommand flags, resolves config, and runs the
// real server (server.Run): it binds the port (already-running guard), opens
// the storage layer, mounts the API router, and serves until interrupted.
// SIGINT/SIGTERM trigger a graceful shutdown that drains in-flight requests.
func runServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	dataDir := fs.String("data-dir", "", "data directory (default: XDG data dir)")
	port := fs.Int("port", 0, fmt.Sprintf("listen port (default: %d, env %s)", config.DefaultPort, config.EnvPort))
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := config.Resolve(*dataDir, *port)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return server.Run(ctx, cfg)
}

// runMCP parses the mcp subcommand flags, resolves the target serve port,
// and runs the real stdio MCP server (mcp.Run): it forwards send_html calls
// to serve's HTTP ingest API and surfaces serve-down/HTTP-error/malformed
// responses as IsError=true tool results. SIGINT/SIGTERM cancel the run.
//
// runMCP MUST NOT write to stdout: stdout is the JSON-RPC transport for the
// MCP protocol, and any stray byte on it corrupts the framing for the
// client driving this process. Diagnostics, if any, belong on stderr.
func runMCP(args []string) error {
	port, err := parseMCPFlags(args)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return mcp.Run(ctx, port)
}

// parseMCPFlags parses the mcp subcommand's flags and resolves the listen
// port via config.ResolvePort (flag > HYPERREADER_PORT > DefaultPort). It is
// split out from runMCP so port resolution can be tested without invoking
// mcp.Run, which blocks reading stdin as the JSON-RPC transport.
func parseMCPFlags(args []string) (int, error) {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	port := fs.Int("port", 0, fmt.Sprintf("serve port to forward to (default: %d, env %s)", config.DefaultPort, config.EnvPort))
	if err := fs.Parse(args); err != nil {
		return 0, err
	}
	return config.ResolvePort(*port), nil
}

func printUsage(w *os.File) {
	fmt.Fprintln(w, "Usage: hyperreader <subcommand> [flags]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Subcommands:")
	fmt.Fprintln(w, "  serve   run the HTTP server (storage + ingest/list/get API)")
	fmt.Fprintln(w, "  mcp     run the stdio MCP server (forwards to serve)")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  serve --data-dir PATH   override the XDG data directory")
	fmt.Fprintf(w, "  serve --port N          override the listen port (env %s)\n", config.EnvPort)
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Environment:")
	fmt.Fprintf(w, "  %s   override the data directory\n", config.EnvDataDir)
	fmt.Fprintf(w, "  %s        override the listen port\n", config.EnvPort)
	fmt.Fprintf(w, "  %s   override the XDG data home base\n", config.EnvXDGDataHome)
}

func usageError(sub string) error {
	if sub == "" {
		return fmt.Errorf("no subcommand given; expected 'serve' or 'mcp'")
	}
	return fmt.Errorf("unknown subcommand %q; expected 'serve' or 'mcp'", sub)
}
