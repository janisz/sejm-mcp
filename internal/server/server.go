package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/alexshin/httpcache"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/janisz/sejm-mcp/pkg/eli"
	"github.com/mark3labs/mcp-go/server"
)

// Config holds server configuration options
type Config struct {
	DebugMode bool
}

// PopularAct represents a frequently searched legal act
type PopularAct struct {
	Publisher   string
	Year        string
	Position    string
	Title       string
	Description string
}

// CacheEntry represents a cached item with expiration
type CacheEntry struct {
	Data      interface{}
	ExpiresAt time.Time
}

// HTTPCacheStats tracks cache performance metrics
type HTTPCacheStats struct {
	Hits        int64
	Misses      int64
	Requests    int64
	LastCleanup time.Time
}

// Cache holds cached reference data
type Cache struct {
	Publishers    *CacheEntry
	PopularActs   *CacheEntry
	StatusTypes   *CacheEntry
	DocumentTypes *CacheEntry
	Keywords      *CacheEntry
	Institutions  *CacheEntry
	HTTPStats     *HTTPCacheStats
	mu            sync.RWMutex
}

// SejmServer provides access to Polish Parliament and Legal Information System APIs through MCP protocol.
type SejmServer struct {
	server *server.MCPServer
	client *http.Client
	cache  *Cache
	logger *slog.Logger
	config Config
}


// LRUTTLCache implements httpcache.Cache using hashicorp's LRU with TTL
type LRUTTLCache struct {
	cache *expirable.LRU[string, []byte]
}

// NewLRUTTLCache creates a new LRU cache with TTL expiration
func NewLRUTTLCache(size int, ttl time.Duration) *LRUTTLCache {
	cache := expirable.NewLRU[string, []byte](size, nil, ttl)
	return &LRUTTLCache{
		cache: cache,
	}
}

// Get retrieves a cached response if it exists and hasn't expired
func (c *LRUTTLCache) Get(key string) ([]byte, bool) {
	return c.cache.Get(key)
}

// Set stores a response in the cache with TTL expiration
func (c *LRUTTLCache) Set(key string, data []byte) {
	c.cache.Add(key, data)
}

// Delete removes an entry from the cache
func (c *LRUTTLCache) Delete(key string) {
	c.cache.Remove(key)
}

// NewSejmServer creates a new instance of SejmServer with default configuration.
func NewSejmServer() *SejmServer {
	return NewSejmServerWithConfig(Config{DebugMode: false})
}

// NewSejmServerWithConfig creates a new instance of SejmServer with custom configuration.
func NewSejmServerWithConfig(config Config) *SejmServer {
	// Create base HTTP transport with improved connection handling
	baseTransport := &http.Transport{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false, // Enable keep-alives for better connection reuse
	}

	// Wrap with HTTP cache for automatic caching of all API responses
	// Use LRU cache with TTL that forces caching even when server sends no-cache headers
	cache := NewLRUTTLCache(1000, 60*time.Minute) // Cache 1000 entries for 1 hour
	cachedTransport := httpcache.NewConfigurableTransport(cache, &httpcache.CacheConfig{
		// Custom cache key function to ensure consistent keys
		CacheKeyFn: func(req *http.Request) string {
			return req.URL.String()
		},
		// Always authorize cache reading, ignoring server cache headers
		AuthorizeCacheFn: func(_ *http.Request, _ *http.Client) bool {
			return true
		},
	})
	cachedTransport.Transport = baseTransport

	// Create HTTP client with caching enabled
	client := &http.Client{
		Timeout:   45 * time.Second, // Increased timeout for stability
		Transport: cachedTransport,
	}

	// Configure log level based on debug mode
	logLevel := slog.LevelInfo
	if config.DebugMode {
		logLevel = slog.LevelDebug
	}

	// Initialize structured logger that writes to stderr for both stdio and HTTP modes
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: true,
	}))
	logger.Info("SEJM-MCP server starting up with enhanced structured logging enabled",
		slog.Bool("debugMode", config.DebugMode),
		slog.String("logLevel", logLevel.String()),
		slog.String("cacheType", "LRU with TTL"),
		slog.Int("cacheSize", 1000),
		slog.Duration("cacheTTL", 60*time.Minute))

	s := &SejmServer{
		client: client,
		cache: &Cache{
			HTTPStats: &HTTPCacheStats{
				LastCleanup: time.Now(),
			},
		},
		logger: logger,
		config: config,
	}

	mcpServer := server.NewMCPServer(
		"sejm-mcp",
		"1.0.0",
		server.WithLogging(),
	)

	s.server = mcpServer
	s.registerTools()

	return s
}

// RunStdio starts the server in stdio mode for MCP client communication.
func (s *SejmServer) RunStdio() error {
	s.logger.Debug("Starting server in stdio mode")
	return server.ServeStdio(s.server)
}

