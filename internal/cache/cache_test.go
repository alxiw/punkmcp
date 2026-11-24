package cache_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alxiw/punkmcp/internal/cache"
	"github.com/alxiw/punkmcp/internal/model"
)

type mockSource struct {
	mu    sync.Mutex
	fn    func(ctx context.Context, call int) ([]model.Beer, error)
	calls atomic.Int32
}

func (m *mockSource) LoadBeers(ctx context.Context) ([]model.Beer, error) {
	n := int(m.calls.Add(1))
	m.mu.Lock()
	fn := m.fn
	m.mu.Unlock()
	return fn(ctx, n)
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func sampleBeers(ids ...int) []model.Beer {
	out := make([]model.Beer, 0, len(ids))
	for _, id := range ids {
		ibu := float64(id * 10)
		yeast := "US-05"
		twist := "Oak chips"
		attr := "bitter"
		temp := 65.0
		dur := 60
		img := "001.png"

		out = append(out, model.Beer{
			ID:          id,
			Name:        "Beer",
			Tagline:     "Tagline",
			FirstBrewed: "05/2019",
			Description: "Description",
			Image:       &img,
			ABV:         5.5,
			IBU:         &ibu,
			Method: model.Method{
				MashTemp: []model.MashTemp{
					{
						Temp:     model.Temperature{Value: &temp, Unit: "celsius"},
						Duration: &dur,
					},
				},
				Fermentation: model.Fermentation{
					Temp: model.Temperature{Value: &temp, Unit: "celsius"},
				},
				Twist: &twist,
			},
			Ingredients: model.Ingredients{
				Malt:  []model.Malt{{Name: "Pale", Amount: model.Amount{Value: 4.5, Unit: "kilograms"}}},
				Hops:  []model.Hop{{Name: "Simcoe", Amount: model.Amount{Value: 20, Unit: "grams"}, Add: "start", Attribute: &attr}},
				Yeast: &yeast,
			},
			FoodPairing: []string{"Tacos", "Burgers"},
		})
	}
	return out
}

func TestCache_LoadAndLookup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		loadBeers    []model.Beer
		loadErr      error
		lookupID     int
		wantErr      bool
		wantCount    int
		wantFound    bool
		wantLookupID int
	}{
		{
			name:         "loads multiple beers and finds existing ID",
			loadBeers:    sampleBeers(1, 2, 42),
			lookupID:     42,
			wantCount:    3,
			wantFound:    true,
			wantLookupID: 42,
		},
		{
			name:      "lookup missing ID returns false",
			loadBeers: sampleBeers(1, 2),
			lookupID:  999,
			wantCount: 2,
			wantFound: false,
		},
		{
			name:      "empty catalog loads cleanly",
			loadBeers: []model.Beer{},
			lookupID:  1,
			wantCount: 0,
			wantFound: false,
		},
		{
			name:      "initial load error fails fast and leaves cache empty",
			loadErr:   errors.New("upstream unavailable"),
			lookupID:  1,
			wantErr:   true,
			wantCount: 0,
			wantFound: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			src := &mockSource{
				fn: func(ctx context.Context, call int) ([]model.Beer, error) {
					return tc.loadBeers, tc.loadErr
				},
			}

			c := cache.New(src, time.Hour, discardLogger())
			err := c.Load(context.Background())

			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !c.LastUpdated().IsZero() {
					t.Errorf("expected LastUpdated to remain zero on failed initial load, got %v", c.LastUpdated())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if c.LastUpdated().IsZero() {
					t.Errorf("expected LastUpdated to be populated after load")
				}
			}

			if got := c.Count(); got != tc.wantCount {
				t.Fatalf("Count() = %d, want %d", got, tc.wantCount)
			}

			all := c.GetAll()
			if len(all) != tc.wantCount {
				t.Fatalf("len(GetAll()) = %d, want %d", len(all), tc.wantCount)
			}

			beer, ok := c.GetByID(tc.lookupID)
			if ok != tc.wantFound {
				t.Fatalf("GetByID(%d) found = %v, want %v", tc.lookupID, ok, tc.wantFound)
			}
			if tc.wantFound && beer.ID != tc.wantLookupID {
				t.Fatalf("GetByID(%d).ID = %d, want %d", tc.lookupID, beer.ID, tc.wantLookupID)
			}
		})
	}
}

