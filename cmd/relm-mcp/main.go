// Command relm-mcp runs relm as a Model Context Protocol (MCP) server
// over stdio (stdin/stdout JSON-RPC), exposing database browsing and
// querying tools to AI agents.
//
// Usage:
//
//	relm-mcp [--read-only] [DSN]
//
// If a DSN is provided the server connects immediately; otherwise the
// AI agent must call the "connect" tool first.
package main

import (
	"flag"
	"fmt"
	"os"

	_ "github.com/agmonetti/relm/internal/store/cassandra"
	_ "github.com/agmonetti/relm/internal/store/mongo"
	_ "github.com/agmonetti/relm/internal/store/mssql"
	_ "github.com/agmonetti/relm/internal/store/mysql"
	_ "github.com/agmonetti/relm/internal/store/neo4j"
	_ "github.com/agmonetti/relm/internal/store/postgres"
	_ "github.com/agmonetti/relm/internal/store/redis"
	_ "github.com/agmonetti/relm/internal/store/sqlite"

	relmMCP "github.com/agmonetti/relm/internal/mcp"
)

func main() {
	readOnly := flag.Bool("read-only", false, "block all mutation queries (default: allow writes)")
	flag.Parse()

	if flag.NArg() > 1 {
		fmt.Fprintln(os.Stderr, "relm-mcp: too many arguments — usage: relm-mcp [--read-only] [DSN]")
		os.Exit(1)
	}

	srv := relmMCP.New(*readOnly)
	defer srv.Close()

	// If a DSN was provided, connect before starting the server.
	if flag.NArg() == 1 {
		if err := srv.ConnectDSN(flag.Arg(0)); err != nil {
			fmt.Fprintf(os.Stderr, "relm-mcp: %v\n", err)
			os.Exit(1)
		}
	}

	if err := srv.Serve(); err != nil {
		fmt.Fprintf(os.Stderr, "relm-mcp: %v\n", err)
		os.Exit(1)
	}
}