// RunSSE starts the server in SSE mode with real-time streaming capabilities.
func (s *SejmServer) RunSSE(addr string) error {
	s.logger.Info("Starting server in SSE mode", slog.String("address", addr))

	// Create SSE server using the MCP library
	sseServer := server.NewSSEServer(s.server,
		server.WithSSEEndpoint("/mcp"),
		server.WithMessageEndpoint("/mcp/message"),
		server.WithKeepAlive(true),
		server.WithKeepAliveInterval(10*time.Second))

	// Create a custom HTTP server that includes health check and uses the SSE server
	mux := http.NewServeMux()

	// Add health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		s.logger.Debug("Health check request received", slog.String("method", r.Method), slog.String("path", r.URL.Path))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"status":"healthy","service":"sejm-mcp","version":"1.0.0"}`)); err != nil {
			s.logger.Warn("Failed to write health check response", slog.Any("error", err))
		}
	})

	// Add root endpoint for health checking
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s.logger.Debug("Root endpoint request", slog.String("method", r.Method), slog.String("path", r.URL.Path))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		rootResponse := map[string]interface{}{
			"service": "sejm-mcp",
			"version": "1.0.0",
			"status":  "healthy",
			"mcp":     "/mcp",
		}
		if err := json.NewEncoder(w).Encode(rootResponse); err != nil {
			s.logger.Warn("Failed to encode root response", slog.Any("error", err))
		}
	})

	// Add MCP health check endpoint
	mux.HandleFunc("/mcp/health", func(w http.ResponseWriter, r *http.Request) {
		s.logger.Debug("MCP health check request received", slog.String("method", r.Method), slog.String("path", r.URL.Path))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		healthResponse := map[string]interface{}{
			"jsonrpc": "2.0",
			"result": map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]interface{}{
					"logging": map[string]interface{}{},
					"tools": map[string]interface{}{
						"listChanged": true,
					},
				},
				"serverInfo": map[string]interface{}{
					"name":    "sejm-mcp",
					"version": "1.0.0",
				},
			},
		}
		if err := json.NewEncoder(w).Encode(healthResponse); err != nil {
			s.logger.Warn("Failed to encode health response", slog.Any("error", err))
		}
	})

	// Mount the SSE server on the MCP endpoint
	mux.Handle("/mcp", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.logger.Info("MCP request received",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("userAgent", r.Header.Get("User-Agent")),
			slog.String("accept", r.Header.Get("Accept")))

		sseServer.ServeHTTP(w, r)
	}))

	// Mount the message handler for SSE
	mux.Handle("/mcp/message", sseServer.MessageHandler())

	// Create listener to get the actual assigned port
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		// If the specified port is busy, provide helpful error message
		if strings.Contains(err.Error(), "address already in use") {
			s.logger.Error("Port already in use",
				slog.String("address", addr),
				slog.String("suggestion", "Try a different port with -addr :8081 or kill existing processes"))
		}
		return fmt.Errorf("failed to create listener on %s: %w", addr, err)
	}

	// Get the actual address (important for random ports)
	actualAddr := listener.Addr().String()
	_, port, _ := net.SplitHostPort(actualAddr)
	s.logger.Info("HTTP server will be available with endpoints",
		slog.String("actualAddress", actualAddr),
		slog.String("health", "http://localhost:"+port+"/health"),
		slog.String("mcp", "http://localhost:"+port+"/mcp"),
		slog.String("mcpMessage", "http://localhost:"+port+"/mcp/message"))

	// Start the HTTP server with our custom mux and listener
	httpServer := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 30 * time.Second,
	}

	return httpServer.Serve(listener)
}

// RunHTTP starts the server in stateless HTTP mode for production deployment.
func (s *SejmServer) RunHTTP(addr string) error {
	s.logger.Info("Starting server in HTTP mode", slog.String("address", addr))

	// Create StreamableHTTPServer for stateless operation
	httpServer := server.NewStreamableHTTPServer(s.server,
		server.WithEndpointPath("/mcp"),
		server.WithStateLess(true),
		server.WithHeartbeatInterval(30*time.Second))

	// Create a custom HTTP server that includes health check and uses the HTTP server
	mux := http.NewServeMux()

	// Add health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		s.logger.Debug("Health check request received", slog.String("method", r.Method), slog.String("path", r.URL.Path))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"status":"healthy","service":"sejm-mcp","version":"1.0.0"}`)); err != nil {
			s.logger.Warn("Failed to write health check response", slog.Any("error", err))
		}
	})

	// Add root endpoint for health checking
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s.logger.Debug("Root endpoint request", slog.String("method", r.Method), slog.String("path", r.URL.Path))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		rootResponse := map[string]interface{}{
			"service": "sejm-mcp",
			"version": "1.0.0",
			"status":  "healthy",
			"mcp":     "/mcp",
		}
		if err := json.NewEncoder(w).Encode(rootResponse); err != nil {
			s.logger.Warn("Failed to encode root response", slog.Any("error", err))
		}
	})

	// Add MCP health check endpoint
	mux.HandleFunc("/mcp/health", func(w http.ResponseWriter, r *http.Request) {
		s.logger.Debug("MCP health check request received", slog.String("method", r.Method), slog.String("path", r.URL.Path))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		healthResponse := map[string]interface{}{
			"jsonrpc": "2.0",
			"result": map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]interface{}{
					"logging": map[string]interface{}{},
					"tools": map[string]interface{}{
						"listChanged": true,
					},
				},
				"serverInfo": map[string]interface{}{
					"name":    "sejm-mcp",
					"version": "1.0.0",
				},
			},
		}
		if err := json.NewEncoder(w).Encode(healthResponse); err != nil {
			s.logger.Warn("Failed to encode health response", slog.Any("error", err))
		}
	})

	// Mount the HTTP server on the MCP endpoint
	mux.Handle("/mcp", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.logger.Info("MCP HTTP request received",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("userAgent", r.Header.Get("User-Agent")),
			slog.String("contentType", r.Header.Get("Content-Type")))

		httpServer.ServeHTTP(w, r)
	}))

	// Create listener to get the actual assigned port
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		// If the specified port is busy, provide helpful error message
		if strings.Contains(err.Error(), "address already in use") {
			s.logger.Error("Port already in use",
				slog.String("address", addr),
				slog.String("suggestion", "Try a different port with -addr :8081 or kill existing processes"))
		}
		return fmt.Errorf("failed to create listener on %s: %w", addr, err)
	}

	// Get the actual address (important for random ports)
	actualAddr := listener.Addr().String()
	_, port, _ := net.SplitHostPort(actualAddr)
	s.logger.Info("HTTP server will be available with endpoints",
		slog.String("actualAddress", actualAddr),
		slog.String("health", "http://localhost:"+port+"/health"),
		slog.String("mcp", "http://localhost:"+port+"/mcp"))

	// Start the HTTP server with our custom mux and listener
	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 30 * time.Second,
	}

	return srv.Serve(listener)
}

