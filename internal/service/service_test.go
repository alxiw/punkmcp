package service_test

import (
	"encoding/json"
	"errors"
	"math/rand/v2"
	"sync"
	"testing"

	"github.com/alxiw/punkmcp/internal/cache"
	"github.com/alxiw/punkmcp/internal/model"
	"github.com/alxiw/punkmcp/internal/service"
)

// Compile-time check that *cache.Cache satisfies service.BeerStore.
var _ service.BeerStore = (*cache.Cache)(nil)

type memStore struct {
	beers []model.Beer
	byID  map[int]model.Beer
}

func newMemStore(beers ...model.Beer) *memStore {
	byID := make(map[int]model.Beer, len(beers))
	copied := make([]model.Beer, len(beers))
	for i, b := range beers {
		c := b.Clone()
		copied[i] = c
		byID[c.ID] = c
	}
	return &memStore{
		beers: copied,
		byID:  byID,
	}
}

func (m *memStore) GetAll() []model.Beer {
	out := make([]model.Beer, len(m.beers))
	for i, b := range m.beers {
		out[i] = b.Clone()
	}
	return out
}

func (m *memStore) GetByID(id int) (model.Beer, bool) {
	b, ok := m.byID[id]
	if !ok {
		return model.Beer{}, false
	}
	return b.Clone(), true
}

func f64(v float64) *float64 { return &v }
func str(v string) *string   { return &v }

func hops(names ...string) []model.Hop {
	out := make([]model.Hop, len(names))
	for i, n := range names {
		out[i] = model.Hop{Name: n}
	}
	return out
}

func malts(names ...string) []model.Malt {
	out := make([]model.Malt, len(names))
	for i, n := range names {
		out[i] = model.Malt{Name: n}
	}
	return out
}

func catalogFixture() []model.Beer {
	return []model.Beer{
		{
			ID:          1,
			Name:        "Punk IPA",
			Tagline:     "Post Modern Classic.",
			FirstBrewed: "04/2007",
			Description: "Our flagship India Pale Ale with tropical citrus hops.",
			ABV:         5.6,
			IBU:         f64(45),
			EBC:         f64(17),
			PH:          f64(4.4),
			Ingredients: model.Ingredients{
				Hops:  hops("Chinook", "Ahtanum", "Simcoe", "Chinook", "Cascade"),
				Malt:  malts("Extra Pale", "Caramalt"),
				Yeast: str("Wyeast 1056 - American Ale"),
			},
			FoodPairing: []string{"Spicy carne asada", "Cheesecake"},
		},
		{
			ID:          2,
			Name:        "Trashy Blonde",
			Tagline:     "You Know You Shouldn't.",
			FirstBrewed: "04/2008",
			Description: "A titillating, neurotic, peroxide punk of a Pale Ale.",
			ABV:         4.1,
			IBU:         f64(41.5),
			EBC:         f64(15),
			PH:          f64(4.4),
			Ingredients: model.Ingredients{
				Hops:  hops("Amarillo", "Simcoe", "Motueka"),
				Malt:  malts("Maris Otter Extra Pale", "Caramalt"),
				Yeast: str("Wyeast 1056 - American Ale"),
			},
			FoodPairing: []string{"Fresh crab with lemon", "Garlic butter dipping sauce"},
		},
		{
			ID:          3,
			Name:        "Imperial Cocoa Stout",
			Tagline:     "Dark Roasty Monster.",
			FirstBrewed: "2015",
			Description: "Heavy chocolate and roasted coffee imperial stout.",
			ABV:         10.5,
			IBU:         f64(85),
			EBC:         f64(140),
			PH:          f64(4.2),
			Ingredients: model.Ingredients{
				Hops:  hops("Magnum", "Cascade"),
				Malt:  malts("Extra Pale", "Chocolate", "Roasted Barley"),
				Yeast: str("Wyeast 1272 - American Ale II"),
			},
			FoodPairing: []string{"Dark chocolate mousse", "Roast beef"},
		},
		{
			ID:          4,
			Name:        "Mystery Pilsner",
			Tagline:     "Crisp Lager Prototype.",
			FirstBrewed: "11/2015",
			Description: "Crisp German style pilsner with missing lab measurements.",
			ABV:         4.8,
			IBU:         nil,
			EBC:         nil,
			PH:          nil,
			Ingredients: model.Ingredients{
				Hops:  hops("Saaz", "Hallertauer"),
				Malt:  malts("Pilsner"),
				Yeast: nil,
			},
			FoodPairing: []string{"Grilled bratwurst", "Soft pretzels"},
		},
		{
			ID:          5,
			Name:        "amber haze",
			Tagline:     "Unfiltered Citrus Bomb.",
			FirstBrewed: "06/2018",
			Description: "Juicy hazy pale ale dry-hopped heavily with Citra.",
			ABV:         5.6,
			IBU:         f64(35),
			EBC:         nil,
			PH:          f64(4.5),
			Ingredients: model.Ingredients{
				Hops:  hops("Citra", "Mosaic", "cascade"),
				Malt:  malts("Extra Pale", "Oats"),
				Yeast: str("wyeast 1056 - american ale"),
			},
			FoodPairing: []string{"Spicy tacos", "Mango salsa"},
		},
		{
			ID:          6,
			Name:        "Undated Experimental",
			Tagline:     "Vault Release.",
			FirstBrewed: "",
			Description: "Small batch release with no brew date.",
			ABV:         6.2,
			IBU:         nil,
			EBC:         f64(30),
			Ingredients: model.Ingredients{
				Hops:  hops("Cascade"),
				Malt:  malts("Extra Pale"),
				Yeast: str("Wyeast 1056 - American Ale"),
			},
			FoodPairing: []string{"Aged cheddar"},
		},
	}
}

