package service

import (
	"fmt"
	"slices"
	"strings"
)

const (
	defaultIngredientLimit = 20
	maxIngredientLimit     = 100
)

// ListIngredients returns unique ingredients of the requested type ("hops", "malt", or "yeast")
// counted by unique recipes, sorted by recipe count descending then name ascending.
func (s *Service) ListIngredients(params IngredientParams) (IngredientListResult, error) {
	typ := strings.ToLower(strings.TrimSpace(params.Type))
	switch typ {
	case "hops", "malt", "yeast":
	default:
		return IngredientListResult{}, fmt.Errorf("%w: unsupported ingredient type %q (allowed: hops, malt, yeast)", ErrInvalidInput, params.Type)
	}

	limit := clampLimit(params.Limit, defaultIngredientLimit, maxIngredientLimit)
	queryNeedle := strings.ToLower(strings.TrimSpace(params.Query))

	counter := newIngredientCounter()
	for _, b := range s.allBeers() {
		switch typ {
		case "hops":
			names := make([]string, 0, len(b.Ingredients.Hops))
			for _, h := range b.Ingredients.Hops {
				names = append(names, h.Name)
			}
			counter.addRecipe(names)
		case "malt":
			names := make([]string, 0, len(b.Ingredients.Malt))
			for _, m := range b.Ingredients.Malt {
				names = append(names, m.Name)
			}
			counter.addRecipe(names)
		case "yeast":
			if b.Ingredients.Yeast != nil {
				counter.addRecipe([]string{*b.Ingredients.Yeast})
			}
		}
	}

	stats := counter.toSortedStats(queryNeedle)
	totalUnique := len(stats)
	if len(stats) > limit {
		stats = stats[:limit]
	}

	return IngredientListResult{
		Type:        typ,
		TotalUnique: totalUnique,
		Ingredients: stats,
	}, nil
}

type ingredientBucket struct {
	recipeCount int
	spellings   map[string]int
}

type ingredientCounter struct {
	buckets map[string]*ingredientBucket
}

func newIngredientCounter() *ingredientCounter {
	return &ingredientCounter{
		buckets: make(map[string]*ingredientBucket),
	}
}

// addRecipe records all unique normalized ingredient names for a single recipe.
// Repeated occurrences of the same ingredient within one recipe are counted only once.
func (c *ingredientCounter) addRecipe(rawNames []string) {
	if len(rawNames) == 0 {
		return
	}

	seen := make(map[string]string, len(rawNames))
	for _, raw := range rawNames {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, exists := seen[key]; !exists {
			seen[key] = trimmed
		}
	}

	for key, display := range seen {
		bucket, ok := c.buckets[key]
		if !ok {
			bucket = &ingredientBucket{
				spellings: make(map[string]int),
			}
			c.buckets[key] = bucket
		}
		bucket.recipeCount++
		bucket.spellings[display]++
	}
}

func (c *ingredientCounter) canonicalName(key string) string {
	bucket, ok := c.buckets[key]
	if !ok {
		return ""
	}
	return pickCanonicalSpelling(bucket.spellings)
}

func (c *ingredientCounter) toSortedStats(queryNeedle string) []IngredientStat {
	out := make([]IngredientStat, 0, len(c.buckets))
	for key, bucket := range c.buckets {
		if queryNeedle != "" && !strings.Contains(key, queryNeedle) {
			continue
		}
		out = append(out, IngredientStat{
			Name:        pickCanonicalSpelling(bucket.spellings),
			RecipeCount: bucket.recipeCount,
		})
	}

	slices.SortFunc(out, func(a, b IngredientStat) int {
		if c := compareInt(b.RecipeCount, a.RecipeCount); c != 0 {
			return c
		}
		return compareCaseInsensitive(a.Name, b.Name)
	})

	return out
}

func pickCanonicalSpelling(spellings map[string]int) string {
	var (
		best      string
		bestCount int
	)
	for spelling, count := range spellings {
		if count > bestCount || (count == bestCount && (best == "" || spelling < best)) {
			best = spelling
			bestCount = count
		}
	}
	return best
}

func normalizeIngredientName(raw string) (key string, display string, ok bool) {
	display = strings.TrimSpace(raw)
	if display == "" {
		return "", "", false
	}
	return strings.ToLower(display), display, true
}
