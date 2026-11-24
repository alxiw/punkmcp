package service

import (
	"slices"
)

const catalogTopIngredientsLimit = 10

type metricAccumulator struct {
	count int
	min   float64
	max   float64
	sum   float64
}

func (m *metricAccumulator) add(v float64) {
	if m.count == 0 {
		m.min = v
		m.max = v
	} else {
		if v < m.min {
			m.min = v
		}
		if v > m.max {
			m.max = v
		}
	}
	m.sum += v
	m.count++
}

func (m *metricAccumulator) toStats() MetricStats {
	if m.count == 0 {
		return MetricStats{}
	}
	return MetricStats{
		Min:   m.min,
		Max:   m.max,
		Avg:   roundToDecimals(m.sum/float64(m.count), 2),
		Count: m.count,
	}
}

// GetCatalogStats computes catalog-wide metrics, top hops/yeasts, and yearly recipe counts in a single pass.
func (s *Service) GetCatalogStats() CatalogStats {
	all := s.allBeers()

	var (
		abvAcc metricAccumulator
		ibuAcc metricAccumulator
		ebcAcc metricAccumulator
	)

	hopCounter := newIngredientCounter()
	yeastCounter := newIngredientCounter()
	yearCounts := make(map[int]int)

	for _, b := range all {
		abvAcc.add(b.ABV)
		if b.IBU != nil {
			ibuAcc.add(*b.IBU)
		}
		if b.EBC != nil {
			ebcAcc.add(*b.EBC)
		}

		if len(b.Ingredients.Hops) > 0 {
			hopNames := make([]string, 0, len(b.Ingredients.Hops))
			for _, h := range b.Ingredients.Hops {
				hopNames = append(hopNames, h.Name)
			}
			hopCounter.addRecipe(hopNames)
		}

		if b.Ingredients.Yeast != nil {
			yeastCounter.addRecipe([]string{*b.Ingredients.Yeast})
		}

		if d, err := parseBrewDate(b.FirstBrewed); err == nil {
			yearCounts[d.Year]++
		}
	}

	topHops := hopCounter.toSortedStats("")
	if len(topHops) > catalogTopIngredientsLimit {
		topHops = topHops[:catalogTopIngredientsLimit]
	}

	topYeasts := yeastCounter.toSortedStats("")
	if len(topYeasts) > catalogTopIngredientsLimit {
		topYeasts = topYeasts[:catalogTopIngredientsLimit]
	}

	byYear := make([]YearCount, 0, len(yearCounts))
	for year, count := range yearCounts {
		byYear = append(byYear, YearCount{
			Year:  year,
			Count: count,
		})
	}
	slices.SortFunc(byYear, func(a, b YearCount) int {
		return compareInt(a.Year, b.Year)
	})

	return CatalogStats{
		TotalBeers: len(all),
		ABV:        abvAcc.toStats(),
		IBU:        ibuAcc.toStats(),
		EBC:        ebcAcc.toStats(),
		TopHops:    topHops,
		TopYeasts:  topYeasts,
		ByYear:     byYear,
	}
}
