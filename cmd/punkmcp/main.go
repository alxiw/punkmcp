package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/alxiw/punkmcp/internal/cache"
	punkmcp "github.com/alxiw/punkmcp/internal/mcp"
	"github.com/alxiw/punkmcp/internal/service"
	"github.com/alxiw/punkmcp/internal/source"
)

const (
	defaultTransport = "stdio"
	defaultAddr      = ":8080"
	shutdownTimeout  = 10 * time.Second
)

// version can be overridden at build time via -ldflags "-X main.version=...".
var version = "dev"

type config struct {
	Transport      string
	Addr           string
	PunkAPIBaseURL string
	CacheTTL       time.Duration
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg, err := parseConfig(os.Args[1:], os.Getenv, os.Stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	src := source.NewHTTPSource(cfg.PunkAPIBaseURL)
	beerCache := cache.New(src, cfg.CacheTTL, logger)

	if err := beerCache.Load(ctx); err != nil {
		logger.Error("failed to load beer catalog on startup", "base_url", cfg.PunkAPIBaseURL, "error", err)
		os.Exit(1)
	}

	logger.Info(
		"loaded beer catalog",
		"count", beerCache.Count(),
		"base_url", cfg.PunkAPIBaseURL,
		"cache_ttl", cfg.CacheTTL.String(),
		"last_updated", beerCache.LastUpdated().Format(time.RFC3339),
	)

	beerCache.StartRefreshLoop(ctx)

	svc := service.New(beerCache)
	mcpServer := punkmcp.NewServer(svc, version)

	switch cfg.Transport {
	case "stdio":
		if err := runStdio(ctx, mcpServer, logger); err != nil {
			logger.Error("stdio server exited with error", "error", err)
			os.Exit(1)
		}
	case "http":
		if err := runHTTP(ctx, mcpServer, beerCache, cfg.Addr, logger); err != nil {
			logger.Error("http server exited with error", "error", err)
			os.Exit(1)
		}
	}
}

func runStdio(ctx context.Context, server *gomcp.Server, logger *slog.Logger) error {
	logger.Info("starting punkmcp server", "transport", "stdio", "version", version)
	if err := server.Run(ctx, &gomcp.StdioTransport{}); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	logger.Info("punkmcp stdio server stopped")
	return nil
}

func runHTTP(ctx context.Context, server *gomcp.Server, counter punkmcp.CatalogCounter, addr string, logger *slog.Logger) error {
	handler := punkmcp.NewHTTPHandler(server, counter, logger)
	httpSrv := punkmcp.NewHTTPServer(addr, handler)

	errCh := make(chan error, 1)
	go func() {
		logger.Info("starting punkmcp server", "transport", "http", "addr", addr, "version", version)
		errCh <- httpSrv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
		logger.Info("shutting down http server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		shutdownErr := httpSrv.Shutdown(shutdownCtx)
		serveErr := <-errCh
		if shutdownErr != nil {
			return fmt.Errorf("graceful http shutdown: %w", shutdownErr)
		}
		if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			return serveErr
		}
		logger.Info("punkmcp http server stopped cleanly")
		return nil
	}
}

func parseConfig(args []string, getenv func(string) string, flagOutput io.Writer) (config, error) {
	defaultTransportVal := strings.TrimSpace(getenv("MCP_TRANSPORT"))
	if defaultTransportVal == "" {
		defaultTransportVal = defaultTransport
	}

	defaultAddrVal := strings.TrimSpace(getenv("MCP_ADDR"))
	if defaultAddrVal == "" {
		defaultAddrVal = defaultAddr
	}

	defaultBaseURL := strings.TrimSpace(getenv("PUNKAPI_BASE_URL"))
	if defaultBaseURL == "" {
		defaultBaseURL = source.DefaultBaseURL
	}

	defaultTTL := cache.DefaultTTL
	if rawTTL := strings.TrimSpace(getenv("CACHE_TTL")); rawTTL != "" {
		parsed, err := time.ParseDuration(rawTTL)
		if err != nil {
			return config{}, fmt.Errorf("parse CACHE_TTL %q: %w", rawTTL, err)
		}
		if parsed <= 0 {
			return config{}, fmt.Errorf("CACHE_TTL must be positive, got %s", parsed)
		}
		defaultTTL = parsed
	}

	fs := flag.NewFlagSet("punkmcp", flag.ContinueOnError)
	if flagOutput != nil {
		fs.SetOutput(flagOutput)
	}

	transport := fs.String("transport", defaultTransportVal, "MCP transport mode: stdio or http (env: MCP_TRANSPORT)")
	addr := fs.String("addr", defaultAddrVal, "HTTP listen address in http mode (env: MCP_ADDR)")
	baseURL := fs.String("punkapi-base-url", defaultBaseURL, "Base URL for PunkAPI (env: PUNKAPI_BASE_URL)")
	cacheTTL := fs.Duration("cache-ttl", defaultTTL, "Cache refresh interval (env: CACHE_TTL)")

	if err := fs.Parse(args); err != nil {
		return config{}, err
	}

	normalizedTransport := strings.ToLower(strings.TrimSpace(*transport))
	switch normalizedTransport {
	case "stdio", "http":
	default:
		return config{}, fmt.Errorf("unsupported transport %q (allowed: stdio, http)", *transport)
	}

	trimmedAddr := strings.TrimSpace(*addr)
	if normalizedTransport == "http" && trimmedAddr == "" {
		return config{}, errors.New("addr must not be empty when transport is http")
	}

	if *cacheTTL <= 0 {
		return config{}, fmt.Errorf("cache-ttl must be positive, got %s", *cacheTTL)
	}

	return config{
		Transport:      normalizedTransport,
		Addr:           trimmedAddr,
		PunkAPIBaseURL: strings.TrimSpace(*baseURL),
		CacheTTL:       *cacheTTL,
	}, nil
}
