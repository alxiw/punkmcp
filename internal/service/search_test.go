package service_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/alxiw/punkmcp/internal/service"
)

func TestSearchBeers_Filters(t *testing.T) {
	t.Parallel()

	svc := service.New(newMemStore(catalogFixture()...))

	tests := []struct {
		name      string
		params    service.SearchParams
		wantIDs   []int
		wantTotal int
		wantHint  bool
		wantErr   error
	}{
		{
			name:      "no filters returns all beers ordered by ID ascending",
			params:    service.SearchParams{},
			wantIDs:   []int{1, 2, 3, 4, 5, 6},
			wantTotal: 6,
		},
		{
			name:      "name matches beer.Name case-insensitively and ignores description matches",
			params:    service.SearchParams{Name: "PUNK"},
			wantIDs:   []int{1}, // Matches Punk IPA only, NOT Trashy Blonde whose description contains "punk"
			wantTotal: 1,
		},
		{
			name:      "name does not match tagline-only text",
			params:    service.SearchParams{Name: "roasty monster"},
			wantIDs:   []int{},
			wantTotal: 0,
			wantHint:  true,
		},
		{
			name:      "name does not match description-only text",
			params:    service.SearchParams{Name: "german style"},
			wantIDs:   []int{},
			wantTotal: 0,
			wantHint:  true,
		},
		{
			name: "ABV inclusive range boundaries",
			params: service.SearchParams{
				ABVMin: f64(4.8),
				ABVMax: f64(5.6),
			},
			wantIDs:   []int{1, 4, 5},
			wantTotal: 3,
		},
		{
			name: "IBU filter excludes beers with nil IBU",
			params: service.SearchParams{
				IBUMin: f64(0),
				IBUMax: f64(100),
			},
			wantIDs:   []int{1, 2, 3, 5},
			wantTotal: 4,
		},
		{
			name: "EBC filter excludes beers with nil EBC and respects upper bound",
			params: service.SearchParams{
				EBCMax: f64(30),
			},
			wantIDs:   []int{1, 2, 6},
			wantTotal: 3,
		},
		{
			name:      "hops filter matches case-insensitively",
			params:    service.SearchParams{Hops: "CASCADE"},
			wantIDs:   []int{1, 3, 5, 6},
			wantTotal: 4,
		},
		{
			name:      "malt filter matches substring",
			params:    service.SearchParams{Malt: "caramalt"},
			wantIDs:   []int{1, 2},
			wantTotal: 2,
		},
		{
			name:      "yeast filter matches case-insensitively and excludes nil yeast",
			params:    service.SearchParams{Yeast: "1056"},
			wantIDs:   []int{1, 2, 5, 6},
			wantTotal: 4,
		},
		{
			name:      "food filter matches substring",
			params:    service.SearchParams{Food: "chocolate"},
			wantIDs:   []int{3},
			wantTotal: 1,
		},
		{
			name:      "brewed_after MM/YYYY excludes earlier and undated beers",
			params:    service.SearchParams{BrewedAfter: "01/2015"},
			wantIDs:   []int{3, 4, 5},
			wantTotal: 3,
		},
		{
			name:      "brewed_after YYYY includes year-only and MM/YYYY in that year",
			params:    service.SearchParams{BrewedAfter: "2015"},
			wantIDs:   []int{3, 4, 5},
			wantTotal: 3,
		},
		{
			name:      "brewed_before YYYY includes entire year and earlier",
			params:    service.SearchParams{BrewedBefore: "2015"},
			wantIDs:   []int{1, 2, 3, 4},
			wantTotal: 4,
		},
		{
			name: "brewed_after and brewed_before same year filters single year",
			params: service.SearchParams{
				BrewedAfter:  "2015",
				BrewedBefore: "2015",
			},
			wantIDs:   []int{3, 4},
			wantTotal: 2,
		},
		{
			name: "brewed_after MM/YYYY mid-year excludes year-only and earlier months",
			params: service.SearchParams{
				BrewedAfter:  "06/2015",
				BrewedBefore: "12/2015",
			},
			wantIDs:   []int{4},
			wantTotal: 1,
		},
		{
			name: "combined multi-criteria AND filter",
			params: service.SearchParams{
				ABVMin:      f64(5.0),
				IBUMin:      f64(30),
				Hops:        "cascade",
				Malt:        "extra pale",
				Yeast:       "1056",
				Food:        "spicy",
				BrewedAfter: "2007",
			},
			wantIDs:   []int{1, 5},
			wantTotal: 2,
		},
		{
			name: "empty results return helpful hint mentioning list_ingredients",
			params: service.SearchParams{
				Hops: "nonexistent-hop",
			},
			wantIDs:   []int{},
			wantTotal: 0,
			wantHint:  true,
		},
		{
			name:    "invalid brewed_before format returns ErrInvalidInput",
			params:  service.SearchParams{BrewedBefore: "2015-05"},
			wantErr: service.ErrInvalidInput,
		},
		{
			name:    "invalid brewed_after month 13 returns ErrInvalidInput",
			params:  service.SearchParams{BrewedAfter: "13/2015"},
			wantErr: service.ErrInvalidInput,
		},
		{
			name:    "invalid brewed_after month 00 returns ErrInvalidInput",
			params:  service.SearchParams{BrewedAfter: "00/2015"},
			wantErr: service.ErrInvalidInput,
		},
		{
			name:    "invalid brewed_before non-numeric returns ErrInvalidInput",
			params:  service.SearchParams{BrewedBefore: "ab/2015"},
			wantErr: service.ErrInvalidInput,
		},
		{
			name:    "invalid sort_by returns ErrInvalidInput",
			params:  service.SearchParams{SortBy: "srm"},
			wantErr: service.ErrInvalidInput,
		},
		{
			name:    "invalid order returns ErrInvalidInput",
			params:  service.SearchParams{Order: "sideways"},
			wantErr: service.ErrInvalidInput,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := svc.SearchBeers(tc.params)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.TotalFound != tc.wantTotal {
				t.Fatalf("TotalFound = %d, want %d", got.TotalFound, tc.wantTotal)
			}

			gotIDs := make([]int, len(got.Beers))
			for i, b := range got.Beers {
				gotIDs[i] = b.ID
			}
			if !slices.Equal(gotIDs, tc.wantIDs) {
				t.Fatalf("Beers IDs = %v, want %v", gotIDs, tc.wantIDs)
			}

			if tc.wantHint {
				if got.Hint == "" || !strings.Contains(got.Hint, "list_ingredients") {
					t.Fatalf("expected helpful hint mentioning list_ingredients, got %q", got.Hint)
				}
			} else if got.Hint != "" {
				t.Fatalf("expected empty hint when results exist, got %q", got.Hint)
			}
		})
	}
}

