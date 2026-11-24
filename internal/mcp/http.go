package mcp

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	defaultReadHeaderTimeout = 10 * time.Second
	defaultIdleTimeout       = 120 * time.Second
	defaultSessionTimeout    = 30 * time.Minute
	defaultMaxHeaderBytes    = 1 << 20 // 1 MiB
)

// CatalogCounter reports the number of beer recipes currently loaded in memory.
// It is satisfied by *cache.Cache.
type CatalogCounter interface {
	Count() int
}

type healthResponse struct {
	Status      string `json:"status"`
	BeersLoaded int    `json:"beers_loaded"`
}

// NewHTTPHandler builds an http.Handler serving the MCP Streamable HTTP endpoint
// at /mcp (POST/GET/DELETE) and a lightweight JSON healthcheck at GET /healthz.
func NewHTTPHandler(server *gomcp.Server, counter CatalogCounter, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()

	mcpHandler := gomcp.NewStreamableHTTPHandler(
		func(*http.Request) *gomcp.Server {
			return server
		},
		&gomcp.StreamableHTTPOptions{
			Logger:         logger,
			SessionTimeout: defaultSessionTimeout,
		},
	)

	mux.Handle("/mcp", mcpHandler)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		count := 0
		if counter != nil {
			count = counter.Count()
		}

		w.Header().Set("Content-Type", "application/json")
		resp := healthResponse{
			Status:      "ok",
			BeersLoaded: count,
		}
		statusCode := http.StatusOK
		if count <= 0 {
			resp.Status = "unavailable"
			statusCode = http.StatusServiceUnavailable
		}

		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(resp)
	})

	return mux
}

// NewHTTPServer creates an *http.Server configured for Streamable HTTP MCP traffic.
// WriteTimeout and ReadTimeout are intentionally left unset (0) so long-lived SSE
// and streaming MCP connections are not prematurely terminated.
func NewHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		IdleTimeout:       defaultIdleTimeout,
		MaxHeaderBytes:    defaultMaxHeaderBytes,
	}
}