func (s *SejmServer) registerTools() {
	s.registerSejmTools()
	s.registerELITools()
}

func (s *SejmServer) makeAPIRequest(ctx context.Context, endpoint string, params map[string]string) ([]byte, error) {
	return s.makeAPIRequestWithHeaders(ctx, endpoint, params, map[string]string{"Accept": "application/json"})
}

func (s *SejmServer) makeTextRequest(ctx context.Context, endpoint string, format string) ([]byte, error) {
	var acceptHeader string
	if format == "pdf" {
		acceptHeader = "application/pdf"
	} else {
		acceptHeader = "text/html"
	}
	return s.makeAPIRequestWithHeaders(ctx, endpoint, nil, map[string]string{"Accept": acceptHeader})
}

func (s *SejmServer) makeAPIRequestWithHeaders(ctx context.Context, endpoint string, params map[string]string, headers map[string]string) ([]byte, error) {
	reqURL, err := url.Parse(endpoint)
	if err != nil {
		s.logger.Error("Invalid URL parsing failed",
			slog.String("endpoint", endpoint),
			slog.Any("error", err))
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if params != nil {
		q := reqURL.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		reqURL.RawQuery = q.Encode()
	}

	finalURL := reqURL.String()
	s.logger.Info("Starting API request",
		slog.String("url", finalURL),
		slog.Any("headers", headers),
		slog.Any("params", params))

	// Log request headers
	for k, v := range headers {
		s.logger.Debug("Request header", slog.String("key", k), slog.String("value", v))
	}

	// Retry logic for connection stability
	maxRetries := 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s
			backoffDuration := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			s.logger.Warn("Retrying request",
				slog.Int("attempt", attempt+1),
				slog.Int("maxRetries", maxRetries),
				slog.Duration("backoff", backoffDuration))
			select {
			case <-ctx.Done():
				s.logger.Error("Request cancelled by context", slog.Any("error", ctx.Err()))
				return nil, ctx.Err()
			case <-time.After(backoffDuration):
			}
		}

		s.logger.Debug("Creating HTTP request",
			slog.Int("attempt", attempt+1),
			slog.Int("maxRetries", maxRetries))
		req, err := http.NewRequestWithContext(ctx, "GET", reqURL.String(), nil)
		if err != nil {
			s.logger.Error("Failed to create HTTP request", slog.Any("error", err))
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		for k, v := range headers {
			req.Header.Set(k, v)
		}

		s.logger.Debug("Executing HTTP request", slog.String("url", finalURL))
		start := time.Now()
		resp, err := s.client.Do(req)
		duration := time.Since(start)

		if err != nil {
			s.logger.Error("HTTP request failed",
				slog.Int("attempt", attempt+1),
				slog.Int("maxRetries", maxRetries),
				slog.Duration("duration", duration),
				slog.Any("error", err))
			lastErr = err
			// Check if this is a network error that might benefit from retry
			if attempt < maxRetries-1 {
				continue
			}
			return nil, fmt.Errorf("failed to make request after %d attempts: %w", maxRetries, err)
		}

		s.logger.Info("HTTP request completed",
			slog.Int("attempt", attempt+1),
			slog.Int("maxRetries", maxRetries),
			slog.Duration("duration", duration),
			slog.Int("status", resp.StatusCode))

		// Handle HTTP status errors
		if resp.StatusCode != http.StatusOK {
			s.logger.Warn("HTTP request returned non-200 status",
				slog.Int("status", resp.StatusCode),
				slog.String("statusText", resp.Status),
				slog.String("url", finalURL))
			if err := resp.Body.Close(); err != nil {
				s.logger.Warn("Failed to close response body", slog.Any("error", err))
			}
			// Enhanced error messages with specific status codes
			switch resp.StatusCode {
			case http.StatusNotFound:
				s.logger.Error("Resource not found", slog.String("url", finalURL))
				return nil, fmt.Errorf("resource not found (404) - the requested document or endpoint does not exist")
			case http.StatusForbidden:
				s.logger.Error("Access denied", slog.String("url", finalURL))
				return nil, fmt.Errorf("access denied (403) - this may indicate: format not available, API access restrictions, or invalid parameters")
			case http.StatusTooManyRequests:
				s.logger.Warn("Rate limit exceeded",
					slog.String("url", finalURL),
					slog.Int("attempt", attempt+1),
					slog.Int("maxRetries", maxRetries))
				// Rate limit - retry this one
				if attempt < maxRetries-1 {
					continue
				}
				return nil, fmt.Errorf("rate limit exceeded (429) - please wait before making additional requests")
			case http.StatusInternalServerError:
				s.logger.Warn("Server error",
					slog.String("url", finalURL),
					slog.Int("attempt", attempt+1),
					slog.Int("maxRetries", maxRetries))
				// Server error - retry this one
				if attempt < maxRetries-1 {
					continue
				}
				return nil, fmt.Errorf("server error (500) - the API service is experiencing technical difficulties")
			case http.StatusBadRequest:
				s.logger.Error("Bad request", slog.String("url", finalURL))
				return nil, fmt.Errorf("bad request (400) - invalid parameters or malformed request")
			case http.StatusUnauthorized:
				s.logger.Error("Unauthorized", slog.String("url", finalURL))
				return nil, fmt.Errorf("unauthorized (401) - authentication required or invalid credentials")
			default:
				s.logger.Error("Unexpected HTTP status",
					slog.Int("status", resp.StatusCode),
					slog.String("url", finalURL))
				return nil, fmt.Errorf("API request failed with status %d - unexpected error occurred", resp.StatusCode)
			}
		}

		// Success! Process the response
		defer func() {
			if err := resp.Body.Close(); err != nil {
				s.logger.Warn("Failed to close response body", slog.Any("error", err))
			}
		}()

		// Update cache statistics
		s.updateHTTPCacheStats(resp)

		// Log cache status
		cacheStatus := "MISS"
		if resp.Header.Get("X-From-Cache") == "1" {
			cacheStatus = "HIT"
		}

		s.logger.Info("Processing successful response",
			slog.Int64("contentLength", resp.ContentLength),
			slog.String("contentType", resp.Header.Get("Content-Type")),
			slog.String("cacheStatus", cacheStatus))

		// For JSON responses (when Accept header is application/json)
		if acceptType := headers["Accept"]; acceptType == "application/json" {
			s.logger.Debug("Decoding JSON response")
			var result json.RawMessage
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				s.logger.Error("Failed to decode JSON response", slog.Any("error", err))
				return nil, fmt.Errorf("failed to decode response: %w", err)
			}
			s.logger.Info("Successfully decoded JSON response", slog.Int("bytes", len(result)))
			return result, nil
		}

		// For text/HTML/PDF responses, read raw body
		s.logger.Debug("Reading raw response body", slog.Int64("expectedLength", resp.ContentLength))

		// Handle unknown content length (-1) by starting with empty slice
		var body []byte
		if resp.ContentLength > 0 {
			// Pre-allocate if we know the size
			body = make([]byte, 0, resp.ContentLength)
		} else {
			// Start with empty slice for unknown size
			body = make([]byte, 0)
		}

		buf := make([]byte, 4096)
		totalRead := 0
		for {
			n, err := resp.Body.Read(buf)
			if n > 0 {
				body = append(body, buf[:n]...)
				totalRead += n
			}
			if err != nil {
				if err.Error() == "EOF" {
					s.logger.Info("Successfully read response body", slog.Int("bytes", totalRead))
					break
				}
				s.logger.Error("Failed to read response body",
					slog.Int("bytesRead", totalRead),
					slog.Any("error", err))
				return nil, fmt.Errorf("failed to read response body: %w", err)
			}
		}

		return body, nil
	}

	// This should never be reached but keep for safety
	return nil, fmt.Errorf("all retry attempts failed: %w", lastErr)
}

