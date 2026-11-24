package service_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/alxiw/punkmcp/internal/service"
)

func TestCompareBeers_ValidationErrors(t *testing.T) {
	t.Parallel()

	svc := service.New(newMemStore(catalogFixture()...))

	tests := []struct {
		name        string
		ids         []int
		wantErr     error
		wantContain string
	}{
		{
			name:    "empty IDs returns ErrInvalidInput",
			ids:     nil,
			wantErr: service.ErrInvalidInput,
		},
		{
			name:    "single ID returns ErrInvalidInput",
			ids:     []int{1},
			wantErr: service.ErrInvalidInput,
		},
		{
			name:    "five IDs returns ErrInvalidInput",
			ids:     []int{1, 2, 3, 4, 5},
			wantErr: service.ErrInvalidInput,
		},
		{
			name:        "duplicate IDs returns ErrInvalidInput",
			ids:         []int{1, 2, 1},
			wantErr:     service.ErrInvalidInput,
			wantContain: "1",
		},
		{
			name:        "missing ID returns ErrNotFound with missing ID",
			ids:         []int{1, 999},
			wantErr:     service.ErrNotFound,
			wantContain: "999",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := svc.CompareBeers(tc.ids)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("CompareBeers(%v) err = %v, want %v", tc.ids, err, tc.wantErr)
			}
			if tc.wantContain != "" && !strings.Contains(err.Error(), tc.wantContain) {
				t.Fatalf("expected error %q to mention %q", err.Error(), tc.wantContain)
			}
		})
	}
}

func TestCompareBeers_ComparisonAccuracy(t *testing.T) {
	t.Parallel()

	svc := service.New(newMemStore(catalogFixture()...))

	tests := []struct {
		name          string
		ids           []int
		wantSharedHop []string
		wantSharedMlt []string
		wantSameYeast bool
		wantUnique    map[int]service.UniqueIngredients
	}{
		{
			name:          "two beers with same yeast (case-insensitive) and overlapping ingredients",
			ids:           []int{1, 5},
			wantSharedHop: []string{"Cascade"},
			wantSharedMlt: []string{"Extra Pale"},
			wantSameYeast: true,
			wantUnique: map[int]service.UniqueIngredients{
				1: {
					Hops:  []string{"Ahtanum", "Chinook", "Simcoe"},
					Malts: []string{"Caramalt"},
				},
				5: {
					Hops:  []string{"Citra", "Mosaic"},
					Malts: []string{"Oats"},
				},
			},
		},
		{
			name:          "three beers where some ingredients are in 2 of 3 (neither shared nor unique)",
			ids:           []int{1, 2, 3},
			wantSharedHop: []string{},
			wantSharedMlt: []string{},
			wantSameYeast: false,
			wantUnique: map[int]service.UniqueIngredients{
				1: {
					Hops:  []string{"Ahtanum", "Chinook"},
					Malts: []string{},
				},
				2: {
					Hops:  []string{"Amarillo", "Motueka"},
					Malts: []string{"Maris Otter Extra Pale"},
				},
				3: {
					Hops:  []string{"Magnum"},
					Malts: []string{"Chocolate", "Roasted Barley"},
				},
			},
		},
		{
			name:          "four beers including nil yeast sets SameYeast to false",
			ids:           []int{1, 2, 4, 6},
			wantSharedHop: []string{},
			wantSharedMlt: []string{},
			wantSameYeast: false,
			wantUnique: map[int]service.UniqueIngredients{
				1: {
					Hops:  []string{"Ahtanum", "Chinook"},
					Malts: []string{},
				},
				2: {
					Hops:  []string{"Amarillo", "Motueka"},
					Malts: []string{"Maris Otter Extra Pale"},
				},
				4: {
					Hops:  []string{"Hallertauer", "Saaz"},
					Malts: []string{"Pilsner"},
				},
				6: {
					Hops:  []string{},
					Malts: []string{},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := svc.CompareBeers(tc.ids)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(got.Beers) != len(tc.ids) {
				t.Fatalf("len(Beers) = %d, want %d", len(got.Beers), len(tc.ids))
			}
			for i, wantID := range tc.ids {
				if got.Beers[i].ID != wantID {
					t.Errorf("Beers[%d].ID = %d, want %d (must preserve input order)", i, got.Beers[i].ID, wantID)
				}
			}

			if !slices.Equal(got.Shared.Hops, tc.wantSharedHop) {
				t.Errorf("Shared.Hops = %v, want %v", got.Shared.Hops, tc.wantSharedHop)
			}
			if !slices.Equal(got.Shared.Malts, tc.wantSharedMlt) {
				t.Errorf("Shared.Malts = %v, want %v", got.Shared.Malts, tc.wantSharedMlt)
			}
			if got.Shared.SameYeast != tc.wantSameYeast {
				t.Errorf("Shared.SameYeast = %v, want %v", got.Shared.SameYeast, tc.wantSameYeast)
			}

			for id, wantU := range tc.wantUnique {
				gotU, ok := got.UniqueByBeer[id]
				if !ok {
					t.Fatalf("missing UniqueByBeer[%d]", id)
				}
				if !slices.Equal(gotU.Hops, wantU.Hops) {
					t.Errorf("UniqueByBeer[%d].Hops = %v, want %v", id, gotU.Hops, wantU.Hops)
				}
				if !slices.Equal(gotU.Malts, wantU.Malts) {
					t.Errorf("UniqueByBeer[%d].Malts = %v, want %v", id, gotU.Malts, wantU.Malts)
				}
			}
		})
	}
}
