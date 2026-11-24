package service_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/alxiw/punkmcp/internal/model"
	"github.com/alxiw/punkmcp/internal/service"
)

func TestListIngredients(t *testing.T) {
	t.Parallel()

	beers := []model.Beer{
		{
			ID: 1,
			Ingredients: model.Ingredients{
				// Chinook repeated 3 times with spacing/casing differences -> counts as 1 recipe!
				Hops:  hops("Chinook", " chinook ", "CHINOOK", "Cascade", "Simcoe"),
				Malt:  malts("Extra Pale", "extra pale", "Caramalt"),
				Yeast: str("Wyeast 1056 - American Ale"),
			},
		},
		{
			ID: 2,
			Ingredients: model.Ingredients{
				Hops:  hops("Cascade", "Cascade", "Amarillo", "  "),
				Malt:  malts("Extra Pale", "Munich"),
				Yeast: str("  wyeast 1056 - american ale "),
			},
		},
		{
			ID: 3,
			Ingredients: model.Ingredients{
				Hops:  hops("cascade", "Simcoe"),
				Malt:  malts("Caramalt", "Chocolate"),
				Yeast: str("Wyeast 1056 - American Ale"),
			},
		},
		{
			ID: 4,
			Ingredients: model.Ingredients{
				Hops:  hops("Saaz"),
				Malt:  malts("Pilsner"),
				Yeast: nil, // Nil yeast must be ignored
			},
		},
		{
			ID: 5,
			Ingredients: model.Ingredients{
				Hops:  hops("Amarillo"),
				Malt:  malts("Extra Pale"),
				Yeast: str("Wyeast 1272 - American Ale II"),
			},
		},
	}

	svc := service.New(newMemStore(beers...))

	tests := []struct {
		name            string
		params          service.IngredientParams
		wantType        string
		wantTotalUnique int
		wantIngredients []service.IngredientStat
		wantErr         error
	}{
		{
			name:            "hops deduplicates within recipes, groups case-insensitively, and sorts by count desc then name asc",
			params:          service.IngredientParams{Type: "hops"},
			wantType:        "hops",
			wantTotalUnique: 5,
			wantIngredients: []service.IngredientStat{
				{Name: "Cascade", RecipeCount: 3},
				{Name: "Amarillo", RecipeCount: 2},
				{Name: "Simcoe", RecipeCount: 2},
				{Name: "Chinook", RecipeCount: 1},
				{Name: "Saaz", RecipeCount: 1},
			},
		},
		{
			name:            "hops with query and limit",
			params:          service.IngredientParams{Type: "HOPS", Query: "a", Limit: 2},
			wantType:        "hops",
			wantTotalUnique: 3, // Cascade, Amarillo, Saaz
			wantIngredients: []service.IngredientStat{
				{Name: "Cascade", RecipeCount: 3},
				{Name: "Amarillo", RecipeCount: 2},
			},
		},
		{
			name:            "malt counts unique recipes",
			params:          service.IngredientParams{Type: "malt"},
			wantType:        "malt",
			wantTotalUnique: 5,
			wantIngredients: []service.IngredientStat{
				{Name: "Extra Pale", RecipeCount: 3},
				{Name: "Caramalt", RecipeCount: 2},
				{Name: "Chocolate", RecipeCount: 1},
				{Name: "Munich", RecipeCount: 1},
				{Name: "Pilsner", RecipeCount: 1},
			},
		},
		{
			name:            "yeast ignores nil and normalizes casing/whitespace using canonical display name",
			params:          service.IngredientParams{Type: "yeast"},
			wantType:        "yeast",
			wantTotalUnique: 2,
			wantIngredients: []service.IngredientStat{
				{Name: "Wyeast 1056 - American Ale", RecipeCount: 3},
				{Name: "Wyeast 1272 - American Ale II", RecipeCount: 1},
			},
		},
		{
			name:            "query with no match returns empty non-nil slice",
			params:          service.IngredientParams{Type: "hops", Query: "zzz"},
			wantType:        "hops",
			wantTotalUnique: 0,
			wantIngredients: []service.IngredientStat{},
		},
		{
			name:    "invalid ingredient type returns ErrInvalidInput",
			params:  service.IngredientParams{Type: "water"},
			wantErr: service.ErrInvalidInput,
		},
		{
			name:    "empty ingredient type returns ErrInvalidInput",
			params:  service.IngredientParams{Type: ""},
			wantErr: service.ErrInvalidInput,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := svc.ListIngredients(tc.params)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.Type != tc.wantType {
				t.Errorf("Type = %q, want %q", got.Type, tc.wantType)
			}
			if got.TotalUnique != tc.wantTotalUnique {
				t.Errorf("TotalUnique = %d, want %d", got.TotalUnique, tc.wantTotalUnique)
			}
			if got.Ingredients == nil {
				t.Fatalf("expected non-nil Ingredients slice")
			}
			if !slices.Equal(got.Ingredients, tc.wantIngredients) {
				t.Fatalf("Ingredients = %+v, want %+v", got.Ingredients, tc.wantIngredients)
			}
		})
	}
}

func TestListIngredients_LimitClamping(t *testing.T) {
	t.Parallel()

	// Create 120 distinct hops across beers.
	hopList := make([]model.Hop, 120)
	for i := range hopList {
		hopList[i] = model.Hop{Name: "Hop-" + string(rune('A'+i/26)) + string(rune('A'+i%26))}
	}
	svc := service.New(newMemStore(model.Beer{
		ID:          1,
		Ingredients: model.Ingredients{Hops: hopList},
	}))

	// Default limit (0 -> 20)
	defRes, err := svc.ListIngredients(service.IngredientParams{Type: "hops"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if defRes.TotalUnique != 120 || len(defRes.Ingredients) != 20 {
		t.Fatalf("default limit: total=%d, len=%d, want 120, 20", defRes.TotalUnique, len(defRes.Ingredients))
	}

	// Max limit (500 -> clamped to 100)
	maxRes, err := svc.ListIngredients(service.IngredientParams{Type: "hops", Limit: 500})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if maxRes.TotalUnique != 120 || len(maxRes.Ingredients) != 100 {
		t.Fatalf("max limit: total=%d, len=%d, want 120, 100", maxRes.TotalUnique, len(maxRes.Ingredients))
	}
}
