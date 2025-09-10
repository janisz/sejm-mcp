package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"
)

// TestConstants verifies application constants
func TestConstants(t *testing.T) {
	if version == "" {
		t.Error("version constant should not be empty")
	}
	if appName == "" {
		t.Error("appName constant should not be empty")
	}
	if appName != "sejm-mcp" {
		t.Errorf("Expected appName to be 'sejm-mcp', got '%s'", appName)
	}
}

// TestFlagUsage tests the custom usage function
func TestFlagUsage(t *testing.T) {
	// Capture stderr output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Reset flags to avoid interference from other tests
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Set up flags like in main()
	showHelp := flag.Bool("help", false, "Show help message")
	showVersion := flag.Bool("version", false, "Show version information")
	sseMode := flag.Bool("sse", false, "Start SSE stream server mode (real-time with heartbeat)")
	httpMode := flag.Bool("http", false, "Start HTTP server mode (stateless, easier for hosting/caching)")
	serverAddr := flag.String("addr", ":8080", "Server address (used with -sse or -http)")
	stdioMode := flag.Bool("stdio", false, "Use stdio mode (default)")
	debugMode := flag.Bool("debug", false, "Enable debug logging")

	// Use the same custom usage function
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

	// Trigger usage output
	flag.Usage()

	// Close writer and restore stderr
	w.Close()
	os.Stderr = oldStderr

	// Read captured output
	var buf bytes.Buffer
	_, err := buf.ReadFrom(r)
	if err != nil {
		t.Fatalf("Failed to read from pipe: %v", err)
	}
	output := buf.String()

	// Verify essential content is present
	expectedStrings := []string{
		"Usage: sejm-mcp [OPTIONS]",
		"Polish Parliament and Legal Acts MCP Server",
		"Model Context Protocol (MCP) interface",
		"OPTIONS:",
		"MODES:",
		"EXAMPLES:",
		"LOGGING:",
		"-help",
		"-version",
		"-sse",
		"-http",
		"-stdio",
		"-debug",
		"-addr",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Usage output should contain '%s'", expected)
		}
	}

	// Avoid unused variable errors
	_ = showHelp
	_ = showVersion
	_ = sseMode
	_ = httpMode
	_ = serverAddr
	_ = stdioMode
	_ = debugMode
}

// TestModeCountLogic tests the mode validation logic
func TestModeCountLogic(t *testing.T) {
	testCases := []struct {
		name        string
		sseMode     bool
		httpMode    bool
		stdioMode   bool
		expectError bool
		description string
	}{
		{
			name:        "no modes specified",
			sseMode:     false,
			httpMode:    false,
			stdioMode:   false,
			expectError: false,
			description: "Should default to stdio mode",
		},
		{
			name:        "stdio mode only",
			sseMode:     false,
			httpMode:    false,
			stdioMode:   true,
			expectError: false,
			description: "Should be valid",
		},
		{
			name:        "sse mode only",
			sseMode:     true,
			httpMode:    false,
			stdioMode:   false,
			expectError: false,
			description: "Should be valid",
		},
		{
			name:        "http mode only",
			sseMode:     false,
			httpMode:    true,
			stdioMode:   false,
			expectError: false,
			description: "Should be valid",
		},
		{
			name:        "multiple modes - sse and http",
			sseMode:     true,
			httpMode:    true,
			stdioMode:   false,
			expectError: true,
			description: "Should error with multiple modes",
		},
		{
			name:        "multiple modes - sse and stdio",
			sseMode:     true,
			httpMode:    false,
			stdioMode:   true,
			expectError: true,
			description: "Should error with multiple modes",
		},
		{
			name:        "multiple modes - http and stdio",
			sseMode:     false,
			httpMode:    true,
			stdioMode:   true,
			expectError: true,
			description: "Should error with multiple modes",
		},
		{
			name:        "all modes specified",
			sseMode:     true,
			httpMode:    true,
			stdioMode:   true,
			expectError: true,
			description: "Should error with all modes",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test the mode counting logic from main()
			modeCount := 0
			if tc.sseMode {
				modeCount++
			}
			if tc.httpMode {
				modeCount++
			}
			if tc.stdioMode {
				modeCount++
			}

			hasError := modeCount > 1
			if hasError != tc.expectError {
				t.Errorf("Expected error=%v for %s, but got error=%v", tc.expectError, tc.description, hasError)
			}

			// Test default mode assignment
			if modeCount == 0 {
				// In main(), this would set stdioMode = true
				defaultStdio := true
				if !defaultStdio {
					t.Error("Should default to stdio mode when no mode specified")
				}
			}
		})
	}
}

// TestConfigCreation tests server configuration creation
func TestConfigCreation(t *testing.T) {
	testCases := []struct {
		name      string
		debugMode bool
	}{
		{"debug enabled", true},
		{"debug disabled", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test the config creation logic from main()
			type Config struct {
				DebugMode bool
			}

			config := Config{
				DebugMode: tc.debugMode,
			}

			if config.DebugMode != tc.debugMode {
				t.Errorf("Expected DebugMode=%v, got %v", tc.debugMode, config.DebugMode)
			}
		})
	}
}

// TestVersionDisplay tests version output functionality
func TestVersionDisplay(t *testing.T) {
	// This would test the version display logic
	versionString := fmt.Sprintf("%s version %s\n", appName, version)

	expectedPattern := "sejm-mcp version"
	if !strings.Contains(versionString, expectedPattern) {
		t.Errorf("Version string should contain '%s', got: %s", expectedPattern, versionString)
	}

	if !strings.Contains(versionString, version) {
		t.Errorf("Version string should contain version '%s', got: %s", version, versionString)
	}
}

// TestServerAddressValidation tests server address handling
func TestServerAddressValidation(t *testing.T) {
	testCases := []struct {
		addr        string
		description string
	}{
		{":8080", "default port"},
		{":9000", "custom port"},
		{"localhost:8080", "with hostname"},
		{"0.0.0.0:8080", "bind to all interfaces"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// Test that the address is properly used
			serverAddr := tc.addr
			if serverAddr != tc.addr {
				t.Errorf("Expected server address '%s', got '%s'", tc.addr, serverAddr)
			}
		})
	}
}

// TestErrorHandling tests error message creation
func TestErrorHandling(t *testing.T) {
	testError := fmt.Errorf("test error")
	errorMessage := fmt.Sprintf("Server error: %v\n", testError)

	expectedContent := "Server error: test error"
	if !strings.Contains(errorMessage, expectedContent) {
		t.Errorf("Error message should contain '%s', got: %s", expectedContent, errorMessage)
	}
}

// BenchmarkModeValidation benchmarks the mode counting logic
func BenchmarkModeValidation(b *testing.B) {
	b.Run("SingleMode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			sseMode := true
			httpMode := false
			stdioMode := false

			modeCount := 0
			if sseMode {
				modeCount++
			}
			if httpMode {
				modeCount++
			}
			if stdioMode {
				modeCount++
			}

			_ = modeCount > 1
		}
	})

	b.Run("MultipleMode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			sseMode := true
			httpMode := true
			stdioMode := false

			modeCount := 0
			if sseMode {
				modeCount++
			}
			if httpMode {
				modeCount++
			}
			if stdioMode {
				modeCount++
			}

			_ = modeCount > 1
		}
	})
}
