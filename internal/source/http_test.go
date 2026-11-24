package source_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alxiw/punkmcp/internal/model"
	"github.com/alxiw/punkmcp/internal/source"
)

func TestHTTPSource_LoadBeers_PaginationAndErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		perPage         int
		pages           map[int]string
		statusByPage    map[int]int
		wantCount       int
		wantRequests    int
		wantErrContains string
	}{
		{
			name:    "multi-page ending with partial page",
			perPage: 10,
			pages: map[int]string{
				1: makeBeerListJSON(1, 10),
				2: makeBeerListJSON(11, 10),
				3: makeBeerListJSON(21, 3),
			},
			wantCount:    23,
			wantRequests: 3,
		},
		{
			name:    "multi-page ending with empty page on exact page boundary",
			perPage: 10,
			pages: map[int]string{
				1: makeBeerListJSON(1, 10),
				2: makeBeerListJSON(11, 10),
				3: `[]`,
			},
			wantCount:    20,
			wantRequests: 3,
		},
		{
			name:    "empty response on first page",
			perPage: 10,
			pages: map[int]string{
				1: `[]`,
			},
			wantCount:    0,
			wantRequests: 1,
		},
		{
			name:    "single partial page",
			perPage: 80,
			pages: map[int]string{
				1: makeBeerListJSON(1, 5),
			},
			wantCount:    5,
			wantRequests: 1,
		},
		{
			name:    "http 500 on first page",
			perPage: 10,
			pages: map[int]string{
				1: `{"detail":"internal server error"}`,
			},
			statusByPage: map[int]int{
				1: http.StatusInternalServerError,
			},
			wantRequests:    1,
			wantErrContains: "HTTP 500",
		},
		{
			name:    "http 404 on second page",
			perPage: 10,
			pages: map[int]string{
				1: makeBeerListJSON(1, 10),
				2: `{"detail":"not found"}`,
			},
			statusByPage: map[int]int{
				2: http.StatusNotFound,
			},
			wantRequests:    2,
			wantErrContains: "HTTP 404",
		},
		{
			name:    "malformed json response",
			perPage: 10,
			pages: map[int]string{
				1: `[{"id": 1, "name": "broken"`,
			},
			wantRequests:    1,
			wantErrContains: "decode beers page 1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var reqCount atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				reqCount.Add(1)

				if r.URL.Path != "/api/v3/beers" {
					t.Errorf("unexpected path: got %q, want %q", r.URL.Path, "/api/v3/beers")
					http.NotFound(w, r)
					return
				}

				gotPerPage := r.URL.Query().Get("per_page")
				if gotPerPage != strconv.Itoa(tc.perPage) {
					t.Errorf("unexpected per_page query param: got %q, want %d", gotPerPage, tc.perPage)
				}

				page, err := strconv.Atoi(r.URL.Query().Get("page"))
				if err != nil {
					t.Errorf("invalid page query param %q: %v", r.URL.Query().Get("page"), err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				status := http.StatusOK
				if s, ok := tc.statusByPage[page]; ok {
					status = s
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				_, _ = w.Write([]byte(tc.pages[page]))
			}))
			defer srv.Close()

			src := source.NewHTTPSource(
				srv.URL+"/",
				source.WithHTTPClient(srv.Client()),
				source.WithPerPage(tc.perPage),
			)

			beers, err := src.LoadBeers(context.Background())
			if tc.wantErrContains != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErrContains)
				}
				if !strings.Contains(err.Error(), tc.wantErrContains) {
					t.Fatalf("error %q does not contain %q", err.Error(), tc.wantErrContains)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(beers) != tc.wantCount {
					t.Fatalf("beer count: got %d, want %d", len(beers), tc.wantCount)
				}
				for i, b := range beers {
					wantID := i + 1
					if b.ID != wantID {
						t.Fatalf("beers[%d].ID = %d, want %d", i, b.ID, wantID)
					}
				}
			}

			if got := int(reqCount.Load()); got != tc.wantRequests {
				t.Fatalf("request count: got %d, want %d", got, tc.wantRequests)
			}
		})
	}
}