func TestGetBeerRecipe(t *testing.T) {
	t.Parallel()

	svc := service.New(newMemStore(catalogFixture()...))

	tests := []struct {
		name     string
		id       int
		wantName string
		wantErr  error
	}{
		{
			name:     "returns existing recipe",
			id:       1,
			wantName: "Punk IPA",
		},
		{
			name:     "returns recipe with nil metrics",
			id:       4,
			wantName: "Mystery Pilsner",
		},
		{
			name:    "returns ErrNotFound for missing ID",
			id:      999,
			wantErr: service.ErrNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := svc.GetBeerRecipe(tc.id)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("GetBeerRecipe(%d) err = %v, want %v", tc.id, err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ID != tc.id || got.Name != tc.wantName {
				t.Fatalf("got (%d, %q), want (%d, %q)", got.ID, got.Name, tc.id, tc.wantName)
			}
		})
	}
}

func TestGetRandomBeer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		params    service.RandomParams
		seed1     uint64
		seed2     uint64
		wantInIDs []int
		wantErr   error
	}{
		{
			name:      "no filters selects deterministically from full catalog",
			params:    service.RandomParams{},
			seed1:     42,
			seed2:     99,
			wantInIDs: []int{1, 2, 3, 4, 5, 6},
		},
		{
			name: "filters by ABV range",
			params: service.RandomParams{
				ABVMin: f64(5.5),
				ABVMax: f64(6.5),
			},
			seed1:     7,
			seed2:     13,
			wantInIDs: []int{1, 5, 6},
		},
		{
			name: "filters by IBU range and excludes nil IBU",
			params: service.RandomParams{
				IBUMin: f64(40),
				IBUMax: f64(50),
			},
			seed1:     1,
			seed2:     2,
			wantInIDs: []int{1, 2},
		},
		{
			name: "filters by EBC range and excludes nil EBC",
			params: service.RandomParams{
				EBCMin: f64(15),
				EBCMax: f64(30),
			},
			seed1:     3,
			seed2:     4,
			wantInIDs: []int{1, 2, 6},
		},
		{
			name: "filters by EBCMin only and excludes nil EBC",
			params: service.RandomParams{
				EBCMin: f64(100),
			},
			seed1:     5,
			seed2:     6,
			wantInIDs: []int{3},
		},
		{
			name: "filters by food case-insensitively",
			params: service.RandomParams{
				Food: "SPICY",
			},
			seed1:     10,
			seed2:     20,
			wantInIDs: []int{1, 5},
		},
		{
			name: "combined filters narrow to single beer",
			params: service.RandomParams{
				ABVMin: f64(10.0),
				IBUMin: f64(80.0),
				EBCMin: f64(120.0),
				EBCMax: f64(150.0),
				Food:   "chocolate",
			},
			seed1:     1,
			seed2:     1,
			wantInIDs: []int{3},
		},
		{
			name: "filters by brewed_after and excludes undated beers",
			params: service.RandomParams{
				BrewedAfter: "2015",
			},
			seed1:     11,
			seed2:     22,
			wantInIDs: []int{3, 4, 5},
		},
		{
			name: "filters by brewed_before MM/YYYY and excludes undated beers",
			params: service.RandomParams{
				BrewedBefore: "04/2008",
			},
			seed1:     33,
			seed2:     44,
			wantInIDs: []int{1, 2},
		},
		{
			name: "filters by brewed_after and brewed_before range (MM/YYYY and YYYY)",
			params: service.RandomParams{
				BrewedAfter:  "2015",
				BrewedBefore: "2015",
			},
			seed1:     55,
			seed2:     66,
			wantInIDs: []int{3, 4},
		},
		{
			name: "date filter excludes undated beer even when all other beers match",
			params: service.RandomParams{
				ABVMin:       f64(6.0),
				ABVMax:       f64(6.5), // Only undated beer 6 (ABV 6.2) is in this ABV range
				BrewedBefore: "2099",
			},
			seed1:   1,
			seed2:   1,
			wantErr: service.ErrNotFound,
		},
		{
			name: "invalid brewed_before returns ErrInvalidInput",
			params: service.RandomParams{
				BrewedBefore: "2015-05",
			},
			seed1:   1,
			seed2:   1,
			wantErr: service.ErrInvalidInput,
		},
		{
			name: "invalid brewed_after returns ErrInvalidInput",
			params: service.RandomParams{
				BrewedAfter: "13/2015",
			},
			seed1:   1,
			seed2:   1,
			wantErr: service.ErrInvalidInput,
		},
		{
			name: "no matching beers returns ErrNotFound",
			params: service.RandomParams{
				ABVMin: f64(25.0),
			},
			seed1:   1,
			seed2:   1,
			wantErr: service.ErrNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store := newMemStore(catalogFixture()...)
			svcA := service.New(store, service.WithRandSource(rand.NewPCG(tc.seed1, tc.seed2)))
			svcB := service.New(store, service.WithRandSource(rand.NewPCG(tc.seed1, tc.seed2)))

			gotA, errA := svcA.GetRandomBeer(tc.params)
			gotB, errB := svcB.GetRandomBeer(tc.params)

			if tc.wantErr != nil {
				if !errors.Is(errA, tc.wantErr) || !errors.Is(errB, tc.wantErr) {
					t.Fatalf("got errors (%v, %v), want %v", errA, errB, tc.wantErr)
				}
				return
			}
			if errA != nil || errB != nil {
				t.Fatalf("unexpected errors: %v, %v", errA, errB)
			}
			if gotA.ID != gotB.ID {
				t.Fatalf("deterministic seed produced different IDs: %d vs %d", gotA.ID, gotB.ID)
			}

			found := false
			for _, allowed := range tc.wantInIDs {
				if gotA.ID == allowed {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("selected beer ID %d not in expected set %v", gotA.ID, tc.wantInIDs)
			}
		})
	}
}

