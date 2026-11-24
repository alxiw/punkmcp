package service_test

import (
	"slices"
	"testing"

	"github.com/alxiw/punkmcp/internal/model"
	"github.com/alxiw/punkmcp/internal/service"
)

func TestGetCatalogStats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		beers        []model.Beer
		wantTotal    int
		wantABV      service.MetricStats
		wantIBU      service.MetricStats
		wantEBC      service.MetricStats
		wantTopHops  []service.IngredientStat
		wantTopYeast []service.IngredientStat
		wantByYear   []service.YearCount
	}{
		{
			name:         "empty catalog returns zeroed stats with non-nil slices",
			beers:        nil,
			wantTotal:    0,
			wantABV:      service.MetricStats{},
			wantIBU:      service.MetricStats{},
			wantEBC:      service.MetricStats{},
			wantTopHops:  []service.IngredientStat{},
			wantTopYeast: []service.IngredientStat{},
			wantByYear:   []service.YearCount{},
		},
		{
			name:      "mixed nil and non-nil metrics, repeated hops, and both date formats",
			beers:     catalogFixture(),
			wantTotal: 6,
			// ABVs: 5.6, 4.1, 10.5, 4.8, 5.6, 6.2 -> sum = 36.8, avg = 36.8/6 = 6.1333 -> 6.13
			wantABV: service.MetricStats{Min: 4.1, Max: 10.5, Avg: 6.13, Count: 6},
			// IBUs: 45, 41.5, 85, nil, 35, nil -> sum = 206.5, avg = 206.5/4 = 51.625 -> 51.63
			wantIBU: service.MetricStats{Min: 35, Max: 85, Avg: 51.63, Count: 4},
			// EBCs: 17, 15, 140, nil, nil, 30 -> sum = 202, avg = 202/4 = 50.5
			wantEBC: service.MetricStats{Min: 15, Max: 140, Avg: 50.5, Count: 4},
			wantTopHops: []service.IngredientStat{
				{Name: "Cascade", RecipeCount: 4},
				{Name: "Simcoe", RecipeCount: 2},
				{Name: "Ahtanum", RecipeCount: 1},
				{Name: "Amarillo", RecipeCount: 1},
				{Name: "Chinook", RecipeCount: 1},
				{Name: "Citra", RecipeCount: 1},
				{Name: "Hallertauer", RecipeCount: 1},
				{Name: "Magnum", RecipeCount: 1},
				{Name: "Mosaic", RecipeCount: 1},
				{Name: "Motueka", RecipeCount: 1},
			},
			wantTopYeast: []service.IngredientStat{
				{Name: "Wyeast 1056 - American Ale", RecipeCount: 4},
				{Name: "Wyeast 1272 - American Ale II", RecipeCount: 1},
			},
			// Dates: 04/2007, 04/2008, 2015, 11/2015, 06/2018, ""
			wantByYear: []service.YearCount{
				{Year: 2007, Count: 1},
				{Year: 2008, Count: 1},
				{Year: 2015, Count: 2},
				{Year: 2018, Count: 1},
			},
		},
		{
			name: "all nil IBU and EBC report zero count and zero min/max/avg",
			beers: []model.Beer{
				{
					ID:          1,
					ABV:         5.0,
					FirstBrewed: "2020",
				},
			},
			wantTotal:    1,
			wantABV:      service.MetricStats{Min: 5.0, Max: 5.0, Avg: 5.0, Count: 1},
			wantIBU:      service.MetricStats{},
			wantEBC:      service.MetricStats{},
			wantTopHops:  []service.IngredientStat{},
			wantTopYeast: []service.IngredientStat{},
			wantByYear:   []service.YearCount{{Year: 2020, Count: 1}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := service.New(newMemStore(tc.beers...))
			got := svc.GetCatalogStats()

			if got.TotalBeers != tc.wantTotal {
				t.Errorf("TotalBeers = %d, want %d", got.TotalBeers, tc.wantTotal)
			}
			if got.ABV != tc.wantABV {
				t.Errorf("ABV = %+v, want %+v", got.ABV, tc.wantABV)
			}
			if got.IBU != tc.wantIBU {
				t.Errorf("IBU = %+v, want %+v", got.IBU, tc.wantIBU)
			}
			if got.EBC != tc.wantEBC {
				t.Errorf("EBC = %+v, want %+v", got.EBC, tc.wantEBC)
			}
			if got.TopHops == nil || !slices.Equal(got.TopHops, tc.wantTopHops) {
				t.Errorf("TopHops = %+v, want %+v", got.TopHops, tc.wantTopHops)
			}
			if got.TopYeasts == nil || !slices.Equal(got.TopYeasts, tc.wantTopYeast) {
				t.Errorf("TopYeasts = %+v, want %+v", got.TopYeasts, tc.wantTopYeast)
			}
			if got.ByYear == nil || !slices.Equal(got.ByYear, tc.wantByYear) {
				t.Errorf("ByYear = %+v, want %+v", got.ByYear, tc.wantByYear)
			}
		})
	}
}