func TestCache_ReturnsSafeCopies(t *testing.T) {
	t.Parallel()

	src := &mockSource{
		fn: func(ctx context.Context, call int) ([]model.Beer, error) {
			return sampleBeers(1), nil
		},
	}

	c := cache.New(src, time.Hour, discardLogger())
	if err := c.Load(context.Background()); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Mutate slice, nested slices, and pointers returned by GetAll.
	all := c.GetAll()
	all[0].Name = "Mutated"
	*all[0].IBU = 999
	all[0].FoodPairing[0] = "Mutated Food"
	*all[0].Ingredients.Yeast = "Mutated Yeast"
	*all[0].Ingredients.Hops[0].Attribute = "mutated"
	*all[0].Method.MashTemp[0].Duration = 999

	// Mutate copy returned by GetByID.
	byID, ok := c.GetByID(1)
	if !ok {
		t.Fatalf("expected beer 1 in cache")
	}
	byID.Name = "Mutated Again"
	*byID.IBU = 888
	byID.FoodPairing[1] = "Mutated Food 2"

	// Verify internal cache state is completely unaffected.
	fresh, ok := c.GetByID(1)
	if !ok {
		t.Fatalf("expected beer 1 in cache")
	}
	if fresh.Name != "Beer" {
		t.Errorf("Name was mutated: got %q, want %q", fresh.Name, "Beer")
	}
	if *fresh.IBU != 10 {
		t.Errorf("IBU was mutated: got %v, want 10", *fresh.IBU)
	}
	if fresh.FoodPairing[0] != "Tacos" || fresh.FoodPairing[1] != "Burgers" {
		t.Errorf("FoodPairing was mutated: got %v", fresh.FoodPairing)
	}
	if *fresh.Ingredients.Yeast != "US-05" {
		t.Errorf("Yeast was mutated: got %q", *fresh.Ingredients.Yeast)
	}
	if *fresh.Ingredients.Hops[0].Attribute != "bitter" {
		t.Errorf("Hop attribute was mutated: got %q", *fresh.Ingredients.Hops[0].Attribute)
	}
	if *fresh.Method.MashTemp[0].Duration != 60 {
		t.Errorf("MashTemp duration was mutated: got %d", *fresh.Method.MashTemp[0].Duration)
	}
}

func TestCache_StartRefreshLoop(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		refreshBeers   []model.Beer
		refreshErr     error
		wantCountAfter int
		wantHasID      int
		wantMissingID  int
	}{
		{
			name:           "successful background refresh replaces cached data",
			refreshBeers:   sampleBeers(10, 20, 30),
			wantCountAfter: 3,
			wantHasID:      30,
			wantMissingID:  1,
		},
		{
			name:           "failed background refresh retains previous valid data",
			refreshErr:     errors.New("temporary network failure"),
			wantCountAfter: 2,
			wantHasID:      2,
			wantMissingID:  30,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			refreshedCh := make(chan struct{}, 1)
			src := &mockSource{
				fn: func(ctx context.Context, call int) ([]model.Beer, error) {
					if call == 1 {
						return sampleBeers(1, 2), nil
					}
					select {
					case refreshedCh <- struct{}{}:
					default:
					}
					return tc.refreshBeers, tc.refreshErr
				},
			}

			c := cache.New(src, 15*time.Millisecond, discardLogger())
			if err := c.Load(context.Background()); err != nil {
				t.Fatalf("initial Load failed: %v", err)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			c.StartRefreshLoop(ctx)

			select {
			case <-refreshedCh:
			case <-time.After(2 * time.Second):
				t.Fatal("timed out waiting for background refresh tick")
			}

			// Wait briefly for Load to finish applying the swap after returning from mockSource.
			deadline := time.Now().Add(500 * time.Millisecond)
			for time.Now().Before(deadline) {
				if c.Count() == tc.wantCountAfter {
					if _, ok := c.GetByID(tc.wantHasID); ok {
						break
					}
				}
				time.Sleep(5 * time.Millisecond)
			}

			if got := c.Count(); got != tc.wantCountAfter {
				t.Fatalf("Count() after refresh = %d, want %d", got, tc.wantCountAfter)
			}
			if _, ok := c.GetByID(tc.wantHasID); !ok {
				t.Fatalf("expected ID %d to exist after refresh", tc.wantHasID)
			}
			if _, ok := c.GetByID(tc.wantMissingID); ok {
				t.Fatalf("expected ID %d to be absent after refresh", tc.wantMissingID)
			}

			// Cancel context and verify no further refreshes occur.
			cancel()
			time.Sleep(50 * time.Millisecond)
			callsAfterCancel := src.calls.Load()
			time.Sleep(60 * time.Millisecond)
			if got := src.calls.Load(); got != callsAfterCancel {
				t.Fatalf("refresh loop did not stop after cancel: calls grew from %d to %d", callsAfterCancel, got)
			}
		})
	}
}

func TestCache_ConcurrentReadsAndWrites(t *testing.T) {
	t.Parallel()

	src := &mockSource{
		fn: func(ctx context.Context, call int) ([]model.Beer, error) {
			return sampleBeers(1, 2, 3, 4, 5), nil
		},
	}

	c := cache.New(src, time.Hour, discardLogger())
	if err := c.Load(context.Background()); err != nil {
		t.Fatalf("initial Load failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	// Spawn concurrent writers.
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 25; j++ {
				if err := c.Load(ctx); err != nil {
					t.Errorf("concurrent Load failed: %v", err)
					return
				}
			}
		}()
	}

	// Spawn concurrent readers that also mutate their returned copies.
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				if c.Count() != 5 {
					t.Errorf("unexpected count: %d", c.Count())
					return
				}
				all := c.GetAll()
				if len(all) != 5 {
					t.Errorf("unexpected GetAll len: %d", len(all))
					return
				}
				all[0].Name = "local-mutation"
				*all[0].IBU = -1

				b, ok := c.GetByID(3)
				if !ok || b.ID != 3 {
					t.Errorf("GetByID(3) failed: ok=%v, id=%d", ok, b.ID)
					return
				}
				b.FoodPairing[0] = "local-mutation"
			}
		}()
	}

	wg.Wait()
}