func (s *SejmServer) validateTerm(termStr string) (int, error) {
	if termStr == "" {
		return 10, nil // Default to current term
	}

	term, err := strconv.Atoi(termStr)
	if err != nil {
		return 0, fmt.Errorf("invalid term: must be a number")
	}

	if term < 1 || term > 10 {
		return 0, fmt.Errorf("invalid term: must be between 1 and 10")
	}

	return term, nil
}

// getCachedPublishers returns publishers from cache or fetches them
func (s *SejmServer) getCachedPublishers(ctx context.Context) ([]eli.PublishingHouse, error) {
	s.cache.mu.RLock()
	if s.cache.Publishers != nil && time.Now().Before(s.cache.Publishers.ExpiresAt) {
		publishers := s.cache.Publishers.Data.([]eli.PublishingHouse)
		s.cache.mu.RUnlock()
		return publishers, nil
	}
	s.cache.mu.RUnlock()

	// Cache miss or expired, fetch fresh data
	s.cache.mu.Lock()
	defer s.cache.mu.Unlock()

	// Double-check in case another goroutine updated while we waited for lock
	if s.cache.Publishers != nil && time.Now().Before(s.cache.Publishers.ExpiresAt) {
		return s.cache.Publishers.Data.([]eli.PublishingHouse), nil
	}

	// Fetch publishers from API
	endpoint := "https://api.sejm.gov.pl/eli/acts"
	data, err := s.makeAPIRequest(ctx, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch publishers: %w", err)
	}

	var publishers []eli.PublishingHouse
	if err := json.Unmarshal(data, &publishers); err != nil {
		return nil, fmt.Errorf("failed to parse publishers: %w", err)
	}

	// Cache for 24 hours
	s.cache.Publishers = &CacheEntry{
		Data:      publishers,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	return publishers, nil
}

// validatePublisher checks if a publisher code is valid and suggests alternatives
func (s *SejmServer) validatePublisher(ctx context.Context, publisherCode string) (bool, []string, error) {
	if publisherCode == "" {
		return true, nil, nil // Empty is valid (optional parameter)
	}

	publishers, err := s.getCachedPublishers(ctx)
	if err != nil {
		return true, nil, err // If we can't validate, assume it's valid
	}

	// Check if publisher exists
	for _, pub := range publishers {
		if pub.Code != nil && *pub.Code == publisherCode {
			return true, nil, nil
		}
	}

	// Publisher not found, suggest alternatives
	var suggestions []string
	suggestions = append(suggestions, "Valid publisher codes:")

	// Show major publishers first
	majorPublishers := []string{"DU", "MP"}
	for _, major := range majorPublishers {
		for _, pub := range publishers {
			if pub.Code != nil && *pub.Code == major {
				name := "Unknown"
				if pub.Name != nil {
					name = *pub.Name
				}
				suggestions = append(suggestions, fmt.Sprintf("• %s: %s", major, name))
			}
		}
	}

	// Add a few more common ones
	count := 0
	for _, pub := range publishers {
		if pub.Code != nil {
			code := *pub.Code
			if code != "DU" && code != "MP" && count < 3 {
				name := "Unknown"
				if pub.Name != nil {
					name = *pub.Name
				}
				suggestions = append(suggestions, fmt.Sprintf("• %s: %s", code, name))
				count++
			}
		}
	}

	return false, suggestions, nil
}

// getPopularActs returns a curated list of frequently searched legal acts
func (s *SejmServer) getPopularActs() []PopularAct {
	s.cache.mu.RLock()
	if s.cache.PopularActs != nil && time.Now().Before(s.cache.PopularActs.ExpiresAt) {
		acts := s.cache.PopularActs.Data.([]PopularAct)
		s.cache.mu.RUnlock()
		return acts
	}
	s.cache.mu.RUnlock()

	s.cache.mu.Lock()
	defer s.cache.mu.Unlock()

	// Double-check
	if s.cache.PopularActs != nil && time.Now().Before(s.cache.PopularActs.ExpiresAt) {
		return s.cache.PopularActs.Data.([]PopularAct)
	}

	// Create popular acts list (this could be loaded from config in the future)
	popularActs := []PopularAct{
		{
			Publisher:   "DU",
			Year:        "1997",
			Position:    "483",
			Title:       "konstytucja",
			Description: "Constitution of the Republic of Poland",
		},
		{
			Publisher:   "DU",
			Year:        "1964",
			Position:    "93",
			Title:       "kodeks cywilny",
			Description: "Civil Code",
		},
		{
			Publisher:   "DU",
			Year:        "1997",
			Position:    "483",
			Title:       "kodeks karny",
			Description: "Criminal Code",
		},
		{
			Publisher:   "DU",
			Year:        "1974",
			Position:    "24",
			Title:       "kodeks pracy",
			Description: "Labor Code",
		},
		{
			Publisher:   "DU",
			Year:        "2018",
			Position:    "1000",
			Title:       "ochrona danych",
			Description: "GDPR Implementation Law",
		},
	}

	// Cache for 24 hours (static data)
	s.cache.PopularActs = &CacheEntry{
		Data:      popularActs,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	return popularActs
}

// getSearchSuggestions returns intelligent search suggestions based on popular acts and cached data
func (s *SejmServer) getSearchSuggestions(searchTitle string) []string {
	var suggestions []string

	popularActs := s.getPopularActs()

	// Add title-specific suggestions
	if searchTitle != "" {
		titleLower := strings.ToLower(searchTitle)
		for _, act := range popularActs {
			if strings.Contains(strings.ToLower(act.Title), titleLower) || strings.Contains(strings.ToLower(act.Description), titleLower) {
				suggestions = append(suggestions, fmt.Sprintf("Try: eli_search_acts with title='%s' (%s)", act.Title, act.Description))
			}
		}

		// Add document type suggestions if search seems to match a type
		documentTypes := s.getCachedDocumentTypes()
		for _, docType := range documentTypes {
			if strings.Contains(strings.ToLower(docType), titleLower) || strings.Contains(titleLower, strings.ToLower(docType)) {
				suggestions = append(suggestions, fmt.Sprintf("Try: eli_search_acts with type='%s' to find all %s documents", docType, strings.ToLower(docType)))
				break // Only suggest one type match
			}
		}
	}

	// Add general popular searches
	if len(suggestions) < 3 {
		suggestions = append(suggestions, "Popular searches:")
		for i, act := range popularActs {
			if i >= 3 { // Limit to top 3
				break
			}
			suggestions = append(suggestions, fmt.Sprintf("• eli_search_acts with title='%s' (%s)", act.Title, act.Description))
		}
	}

	return suggestions
}




// getCachedDocumentTypes returns document types from cache or builds them from legal system knowledge
func (s *SejmServer) getCachedDocumentTypes() []string {
	s.cache.mu.RLock()
	if s.cache.DocumentTypes != nil && time.Now().Before(s.cache.DocumentTypes.ExpiresAt) {
		docTypes := s.cache.DocumentTypes.Data.([]string)
		s.cache.mu.RUnlock()
		return docTypes
	}
	s.cache.mu.RUnlock()

	s.cache.mu.Lock()
	defer s.cache.mu.Unlock()

	// Double-check
	if s.cache.DocumentTypes != nil && time.Now().Before(s.cache.DocumentTypes.ExpiresAt) {
		return s.cache.DocumentTypes.Data.([]string)
	}

	// Use complete document types from ELI API (matches eli_tools.go eliDocumentTypes)
	documentTypes := []string{
		"Oświadczenie", "Umowa zbiorowa", "Lista", "Konwencja", "Komunikat", "Układ",
		"Orędzie", "Zalecenie", "Dokument wypowiedzenia", "Umowa", "Wykaz",
		"Oświadczenie rządowe", "Statut", "Ustawa", "Raport", "Apel", "Sprostowanie",
		"Pismo okólne", "Okólnik", "Porozumienie", "Obwieszczenie", "Reskrypt",
		"Przepisy", "Dekret", "Traktat", "Rozkaz", "Instrukcja", "Sprawozdanie",
		"Opinia", "Umowa międzynarodowa", "Wyjaśnienie", "Wytyczne", "Decyzja",
		"Wypis", "Stanowisko", "Przepisy wykonawcze", "Rezolucja", "Rozporządzenie",
		"Karta", "Zawiadomienie", "Akt", "Uchwała", "Orzeczenie", "Ogłoszenie",
		"Deklaracja", "Regulamin", "Protokół", "Zarządzenie", "Informacja",
		"Postanowienie", "Interpretacja",
	}

	// Cache for 30 days (document types in legal system change very rarely)
	s.cache.DocumentTypes = &CacheEntry{
		Data:      documentTypes,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}

	return documentTypes
}

// getCachedKeywords returns frequently used legal keywords from cache or builds them
func (s *SejmServer) getCachedKeywords() []string {
	s.cache.mu.RLock()
	if s.cache.Keywords != nil && time.Now().Before(s.cache.Keywords.ExpiresAt) {
		keywords := s.cache.Keywords.Data.([]string)
		s.cache.mu.RUnlock()
		return keywords
	}
	s.cache.mu.RUnlock()

	s.cache.mu.Lock()
	defer s.cache.mu.Unlock()

	// Double-check
	if s.cache.Keywords != nil && time.Now().Before(s.cache.Keywords.ExpiresAt) {
		return s.cache.Keywords.Data.([]string)
	}

	// Build common legal keywords from Polish legal system
	keywords := []string{
		// Constitutional law
		"konstytucja", "sejmk", "senart", "prezydent", "rząd", "minister",
		"trybunał konstytucyjny", "sąd najwyższy", "krajowa rada sądownictwa",

		// Administrative law
		"administracja", "samorząd", "gmina", "powiat", "województwo",
		"naczelne i centralne organy administracji", "kontrola państwowa",

		// Judicial system
		"sądy powszechne", "sądy administracyjne", "prokuratura", "adwokatura",
		"notariat", "komornictwo", "postępowanie cywilne", "postępowanie karne",

		// Economic law
		"podatki", "handel", "przedsiębiorczość", "bankowość", "ubezpieczenia",
		"rynek kapitałowy", "konkurencja", "zamówienia publiczne",

		// Social law
		"praca", "ubezpieczenia społeczne", "ochrona zdrowia", "edukacja",
		"kultura", "sport", "ochrona środowiska", "mieszkalnictwo",

		// International law
		"unia europejska", "traktaty międzynarodowe", "dyplomacja",
		"obronność", "bezpieczeństwo", "NATO",

		// Legal procedures
		"postępowanie", "procedura", "odwołanie", "skarga", "wniosek",
		"decyzja", "rozstrzygnięcie", "wykonanie", "egzekucja",
	}

	// Cache for 7 days (keywords in legal system are relatively stable)
	s.cache.Keywords = &CacheEntry{
		Data:      keywords,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	return keywords
}


// validateDocumentType checks if a document type is valid and suggests alternatives using fuzzy search
func (s *SejmServer) validateDocumentType(docType string) (bool, []string, error) {
	if docType == "" {
		return true, nil, nil // Empty is valid (optional parameter)
	}

	documentTypes := s.getCachedDocumentTypes()

	// Check exact match (case-insensitive)
	for _, validType := range documentTypes {
		if strings.EqualFold(validType, docType) {
			return true, nil, nil
		}
	}

	// Type not found, use fuzzy search to suggest similar types
	fuzzyMatches := s.fuzzyMatchText(docType, documentTypes, 0.5) // Lower threshold for suggestions

	var suggestions []string
	if len(fuzzyMatches) > 0 {
		suggestions = append(suggestions, "Did you mean:")
		for i, match := range fuzzyMatches {
			if i >= 5 { // Limit to top 5 suggestions
				break
			}
			confidence := ""
			if match.Score >= 0.8 {
				confidence = " (high confidence)"
			} else if match.Score >= 0.6 {
				confidence = " (medium confidence)"
			}
			suggestions = append(suggestions, fmt.Sprintf("• %s%s (similarity: %.0f%%)", match.Text, confidence, match.Score*100))
		}
	} else {
		suggestions = append(suggestions, "Valid document types:")

		// Show common types as fallback
		commonTypes := []string{"Ustawa", "Rozporządzenie", "Konstytucja", "Dekret", "Zarządzenie"}
		for _, commonType := range commonTypes {
			for _, validType := range documentTypes {
				if strings.EqualFold(validType, commonType) {
					suggestions = append(suggestions, fmt.Sprintf("• %s", validType))
					break
				}
			}
		}
	}

	return false, suggestions, nil
}

// validateKeywords provides keyword suggestions based on cached keywords using fuzzy search
func (s *SejmServer) validateKeywords(searchTerms string) []string {
	if searchTerms == "" {
		return nil
	}

	keywords := s.getCachedKeywords()

	var suggestions []string

	// First try fuzzy matching
	fuzzyMatches := s.fuzzyMatchText(searchTerms, keywords, 0.4) // Lower threshold for broader suggestions

	if len(fuzzyMatches) > 0 {
		suggestions = append(suggestions, "Related legal keywords:")
		for i, match := range fuzzyMatches {
			if i >= 8 { // Show more suggestions for keywords
				break
			}

			// Add context for the keyword
			context := s.getKeywordContext(match.Text)
			if context != "" {
				suggestions = append(suggestions, fmt.Sprintf("• '%s' (%s)", match.Text, context))
			} else {
				suggestions = append(suggestions, fmt.Sprintf("• '%s' (%.0f%% match)", match.Text, match.Score*100))
			}
		}
	} else {
		// If no fuzzy matches, suggest popular categories
		suggestions = append(suggestions, "Popular legal areas:")
		suggestions = append(suggestions, "• 'konstytucja' (constitutional law)")
		suggestions = append(suggestions, "• 'administracja' (administrative law)")
		suggestions = append(suggestions, "• 'sądy' (judicial system)")
		suggestions = append(suggestions, "• 'praca' (labor law)")
		suggestions = append(suggestions, "• 'podatki' (tax law)")
		suggestions = append(suggestions, "• 'ochrona środowiska' (environmental law)")
		suggestions = append(suggestions, "• 'ubezpieczenia' (insurance law)")
	}

	return suggestions
}

// getKeywordContext provides context for legal keywords
func (s *SejmServer) getKeywordContext(keyword string) string {
	contextMap := map[string]string{
		"konstytucja":             "constitutional law",
		"sejm":                    "parliament",
		"senat":                   "senate",
		"prezydent":               "presidential law",
		"rząd":                    "government",
		"minister":                "ministerial regulations",
		"trybunał konstytucyjny":  "constitutional court",
		"sąd najwyższy":           "supreme court",
		"administracja":           "administrative law",
		"samorząd":                "local government",
		"gmina":                   "municipal law",
		"powiat":                  "county law",
		"województwo":             "regional law",
		"sądy powszechne":         "common courts",
		"prokuratura":             "prosecution",
		"adwokatura":              "legal profession",
		"podatki":                 "tax law",
		"handel":                  "commercial law",
		"przedsiębiorczość":       "business law",
		"bankowość":               "banking law",
		"praca":                   "labor law",
		"ubezpieczenia społeczne": "social security",
		"ochrona zdrowia":         "healthcare law",
		"edukacja":                "education law",
		"ochrona środowiska":      "environmental law",
		"unia europejska":         "EU law",
		"NATO":                    "defense law",
	}

	keywordLower := strings.ToLower(keyword)
	if context, exists := contextMap[keywordLower]; exists {
		return context
	}

	// Try partial matches for compound keywords
	for key, context := range contextMap {
		if strings.Contains(keywordLower, key) || strings.Contains(key, keywordLower) {
			return context
		}
	}

	return ""
}




// FuzzyMatch represents a fuzzy search result with similarity score
type FuzzyMatch struct {
	Text      string
	Score     float64
	MatchType string // "exact", "fuzzy", "phonetic", "partial"
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func levenshteinDistance(s1, s2 string) int {
	if s1 == s2 {
		return 0
	}

	r1 := []rune(s1)
	r2 := []rune(s2)

	len1 := len(r1)
	len2 := len(r2)

	if len1 == 0 {
		return len2
	}
	if len2 == 0 {
		return len1
	}

	// Create a 2D array
	matrix := make([][]int, len1+1)
	for i := range matrix {
		matrix[i] = make([]int, len2+1)
	}

	// Initialize first row and column
	for i := 0; i <= len1; i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len2; j++ {
		matrix[0][j] = j
	}

	// Fill the matrix
	for i := 1; i <= len1; i++ {
		for j := 1; j <= len2; j++ {
			cost := 0
			if r1[i-1] != r2[j-1] {
				cost = 1
			}

			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len1][len2]
}


// min2 returns the minimum of two integers
func min2(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// similarity calculates similarity score (0.0 to 1.0) based on Levenshtein distance
func similarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}

	maxLen := len(s1)
	if len(s2) > maxLen {
		maxLen = len(s2)
	}

	if maxLen == 0 {
		return 1.0
	}

	distance := levenshteinDistance(s1, s2)
	return 1.0 - float64(distance)/float64(maxLen)
}

// normalizePolish normalizes Polish text for better fuzzy matching
func normalizePolish(text string) string {
	// Convert to lowercase
	text = strings.ToLower(text)

	// Polish character replacements for fuzzy matching
	replacements := map[rune]rune{
		'ą': 'a', 'ć': 'c', 'ę': 'e', 'ł': 'l',
		'ń': 'n', 'ó': 'o', 'ś': 's', 'ź': 'z', 'ż': 'z',
	}

	var result strings.Builder
	for _, r := range text {
		if replacement, exists := replacements[r]; exists {
			result.WriteRune(replacement)
		} else {
			result.WriteRune(r)
		}
	}

	// Handle common Polish spelling variations
	resultStr := result.String()

	// Handle 'ć' -> 'cy' pattern
	resultStr = strings.ReplaceAll(resultStr, "cy", "c")

	return resultStr
}

// jaroWinklerSimilarity calculates Jaro-Winkler similarity for better matching of similar strings
func jaroWinklerSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}

	len1, len2 := utf8.RuneCountInString(s1), utf8.RuneCountInString(s2)
	if len1 == 0 || len2 == 0 {
		return 0.0
	}

	// Convert to runes for proper Unicode handling
	r1 := []rune(s1)
	r2 := []rune(s2)

	// Calculate match window
	matchWindow := max(len1, len2)/2 - 1
	if matchWindow < 0 {
		matchWindow = 0
	}

	// Track matches
	matches1 := make([]bool, len1)
	matches2 := make([]bool, len2)

	matches := 0
	transpositions := 0

	// Find matches
	for i := 0; i < len1; i++ {
		start := max(0, i-matchWindow)
		end := min2(i+matchWindow+1, len2)

		for j := start; j < end; j++ {
			if matches2[j] || r1[i] != r2[j] {
				continue
			}

			matches1[i] = true
			matches2[j] = true
			matches++
			break
		}
	}

	if matches == 0 {
		return 0.0
	}

	// Count transpositions
	k := 0
	for i := 0; i < len1; i++ {
		if !matches1[i] {
			continue
		}

		for !matches2[k] {
			k++
		}

		if r1[i] != r2[k] {
			transpositions++
		}
		k++
	}

	// Calculate Jaro similarity
	jaro := (float64(matches)/float64(len1) +
		float64(matches)/float64(len2) +
		float64(matches-transpositions/2)/float64(matches)) / 3.0

	// Calculate Jaro-Winkler similarity (with prefix bonus)
	if jaro < 0.7 {
		return jaro
	}

	// Calculate common prefix length (up to 4 characters)
	prefix := 0
	for i := 0; i < min3(len1, len2, 4); i++ {
		if r1[i] == r2[i] {
			prefix++
		} else {
			break
		}
	}

	return jaro + 0.1*float64(prefix)*(1.0-jaro)
}


// min3 returns the minimum of three integers
func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// fuzzyMatchText performs fuzzy matching against a list of candidates
func (s *SejmServer) fuzzyMatchText(query string, candidates []string, threshold float64) []FuzzyMatch {
	if threshold <= 0 {
		threshold = 0.6 // Default threshold
	}

	var matches []FuzzyMatch
	queryLower := strings.ToLower(query)
	queryNormalized := normalizePolish(query)

	for _, candidate := range candidates {
		candidateLower := strings.ToLower(candidate)
		candidateNormalized := normalizePolish(candidate)

		// Exact match (highest priority)
		if queryLower == candidateLower {
			matches = append(matches, FuzzyMatch{
				Text:      candidate,
				Score:     1.0,
				MatchType: "exact",
			})
			continue
		}

		// Partial match (substring)
		if strings.Contains(candidateLower, queryLower) || strings.Contains(queryLower, candidateLower) {
			// Calculate partial match score based on length ratio
			score := float64(min2(len(queryLower), len(candidateLower))) / float64(max(len(queryLower), len(candidateLower)))
			if score >= threshold {
				matches = append(matches, FuzzyMatch{
					Text:      candidate,
					Score:     score * 0.9, // Slightly lower than exact match
					MatchType: "partial",
				})
				continue
			}
		}

		// Fuzzy match using Jaro-Winkler (good for typos)
		jaroScore := jaroWinklerSimilarity(queryNormalized, candidateNormalized)
		if jaroScore >= threshold {
			matches = append(matches, FuzzyMatch{
				Text:      candidate,
				Score:     jaroScore * 0.8, // Lower priority than partial match
				MatchType: "fuzzy",
			})
			continue
		}

		// Levenshtein-based similarity (good for character substitutions)
		levScore := similarity(queryNormalized, candidateNormalized)
		if levScore >= threshold {
			matches = append(matches, FuzzyMatch{
				Text:      candidate,
				Score:     levScore * 0.7, // Lowest priority
				MatchType: "fuzzy",
			})
		}
	}

	// Sort by score (descending)
	for i := 0; i < len(matches)-1; i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[i].Score < matches[j].Score {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}

	return matches
}

// HTTP Cache Statistics


// updateHTTPCacheStats updates cache statistics based on response headers
func (s *SejmServer) updateHTTPCacheStats(resp *http.Response) {
	s.cache.mu.Lock()
	defer s.cache.mu.Unlock()

	s.cache.HTTPStats.Requests++

	// Check if response came from cache
	if resp.Header.Get("X-From-Cache") == "1" {
		s.cache.HTTPStats.Hits++
	} else {
		s.cache.HTTPStats.Misses++
	}
}
