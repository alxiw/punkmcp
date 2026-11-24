package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alxiw/punkmcp/internal/cache"
	"github.com/alxiw/punkmcp/internal/source"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	baseURL, cacheTTL, err := parseConfig(os.Args[1:])
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	src := source.NewHTTPSource(baseURL)
	beerCache := cache.New(src, cacheTTL, logger)

	if err := beerCache.Load(ctx); err != nil {
		logger.Error("failed to load beer catalog on startup", "base_url", baseURL, "error", err)
		os.Exit(1)
	}

	logger.Info(
		"loaded beer catalog",
		"count", beerCache.Count(),
		"base_url", baseURL,
		"cache_ttl", cacheTTL.String(),
		"last_updated", beerCache.LastUpdated().Format(time.RFC3339),
	)
}

func parseConfig(args []string) (string, time.Duration, error) {
	defaultBaseURL := os.Getenv("PUNKAPI_BASE_URL")
	if defaultBaseURL == "" {
		defaultBaseURL = source.DefaultBaseURL
	}

	defaultTTL := cache.DefaultTTL
	if rawTTL := os.Getenv("CACHE_TTL"); rawTTL != "" {
		parsed, err := time.ParseDuration(rawTTL)
		if err != nil {
			return "", 0, fmt.Errorf("parse CACHE_TTL %q: %w", rawTTL, err)
		}
		if parsed <= 0 {
			return "", 0, fmt.Errorf("CACHE_TTL must be positive, got %s", parsed)
		}
		defaultTTL = parsed
	}

	fs := flag.NewFlagSet("punkmcp", flag.ContinueOnError)
	baseURL := fs.String("punkapi-base-url", defaultBaseURL, "Base URL for PunkAPI (env: PUNKAPI_BASE_URL)")
	cacheTTL := fs.Duration("cache-ttl", defaultTTL, "Cache refresh interval (env: CACHE_TTL)")

	if err := fs.Parse(args); err != nil {
		return "", 0, err
	}
	if *cacheTTL <= 0 {
		return "", 0, fmt.Errorf("cache-ttl must be positive, got %s", *cacheTTL)
	}

	return *baseURL, *cacheTTL, nil
}