func TestSearchBeers_SortingAndNilLast(t *testing.T) {
	t.Parallel()

	svc := service.New(newMemStore(catalogFixture()...))

	tests := []struct {
		name    string
		sortBy  string
		order   string
		wantIDs []int
	}{
		{
			name:    "default sort by id asc",
			wantIDs: []int{1, 2, 3, 4, 5, 6},
		},
		{
			name:    "sort by id desc",
			sortBy:  "id",
			order:   "desc",
			wantIDs: []int{6, 5, 4, 3, 2, 1},
		},
		{
			name:    "sort by abv asc with ID tie-breaker (1 and 5 both 5.6)",
			sortBy:  "abv",
			order:   "asc",
			wantIDs: []int{2, 4, 1, 5, 6, 3},
		},
		{
			name:    "sort by abv desc with ID asc tie-breaker (1 before 5)",
			sortBy:  "abv",
			order:   "desc",
			wantIDs: []int{3, 6, 1, 5, 4, 2},
		},
		{
			name:    "sort by ibu asc places nil IBU last (4, 6)",
			sortBy:  "ibu",
			order:   "asc",
			wantIDs: []int{5, 2, 1, 3, 4, 6},
		},
		{
			name:    "sort by ibu desc still places nil IBU last (4, 6)",
			sortBy:  "ibu",
			order:   "desc",
			wantIDs: []int{3, 1, 2, 5, 4, 6},
		},
		{
			name:    "sort by ebc asc places nil EBC last (4, 5)",
			sortBy:  "ebc",
			order:   "asc",
			wantIDs: []int{2, 1, 6, 3, 4, 5},
		},
		{
			name:    "sort by ebc desc still places nil EBC last (4, 5)",
			sortBy:  "ebc",
			order:   "desc",
			wantIDs: []int{3, 6, 1, 2, 4, 5},
		},
		{
			name:    "sort by first_brewed asc places undated last and year-only before month in same year",
			sortBy:  "first_brewed",
			order:   "asc",
			wantIDs: []int{1, 2, 3, 4, 5, 6},
		},
		{
			name:    "sort by first_brewed desc still places undated last",
			sortBy:  "first_brewed",
			order:   "desc",
			wantIDs: []int{5, 4, 3, 2, 1, 6},
		},
		{
			name:    "sort by name asc case-insensitively",
			sortBy:  "name",
			order:   "asc",
			wantIDs: []int{5, 3, 4, 1, 2, 6},
		},
		{
			name:    "sort by name desc case-insensitively",
			sortBy:  "name",
			order:   "desc",
			wantIDs: []int{6, 2, 1, 4, 3, 5},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := svc.SearchBeers(service.SearchParams{
				SortBy: tc.sortBy,
				Order:  tc.order,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			gotIDs := make([]int, len(got.Beers))
			for i, b := range got.Beers {
				gotIDs[i] = b.ID
			}
			if !slices.Equal(gotIDs, tc.wantIDs) {
				t.Fatalf("got IDs %v, want %v", gotIDs, tc.wantIDs)
			}
		})
	}
}

func TestSearchBeers_Pagination(t *testing.T) {
	t.Parallel()

	svc := service.New(newMemStore(catalogFixture()...))

	tests := []struct {
		name       string
		limit      int
		offset     int
		wantLimit  int
		wantOffset int
		wantIDs    []int
	}{
		{
			name:       "default limit 10 when unset",
			limit:      0,
			offset:     0,
			wantLimit:  10,
			wantOffset: 0,
			wantIDs:    []int{1, 2, 3, 4, 5, 6},
		},
		{
			name:       "negative limit defaults to 10 and negative offset clamps to 0",
			limit:      -5,
			offset:     -10,
			wantLimit:  10,
			wantOffset: 0,
			wantIDs:    []int{1, 2, 3, 4, 5, 6},
		},
		{
			name:       "limit above max clamps to 25",
			limit:      100,
			offset:     0,
			wantLimit:  25,
			wantOffset: 0,
			wantIDs:    []int{1, 2, 3, 4, 5, 6},
		},
		{
			name:       "middle page slice",
			limit:      2,
			offset:     2,
			wantLimit:  2,
			wantOffset: 2,
			wantIDs:    []int{3, 4},
		},
		{
			name:       "offset past end returns empty slice with TotalFound intact",
			limit:      5,
			offset:     10,
			wantLimit:  5,
			wantOffset: 10,
			wantIDs:    []int{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := svc.SearchBeers(service.SearchParams{
				Limit:  tc.limit,
				Offset: tc.offset,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Limit != tc.wantLimit {
				t.Errorf("Limit = %d, want %d", got.Limit, tc.wantLimit)
			}
			if got.Offset != tc.wantOffset {
				t.Errorf("Offset = %d, want %d", got.Offset, tc.wantOffset)
			}
			if got.TotalFound != 6 {
				t.Errorf("TotalFound = %d, want 6", got.TotalFound)
			}
			if got.Beers == nil {
				t.Fatalf("expected non-nil Beers slice")
			}

			gotIDs := make([]int, len(got.Beers))
			for i, b := range got.Beers {
				gotIDs[i] = b.ID
			}
			if !slices.Equal(gotIDs, tc.wantIDs) {
				t.Fatalf("got IDs %v, want %v", gotIDs, tc.wantIDs)
			}
		})
	}
}
