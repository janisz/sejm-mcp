// Package main provides the CLI entry point for the sejm-mcp server that connects to Polish Parliament and Legal Information APIs.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/janisz/sejm-mcp/internal/server"
)

const (
	version = "1.0.0"
	appName = "sejm-mcp"
)

// validateAndSetMode validates that only one mode is specified and sets default mode if none is specified
func validateAndSetMode(sseMode, httpMode, stdioMode *bool) {
	modeCount := 0
	if *sseMode {
		modeCount++
	}
	if *httpMode {
		modeCount++
	}
	if *stdioMode {
		modeCount++
	}

	if modeCount > 1 {
		fmt.Fprintf(os.Stderr, "Error: Cannot specify multiple modes (-sse, -http, and -stdio are mutually exclusive)\n")
		os.Exit(1)
	}

	// Default to stdio mode if no mode specified
	if modeCount == 0 {
		*stdioMode = true
	}
}

func main() {
	var (
		showHelp    = flag.Bool("help", false, "Show help message")
		showVersion = flag.Bool("version", false, "Show version information")
		sseMode     = flag.Bool("sse", false, "Start SSE stream server mode (real-time with heartbeat)")
		httpMode    = flag.Bool("http", false, "Start HTTP server mode (stateless, easier for hosting/caching)")
		serverAddr  = flag.String("addr", ":8080", "Server address (used with -sse or -http)")
		stdioMode   = flag.Bool("stdio", false, "Use stdio mode (default)")
		debugMode   = flag.Bool("debug", false, "Enable debug logging")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n\n", appName)
		fmt.Fprintf(os.Stderr, "%s - Polish Parliament and Legal Acts MCP Server\n\n", appName)
		fmt.Fprintf(os.Stderr, "This server provides access to Polish Parliamentary data (Sejm) and legal acts (ELI)\n")
		fmt.Fprintf(os.Stderr, "through the Model Context Protocol (MCP) interface.\n\n")
		fmt.Fprintf(os.Stderr, "OPTIONS:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nMODES:\n")
		fmt.Fprintf(os.Stderr, "  Default mode is stdio for use with MCP clients\n")
		fmt.Fprintf(os.Stderr, "  SSE mode provides real-time streaming with heartbeat (best for development/testing)\n")
		fmt.Fprintf(os.Stderr, "  HTTP mode is stateless and easier for production hosting with load balancers/caching\n\n")
		fmt.Fprintf(os.Stderr, "EXAMPLES:\n")
		fmt.Fprintf(os.Stderr, "  %s                    # Start in stdio mode (default)\n", appName)
		fmt.Fprintf(os.Stderr, "  %s -stdio             # Explicit stdio mode\n", appName)
		fmt.Fprintf(os.Stderr, "  %s -sse               # Start SSE server on :8080\n", appName)
		fmt.Fprintf(os.Stderr, "  %s -http              # Start HTTP server on :8080\n", appName)
		fmt.Fprintf(os.Stderr, "  %s -sse -addr :9000   # Start SSE server on :9000\n", appName)
		fmt.Fprintf(os.Stderr, "  %s -debug             # Enable debug logging\n", appName)
		fmt.Fprintf(os.Stderr, "\nLOGGING:\n")
		fmt.Fprintf(os.Stderr, "  Logs are written to stderr in stdio, SSE, and HTTP modes\n")
		fmt.Fprintf(os.Stderr, "  Use -debug for detailed request/response logging\n\n")
	}

	flag.Parse()

	if *showHelp {
		flag.Usage()
		os.Exit(0)
	}

	if *showVersion {
		fmt.Printf("%s version %s\n", appName, version)
		os.Exit(0)
	}

	// Validate and set mode
	validateAndSetMode(sseMode, httpMode, stdioMode)

	// Create server with configuration
	config := server.Config{
		DebugMode: *debugMode,
	}

	sejmServer := server.NewSejmServerWithConfig(config)

	var err error
	if *sseMode {
		fmt.Fprintf(os.Stderr, "Starting %s SSE server on %s (debug=%v)\n", appName, *serverAddr, *debugMode)
		fmt.Fprintf(os.Stderr, "SSE mode provides real-time connection with heartbeat. Logs will be visible in this terminal. Use Ctrl+C to stop.\n")
		err = sejmServer.RunSSE(*serverAddr)
	} else if *httpMode {
		fmt.Fprintf(os.Stderr, "Starting %s HTTP server on %s (debug=%v)\n", appName, *serverAddr, *debugMode)
		fmt.Fprintf(os.Stderr, "HTTP mode is stateless and easier for hosting/caching. Logs will be visible in this terminal. Use Ctrl+C to stop.\n")
		err = sejmServer.RunHTTP(*serverAddr)
	} else {
		// stdio mode - don't print startup messages to stderr as it interferes with MCP protocol
		err = sejmServer.RunStdio()
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
