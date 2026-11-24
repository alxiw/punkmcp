package service

import (
	"fmt"
	"slices"
	"strings"

	"github.com/alxiw/punkmcp/internal/model"
)

// CompareBeers performs a side-by-side comparison of 2 to 4 unique beers by ID.
func (s *Service) CompareBeers(ids []int) (BeerComparison, error) {
	if len(ids) < 2 || len(ids) > 4 {
		return BeerComparison{}, fmt.Errorf("%w: compare_beers requires between 2 and 4 beer IDs, got %d", ErrInvalidInput, len(ids))
	}

	seenIDs := make(map[int]struct{}, len(ids))
	beers := make([]model.Beer, 0, len(ids))
	for _, id := range ids {
		if _, dup := seenIDs[id]; dup {
			return BeerComparison{}, fmt.Errorf("%w: duplicate beer id %d", ErrInvalidInput, id)
		}
		seenIDs[id] = struct{}{}

		beer, ok := s.beerByID(id)
		if !ok {
			return BeerComparison{}, fmt.Errorf("%w: beer %d", ErrNotFound, id)
		}
		beers = append(beers, beer)
	}

	hopCounter := newIngredientCounter()
	maltCounter := newIngredientCounter()

	beerHops := make([]map[string]struct{}, len(beers))
	beerMalts := make([]map[string]struct{}, len(beers))

	for i, b := range beers {
		rawHops := make([]string, 0, len(b.Ingredients.Hops))
		hopSet := make(map[string]struct{}, len(b.Ingredients.Hops))
		for _, h := range b.Ingredients.Hops {
			if key, display, ok := normalizeIngredientName(h.Name); ok {
				hopSet[key] = struct{}{}
				rawHops = append(rawHops, display)
			}
		}
		hopCounter.addRecipe(rawHops)
		beerHops[i] = hopSet

		rawMalts := make([]string, 0, len(b.Ingredients.Malt))
		maltSet := make(map[string]struct{}, len(b.Ingredients.Malt))
		for _, m := range b.Ingredients.Malt {
			if key, display, ok := normalizeIngredientName(m.Name); ok {
				maltSet[key] = struct{}{}
				rawMalts = append(rawMalts, display)
			}
		}
		maltCounter.addRecipe(rawMalts)
		beerMalts[i] = maltSet
	}

	items := make([]BeerComparisonItem, 0, len(beers))
	for i, b := range beers {
		foodPairing := []string{}
		if len(b.FoodPairing) > 0 {
			foodPairing = make([]string, len(b.FoodPairing))
			copy(foodPairing, b.FoodPairing)
		}

		items = append(items, BeerComparisonItem{
			ID:          b.ID,
			Name:        b.Name,
			Tagline:     b.Tagline,
			FirstBrewed: b.FirstBrewed,
			ABV:         b.ABV,
			IBU:         cloneFloat64Ptr(b.IBU),
			EBC:         cloneFloat64Ptr(b.EBC),
			PH:          cloneFloat64Ptr(b.PH),
			Yeast:       cloneStringPtr(b.Ingredients.Yeast),
			Hops:        sortedCanonicalNames(beerHops[i], hopCounter),
			Malts:       sortedCanonicalNames(beerMalts[i], maltCounter),
			FoodPairing: foodPairing,
		})
	}

	sharedHops := []string{}
	for key, bucket := range hopCounter.buckets {
		if bucket.recipeCount == len(beers) {
			sharedHops = append(sharedHops, hopCounter.canonicalName(key))
		}
	}
	slices.SortFunc(sharedHops, compareCaseInsensitive)

	sharedMalts := []string{}
	for key, bucket := range maltCounter.buckets {
		if bucket.recipeCount == len(beers) {
			sharedMalts = append(sharedMalts, maltCounter.canonicalName(key))
		}
	}
	slices.SortFunc(sharedMalts, compareCaseInsensitive)

	sameYeast := allSameYeast(beers)

	uniqueByBeer := make(map[int]UniqueIngredients, len(beers))
	for i, b := range beers {
		uHops := []string{}
		for key := range beerHops[i] {
			if hopCounter.buckets[key].recipeCount == 1 {
				uHops = append(uHops, hopCounter.canonicalName(key))
			}
		}
		slices.SortFunc(uHops, compareCaseInsensitive)

		uMalts := []string{}
		for key := range beerMalts[i] {
			if maltCounter.buckets[key].recipeCount == 1 {
				uMalts = append(uMalts, maltCounter.canonicalName(key))
			}
		}
		slices.SortFunc(uMalts, compareCaseInsensitive)

		uniqueByBeer[b.ID] = UniqueIngredients{
			Hops:  uHops,
			Malts: uMalts,
		}
	}

	return BeerComparison{
		Beers: items,
		Shared: SharedIngredients{
			Hops:      sharedHops,
			Malts:     sharedMalts,
			SameYeast: sameYeast,
		},
		UniqueByBeer: uniqueByBeer,
	}, nil
}

func sortedCanonicalNames(keys map[string]struct{}, counter *ingredientCounter) []string {
	out := make([]string, 0, len(keys))
	for key := range keys {
		out = append(out, counter.canonicalName(key))
	}
	slices.SortFunc(out, compareCaseInsensitive)
	return out
}

func allSameYeast(beers []model.Beer) bool {
	if len(beers) == 0 {
		return false
	}
	first, ok := normalizedYeast(beers[0].Ingredients.Yeast)
	if !ok {
		return false
	}
	for _, b := range beers[1:] {
		cur, ok := normalizedYeast(b.Ingredients.Yeast)
		if !ok || cur != first {
			return false
		}
	}
	return true
}

func normalizedYeast(y *string) (string, bool) {
	if y == nil {
		return "", false
	}
	key := strings.ToLower(strings.TrimSpace(*y))
	if key == "" {
		return "", false
	}
	return key, true
}