func TestGetRandomBeer_ConcurrentWithCustomRNG(t *testing.T) {
	t.Parallel()

	svc := service.New(
		newMemStore(catalogFixture()...),
		service.WithRandSource(rand.NewPCG(123, 456)),
	)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 25; j++ {
				if _, err := svc.GetRandomBeer(service.RandomParams{}); err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()
}

func TestService_NilStoreDoesNotPanic(t *testing.T) {
	t.Parallel()

	svc := service.New(nil)

	if _, err := svc.GetBeerRecipe(1); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("GetBeerRecipe on nil store: got %v, want ErrNotFound", err)
	}
	if _, err := svc.GetRandomBeer(service.RandomParams{}); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("GetRandomBeer on nil store: got %v, want ErrNotFound", err)
	}
	res, err := svc.SearchBeers(service.SearchParams{})
	if err != nil {
		t.Fatalf("SearchBeers on nil store: unexpected err %v", err)
	}
	if res.TotalFound != 0 || res.Hint == "" {
		t.Fatalf("expected empty SearchResult with hint, got %+v", res)
	}
	stats := svc.GetCatalogStats()
	if stats.TotalBeers != 0 || stats.TopHops == nil || stats.TopYeasts == nil || stats.ByYear == nil {
		t.Fatalf("expected zeroed non-nil CatalogStats, got %+v", stats)
	}
}

func TestDTOJSONSnakeCase(t *testing.T) {
	t.Parallel()

	svc := service.New(newMemStore(catalogFixture()...))
	res, err := svc.SearchBeers(service.SearchParams{Limit: 1})
	if err != nil {
		t.Fatalf("SearchBeers error: %v", err)
	}

	raw, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}

	for _, key := range []string{"total_found", "limit", "offset", "beers"} {
		if _, ok := decoded[key]; !ok {
			t.Errorf("expected snake_case key %q in SearchResult JSON: %s", key, string(raw))
		}
	}
	if _, ok := decoded["hint"]; ok {
		t.Errorf("expected hint to be omitted when TotalFound > 0: %s", string(raw))
	}

	beersSlice, ok := decoded["beers"].([]any)
	if !ok || len(beersSlice) != 1 {
		t.Fatalf("expected 1 beer in JSON beers array: %s", string(raw))
	}
	beerObj, ok := beersSlice[0].(map[string]any)
	if !ok {
		t.Fatalf("expected beer summary object: %s", string(raw))
	}
	for _, key := range []string{"id", "name", "tagline", "description", "abv", "ibu", "ebc", "first_brewed", "food_pairing"} {
		if _, ok := beerObj[key]; !ok {
			t.Errorf("expected key %q in BeerSummary JSON: %s", key, string(raw))
		}
	}
	if _, ok := beerObj["yeast"]; ok {
		t.Errorf("did not expect yeast key in BeerSummary JSON: %s", string(raw))
	}

	// Verify FoodPairing in BeerSummary is defensively copied and Description is populated.
	if res.Beers[0].Description == "" {
		t.Errorf("expected non-empty Description in BeerSummary")
	}
	if len(res.Beers[0].FoodPairing) != 2 {
		t.Fatalf("expected 2 food pairings in BeerSummary, got %v", res.Beers[0].FoodPairing)
	}
	res.Beers[0].FoodPairing[0] = "Mutated"
	fresh, err := svc.GetBeerRecipe(res.Beers[0].ID)
	if err != nil {
		t.Fatalf("GetBeerRecipe error: %v", err)
	}
	if fresh.FoodPairing[0] == "Mutated" {
		t.Errorf("mutating BeerSummary.FoodPairing affected underlying store data")
	}
}
