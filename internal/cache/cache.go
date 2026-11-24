package cache

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/alxiw/punkmcp/internal/model"
	"github.com/alxiw/punkmcp/internal/source"
)

// DefaultTTL is the fallback refresh interval when a non-positive TTL is provided.
const DefaultTTL = 24 * time.Hour

// Cache is a thread-safe in-memory store for all beer recipes loaded from a DataSource.
type Cache struct {
	src    source.DataSource
	ttl    time.Duration
	logger *slog.Logger

	mu          sync.RWMutex
	beers       []model.Beer
	byID        map[int]model.Beer
	lastUpdated time.Time
}

// New constructs a new Cache backed by src with the given refresh TTL and logger.
func New(src source.DataSource, ttl time.Duration, logger *slog.Logger) *Cache {
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &Cache{
		src:    src,
		ttl:    ttl,
		logger: logger,
		byID:   make(map[int]model.Beer),
	}
}

// Load synchronously fetches all beers from the DataSource and atomically replaces the cache.
// If loading fails, any previously cached data remains untouched and an error is returned.
func (c *Cache) Load(ctx context.Context) error {
	if c.src == nil {
		return errors.New("cache data source is nil")
	}

	loaded, err := c.src.LoadBeers(ctx)
	if err != nil {
		return fmt.Errorf("load beers into cache: %w", err)
	}

	beers := make([]model.Beer, len(loaded))
	byID := make(map[int]model.Beer, len(loaded))
	for i, b := range loaded {
		cloned := b.Clone()
		beers[i] = cloned
		byID[cloned.ID] = cloned
	}

	c.mu.Lock()
	c.beers = beers
	c.byID = byID
	c.lastUpdated = time.Now().UTC()
	c.mu.Unlock()

	return nil
}

// StartRefreshLoop starts a background goroutine that refreshes the cache every TTL.
// If a refresh attempt fails, the error is logged and the previous valid dataset is retained.
// The goroutine stops cleanly when ctx is canceled.
func (c *Cache) StartRefreshLoop(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(c.ttl)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := c.Load(ctx); err != nil {
					if ctx.Err() != nil {
						return
					}
					c.logger.Error(
						"failed to refresh beer cache; retaining previous data",
						"error", err,
						"cached_count", c.Count(),
					)
					continue
				}
				c.logger.Info(
					"refreshed beer cache",
					"count", c.Count(),
				)
			}
		}
	}()
}

// GetAll returns a deep-copied snapshot of all cached beers so callers cannot mutate cache state.
func (c *Cache) GetAll() []model.Beer {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make([]model.Beer, len(c.beers))
	for i, b := range c.beers {
		out[i] = b.Clone()
	}
	return out
}

// GetByID returns a deep copy of the beer with the given ID, if present in the cache.
func (c *Cache) GetByID(id int) (model.Beer, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	b, ok := c.byID[id]
	if !ok {
		return model.Beer{}, false
	}
	return b.Clone(), true
}

// Count returns the number of beers currently held in the cache.
func (c *Cache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.beers)
}

// LastUpdated returns the UTC timestamp of the most recent successful cache load.
func (c *Cache) LastUpdated() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.lastUpdated
}