func TestHTTPSource_LoadBeers_NullableFields(t *testing.T) {
	t.Parallel()

	const payload = `[
		{
			"id": 1,
			"name": "Full Beer",
			"tagline": "Everything populated.",
			"first_brewed": "04/2007",
			"description": "A complete recipe.",
			"image": "001.png",
			"abv": 6.0,
			"ibu": 60.5,
			"target_fg": 1010,
			"target_og": 1056.5,
			"ebc": 17,
			"srm": 8.5,
			"ph": 4.4,
			"attenuation_level": 82.14,
			"volume": {"value": 20, "unit": "litres"},
			"boil_volume": {"value": 25, "unit": "litres"},
			"method": {
				"mash_temp": [
					{"temp": {"value": 65, "unit": "celsius"}, "duration": 75}
				],
				"fermentation": {"temp": {"value": 19, "unit": "celsius"}},
				"twist": "Add orange zest"
			},
			"ingredients": {
				"malt": [{"name": "Extra Pale", "amount": {"value": 5.3, "unit": "kilograms"}}],
				"hops": [{"name": "Ahtanum", "amount": {"value": 17.5, "unit": "grams"}, "add": "start", "attribute": "bitter"}],
				"yeast": "Wyeast 1056 - American Ale"
			},
			"food_pairing": ["Spicy tacos", "Cheesecake"],
			"brewers_tips": "Submerge aroma hops.",
			"contributed_by": "Sam Mason <sammdec>"
		},
		{
			"id": 2,
			"name": "Sparse Beer",
			"tagline": "Nullable fields set to null.",
			"first_brewed": "2019",
			"description": "Recipe with null optional fields.",
			"image": null,
			"abv": 4.5,
			"ibu": null,
			"target_fg": null,
			"target_og": null,
			"ebc": null,
			"srm": null,
			"ph": null,
			"attenuation_level": null,
			"volume": {"value": 20, "unit": "litres"},
			"boil_volume": {"value": 25, "unit": "litres"},
			"method": {
				"mash_temp": [
					{"temp": {"value": null, "unit": "celsius"}, "duration": null}
				],
				"fermentation": {"temp": {"value": null, "unit": "celsius"}},
				"twist": null
			},
			"ingredients": {
				"malt": [],
				"hops": [{"name": "Cascade", "amount": {"value": 10, "unit": "grams"}, "add": "dry hop", "attribute": null}],
				"yeast": null
			},
			"food_pairing": ["Burgers"],
			"brewers_tips": "Keep it cool.",
			"contributed_by": "Ali Skinner <AliSkinner>"
		}
	]`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	src := source.NewHTTPSource(srv.URL, source.WithHTTPClient(srv.Client()))
	beers, err := src.LoadBeers(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(beers) != 2 {
		t.Fatalf("got %d beers, want 2", len(beers))
	}

	full := beers[0]
	if full.Image == nil || *full.Image != "001.png" {
		t.Errorf("full.Image = %v, want 001.png", full.Image)
	}
	if full.IBU == nil || *full.IBU != 60.5 {
		t.Errorf("full.IBU = %v, want 60.5", full.IBU)
	}
	if full.TargetOG == nil || *full.TargetOG != 1056.5 {
		t.Errorf("full.TargetOG = %v, want 1056.5", full.TargetOG)
	}
	if full.AttenuationLevel == nil || *full.AttenuationLevel != 82.14 {
		t.Errorf("full.AttenuationLevel = %v, want 82.14", full.AttenuationLevel)
	}
	if full.Method.Twist == nil || *full.Method.Twist != "Add orange zest" {
		t.Errorf("full.Method.Twist = %v, want Add orange zest", full.Method.Twist)
	}
	if len(full.Method.MashTemp) != 1 || full.Method.MashTemp[0].Duration == nil || *full.Method.MashTemp[0].Duration != 75 {
		t.Errorf("unexpected full.Method.MashTemp: %+v", full.Method.MashTemp)
	}
	if full.Ingredients.Yeast == nil || *full.Ingredients.Yeast != "Wyeast 1056 - American Ale" {
		t.Errorf("full.Ingredients.Yeast = %v", full.Ingredients.Yeast)
	}
	if len(full.Ingredients.Hops) != 1 || full.Ingredients.Hops[0].Attribute == nil || *full.Ingredients.Hops[0].Attribute != "bitter" {
		t.Errorf("unexpected full.Ingredients.Hops: %+v", full.Ingredients.Hops)
	}

	sparse := beers[1]
	if sparse.Image != nil {
		t.Errorf("sparse.Image = %v, want nil", sparse.Image)
	}
	if sparse.IBU != nil || sparse.TargetFG != nil || sparse.TargetOG != nil ||
		sparse.EBC != nil || sparse.SRM != nil || sparse.PH != nil || sparse.AttenuationLevel != nil {
		t.Errorf("expected all nullable numeric metrics to be nil on sparse beer, got %+v", sparse)
	}
	if sparse.Method.Twist != nil {
		t.Errorf("sparse.Method.Twist = %v, want nil", sparse.Method.Twist)
	}
	if sparse.Method.Fermentation.Temp.Value != nil {
		t.Errorf("sparse.Method.Fermentation.Temp.Value = %v, want nil", sparse.Method.Fermentation.Temp.Value)
	}
	if len(sparse.Method.MashTemp) != 1 || sparse.Method.MashTemp[0].Temp.Value != nil || sparse.Method.MashTemp[0].Duration != nil {
		t.Errorf("expected null mash temp value and duration, got %+v", sparse.Method.MashTemp)
	}
	if sparse.Ingredients.Yeast != nil {
		t.Errorf("sparse.Ingredients.Yeast = %v, want nil", sparse.Ingredients.Yeast)
	}
	if len(sparse.Ingredients.Hops) != 1 || sparse.Ingredients.Hops[0].Attribute != nil {
		t.Errorf("expected null hop attribute, got %+v", sparse.Ingredients.Hops)
	}
}

func TestHTTPSource_LoadBeers_ContextCancellation(t *testing.T) {
	t.Parallel()

	t.Run("canceled before first request", func(t *testing.T) {
		t.Parallel()

		var reqCount atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqCount.Add(1)
			_, _ = w.Write([]byte(`[]`))
		}))
		defer srv.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		src := source.NewHTTPSource(srv.URL, source.WithHTTPClient(srv.Client()))
		_, err := src.LoadBeers(ctx)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
		if reqCount.Load() != 0 {
			t.Fatalf("expected 0 HTTP requests, got %d", reqCount.Load())
		}
	})

	t.Run("canceled while waiting for slow second page", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("page") == "1" {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(makeBeerListJSON(1, 10)))
				return
			}
			cancel()
			select {
			case <-r.Context().Done():
			case <-time.After(2 * time.Second):
			}
		}))
		defer srv.Close()

		src := source.NewHTTPSource(
			srv.URL,
			source.WithHTTPClient(srv.Client()),
			source.WithPerPage(10),
		)

		_, err := src.LoadBeers(ctx)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	})
}

func makeBeerListJSON(startID, count int) string {
	beers := make([]model.Beer, 0, count)
	for i := 0; i < count; i++ {
		id := startID + i
		beers = append(beers, model.Beer{
			ID:          id,
			Name:        fmt.Sprintf("Beer %d", id),
			Tagline:     "Test Beer",
			FirstBrewed: "01/2020",
			Description: "Generated test beer.",
			ABV:         5.0,
			Volume:      model.Volume{Value: 20, Unit: "litres"},
			BoilVolume:  model.Volume{Value: 25, Unit: "litres"},
			FoodPairing: []string{"Pizza"},
		})
	}
	b, err := json.Marshal(beers)
	if err != nil {
		panic(err)
	}
	return string(b)
}
