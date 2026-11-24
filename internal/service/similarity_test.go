package service_test

import (
	"errors"
	"math"
	"slices"
	"testing"

	"github.com/alxiw/punkmcp/internal/model"
	"github.com/alxiw/punkmcp/internal/service"
)

func TestFindSimilarBeers(t *testing.T) {
	t.Parallel()

	beers := []model.Beer{
		{
			ID:   1,
			Name: "Target IPA",
			ABV:  6.0,
			IBU:  f64(60),
			EBC:  f64(20),
			Ingredients: model.Ingredients{
				Hops:  hops("Cascade", "Centennial", "Chinook"),
				Malt:  malts("Extra Pale", "Caramalt"),
				Yeast: str("Wyeast 1056"),
			},
		},
		{
			ID:   2,
			Name: "Exact Twin (missing IBU & EBC)",
			ABV:  6.0,
			IBU:  nil,
			EBC:  nil,
			Ingredients: model.Ingredients{
				Hops:  hops("cascade", "Centennial", "Chinook", "Cascade"),
				Malt:  malts("Extra Pale", "Caramalt"),
				Yeast: str("wyeast 1056"),
			},
		},
		{
			ID:   3,
			Name: "Close Pale Ale",
			ABV:  5.5,
			IBU:  f64(55),
			EBC:  f64(18),
			Ingredients: model.Ingredients{
				Hops:  hops("Cascade", "Centennial", "Simcoe"),
				Malt:  malts("Extra Pale", "Caramalt"),
				Yeast: str("Wyeast 1056"),
			},
		},
		{
			ID:   4,
			Name: "Distant Imperial Stout",
			ABV:  16.0,
			IBU:  f64(150),
			EBC:  f64(120),
			Ingredients: model.Ingredients{
				Hops:  hops("Magnum"),
				Malt:  malts("Roasted Barley", "Chocolate"),
				Yeast: str("Wyeast 1272"),
			},
		},
		{
			ID:   5,
			Name: "Tie Score Beer A",
			ABV:  16.0,
			IBU:  f64(150),
			EBC:  f64(120),
			Ingredients: model.Ingredients{
				Hops:  hops("Saaz"),
				Malt:  malts("Pilsner"),
				Yeast: str("Saflager S-23"),
			},
		},
	}

	svc := service.New(newMemStore(beers...))

	tests := []struct {
		name      string
		targetID  int
		limit     int
		wantIDs   []int
		wantErr   error
		checkFunc func(t *testing.T, res service.SimilarBeersResult)
	}{
		{
			name:     "ranks by similarity, excludes self, re-normalizes nil metrics, tie-breaks by ID asc",
			targetID: 1,
			limit:    0, // default 5
			wantIDs:  []int{2, 3, 4, 5},
			checkFunc: func(t *testing.T, res service.SimilarBeersResult) {
				t.Helper()
				if res.Target.ID != 1 {
					t.Errorf("Target.ID = %d, want 1", res.Target.ID)
				}
				// Beer 2 has identical yeast, hops, malts, ABV, and nil IBU/EBC -> re-normalized score must be 1.0!
				if res.Similar[0].Score != 1.0 {
					t.Errorf("Beer 2 score = %v, want 1.0 (nil metrics must be re-normalized)", res.Similar[0].Score)
				}
				wantReasons2 := []string{
					"same yeast",
					"3 shared hops: Cascade, Centennial, Chinook",
					"2 shared malts: Caramalt, Extra Pale",
					"close ABV",
				}
				if !slices.Equal(res.Similar[0].MatchReasons, wantReasons2) {
					t.Errorf("Beer 2 MatchReasons = %v, want %v", res.Similar[0].MatchReasons, wantReasons2)
				}

				// Beer 3 exact mathematical score verification:
				// yeast: 0.25 * 1 = 0.25
				// hops:  0.30 * (2/4) = 0.15
				// malt:  0.15 * (2/2) = 0.15
				// ABV:   0.12 * (1 - 0.5/10) = 0.114
				// IBU:   0.10 * (1 - 5/80) = 0.09375
				// EBC:   0.08 * (1 - 2/60) = 0.077333...
				// total = 0.835083... -> rounded to 3 decimals = 0.835
				if math.Abs(res.Similar[1].Score-0.835) > 1e-9 {
					t.Errorf("Beer 3 score = %v, want 0.835", res.Similar[1].Score)
				}
				wantReasons3 := []string{
					"same yeast",
					"2 shared hops: Cascade, Centennial",
					"2 shared malts: Caramalt, Extra Pale",
					"close ABV",
					"close IBU",
					"close EBC",
				}
				if !slices.Equal(res.Similar[1].MatchReasons, wantReasons3) {
					t.Errorf("Beer 3 MatchReasons = %v, want %v", res.Similar[1].MatchReasons, wantReasons3)
				}

				// Beer 4 and Beer 5 both have 0.0 score -> sorted by ID asc (4 then 5).
				if res.Similar[2].Score != 0 || res.Similar[3].Score != 0 {
					t.Errorf("expected 0 scores for distant beers, got %v and %v", res.Similar[2].Score, res.Similar[3].Score)
				}
				if res.Similar[2].MatchReasons == nil {
					t.Errorf("expected non-nil MatchReasons slice even when empty")
				}
			},
		},
		{
			name:     "respects explicit limit",
			targetID: 1,
			limit:    2,
			wantIDs:  []int{2, 3},
		},
		{
			name:     "missing target ID returns ErrNotFound",
			targetID: 999,
			limit:    5,
			wantErr:  service.ErrNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := svc.FindSimilarBeers(tc.targetID, tc.limit)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			gotIDs := make([]int, len(got.Similar))
			for i, item := range got.Similar {
				if item.ID == tc.targetID {
					t.Fatalf("target beer %d must not appear in Similar results", tc.targetID)
				}
				gotIDs[i] = item.ID
			}
			if !slices.Equal(gotIDs, tc.wantIDs) {
				t.Fatalf("Similar IDs = %v, want %v", gotIDs, tc.wantIDs)
			}
			if tc.checkFunc != nil {
				tc.checkFunc(t, got)
			}
		})
	}
}

func TestFindSimilarBeers_LimitClampedToMax15(t *testing.T) {
	t.Parallel()

	beers := make([]model.Beer, 25)
	for i := range beers {
		beers[i] = model.Beer{
			ID:   i + 1,
			Name: "Beer",
			ABV:  5.0,
		}
	}
	svc := service.New(newMemStore(beers...))

	got, err := svc.FindSimilarBeers(1, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Similar) != 15 {
		t.Fatalf("len(Similar) = %d, want 15 (clamped max)", len(got.Similar))
	}
}
