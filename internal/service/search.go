package service

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/alxiw/punkmcp/internal/model"
)

const (
	defaultSearchLimit = 10
	maxSearchLimit     = 25
	emptySearchHint    = "No beers matched your filters. Try broadening numeric ranges, removing some filters, or calling list_ingredients to see available hops, malts, and yeasts."
)

// SearchBeers filters, sorts, and paginates the catalog according to params.
func (s *Service) SearchBeers(params SearchParams) (SearchResult, error) {
	sortBy, err := normalizeSortBy(params.SortBy)
	if err != nil {
		return SearchResult{}, err
	}
	desc, err := normalizeOrder(params.Order)
	if err != nil {
		return SearchResult{}, err
	}

	dateRange, err := parseBrewDateRange(params.BrewedAfter, params.BrewedBefore)
	if err != nil {
		return SearchResult{}, err
	}

	limit := clampLimit(params.Limit, defaultSearchLimit, maxSearchLimit)
	offset := params.Offset
	if offset < 0 {
		offset = 0
	}

	nameNeedle := strings.ToLower(strings.TrimSpace(params.Name))
	hopsNeedle := strings.ToLower(strings.TrimSpace(params.Hops))
	maltNeedle := strings.ToLower(strings.TrimSpace(params.Malt))
	yeastNeedle := strings.ToLower(strings.TrimSpace(params.Yeast))
	foodNeedle := strings.ToLower(strings.TrimSpace(params.Food))

	type datedBeer struct {
		beer    model.Beer
		date    brewDate
		hasDate bool
	}

	all := s.allBeers()
	matched := make([]datedBeer, 0, len(all))
	needDate := dateRange.active() || sortBy == "first_brewed"

	for _, b := range all {
		if nameNeedle != "" && !containsFold(b.Name, nameNeedle) {
			continue
		}
		if !matchesFloatRange(b.ABV, params.ABVMin, params.ABVMax) {
			continue
		}
		if !matchesPtrFloatRange(b.IBU, params.IBUMin, params.IBUMax) {
			continue
		}
		if !matchesPtrFloatRange(b.EBC, params.EBCMin, params.EBCMax) {
			continue
		}
		if hopsNeedle != "" && !matchesHops(b.Ingredients.Hops, hopsNeedle) {
			continue
		}
		if maltNeedle != "" && !matchesMalts(b.Ingredients.Malt, maltNeedle) {
			continue
		}
		if yeastNeedle != "" && !matchesYeast(b.Ingredients.Yeast, yeastNeedle) {
			continue
		}
		if foodNeedle != "" && !matchesAnyString(b.FoodPairing, foodNeedle) {
			continue
		}

		var (
			d       brewDate
			hasDate bool
		)
		if needDate {
			if parsed, parseErr := parseBrewDate(b.FirstBrewed); parseErr == nil {
				d = parsed
				hasDate = true
			}
		}

		if !dateRange.matches(d, hasDate) {
			continue
		}

		matched = append(matched, datedBeer{
			beer:    b,
			date:    d,
			hasDate: hasDate,
		})
	}

	slices.SortFunc(matched, func(a, b datedBeer) int {
		switch sortBy {
		case "abv":
			if c := cmp.Compare(a.beer.ABV, b.beer.ABV); c != 0 {
				return applyOrder(c, desc)
			}
		case "ibu":
			if c := compareNullableFloat(a.beer.IBU, b.beer.IBU, desc); c != 0 {
				return c
			}
		case "ebc":
			if c := compareNullableFloat(a.beer.EBC, b.beer.EBC, desc); c != 0 {
				return c
			}
		case "first_brewed":
			if c := compareNullableDate(a.date, a.hasDate, b.date, b.hasDate, desc); c != 0 {
				return c
			}
		case "name":
			if c := compareCaseInsensitive(a.beer.Name, b.beer.Name); c != 0 {
				return applyOrder(c, desc)
			}
		case "id":
			return applyOrder(compareInt(a.beer.ID, b.beer.ID), desc)
		}
		return compareInt(a.beer.ID, b.beer.ID)
	})

	totalFound := len(matched)
	result := SearchResult{
		TotalFound: totalFound,
		Limit:      limit,
		Offset:     offset,
		Beers:      []BeerSummary{},
	}

	if totalFound == 0 {
		result.Hint = emptySearchHint
		return result, nil
	}

	if offset >= totalFound {
		return result, nil
	}

	end := min(offset+limit, totalFound)
	result.Beers = make([]BeerSummary, 0, end-offset)
	for _, item := range matched[offset:end] {
		result.Beers = append(result.Beers, toBeerSummary(item.beer))
	}

	return result, nil
}

func normalizeSortBy(raw string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		return "id", nil
	}
	switch s {
	case "id", "abv", "ibu", "ebc", "first_brewed", "name":
		return s, nil
	default:
		return "", fmt.Errorf("%w: unsupported sort_by %q (allowed: id, abv, ibu, ebc, first_brewed, name)", ErrInvalidInput, raw)
	}
}

func normalizeOrder(raw string) (bool, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	switch s {
	case "", "asc":
		return false, nil
	case "desc":
		return true, nil
	default:
		return false, fmt.Errorf("%w: unsupported order %q (allowed: asc, desc)", ErrInvalidInput, raw)
	}
}

func clampLimit(limit, defaultVal, maxVal int) int {
	if limit <= 0 {
		return defaultVal
	}
	if limit > maxVal {
		return maxVal
	}
	return limit
}

func applyOrder(c int, desc bool) int {
	if desc {
		return -c
	}
	return c
}

func compareNullableFloat(a, b *float64, desc bool) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return 1
	}
	if b == nil {
		return -1
	}
	return applyOrder(cmp.Compare(*a, *b), desc)
}

func compareNullableDate(a brewDate, hasA bool, b brewDate, hasB bool, desc bool) int {
	if !hasA && !hasB {
		return 0
	}
	if !hasA {
		return 1
	}
	if !hasB {
		return -1
	}
	return applyOrder(compareBrewDate(a, b), desc)
}

func compareCaseInsensitive(a, b string) int {
	if c := cmp.Compare(strings.ToLower(a), strings.ToLower(b)); c != 0 {
		return c
	}
	return cmp.Compare(a, b)
}

func matchesFloatRange(val float64, minVal, maxVal *float64) bool {
	if minVal != nil && val < *minVal {
		return false
	}
	if maxVal != nil && val > *maxVal {
		return false
	}
	return true
}

func matchesPtrFloatRange(val *float64, minVal, maxVal *float64) bool {
	if minVal == nil && maxVal == nil {
		return true
	}
	if val == nil {
		return false
	}
	return matchesFloatRange(*val, minVal, maxVal)
}

func matchesHops(hops []model.Hop, needle string) bool {
	for _, h := range hops {
		if containsFold(h.Name, needle) {
			return true
		}
	}
	return false
}

func matchesMalts(malts []model.Malt, needle string) bool {
	for _, m := range malts {
		if containsFold(m.Name, needle) {
			return true
		}
	}
	return false
}

func matchesYeast(yeast *string, needle string) bool {
	return yeast != nil && containsFold(*yeast, needle)
}

func matchesAnyString(values []string, needle string) bool {
	for _, v := range values {
		if containsFold(v, needle) {
			return true
		}
	}
	return false
}

func containsFold(s, lowerNeedle string) bool {
	return strings.Contains(strings.ToLower(s), lowerNeedle)
}
