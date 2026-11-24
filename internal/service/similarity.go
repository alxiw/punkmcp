package service

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/alxiw/punkmcp/internal/model"
)

const (
	defaultSimilarLimit = 5
	maxSimilarLimit     = 15

	weightYeast = 0.25
	weightHops  = 0.30
	weightMalt  = 0.15
	weightABV   = 0.12
	weightIBU   = 0.10
	weightEBC   = 0.08

	abvScale = 10.0
	ibuScale = 80.0
	ebcScale = 60.0

	closeABVThreshold = 1.0
	closeIBUThreshold = 10.0
	closeEBCThreshold = 8.0

	maxSharedNamesInReason = 4
)

// FindSimilarBeers returns the top-N most similar beers to the target beer ID using
// a weighted combination of yeast, hop/malt Jaccard overlap, and numeric proximity.
func (s *Service) FindSimilarBeers(id int, limit int) (SimilarBeersResult, error) {
	target, ok := s.beerByID(id)
	if !ok {
		return SimilarBeersResult{}, fmt.Errorf("%w: beer %d", ErrNotFound, id)
	}

	limit = clampLimit(limit, defaultSimilarLimit, maxSimilarLimit)

	targetYeast, hasTargetYeast := normalizedYeast(target.Ingredients.Yeast)
	targetHops := uniqueHopMap(target.Ingredients.Hops)
	targetMalts := uniqueMaltMap(target.Ingredients.Malt)

	all := s.allBeers()
	scored := make([]SimilarBeerItem, 0, len(all))

	for _, cand := range all {
		if cand.ID == target.ID {
			continue
		}

		var (
			weightedSum float64
			totalWeight float64
			reasons     = []string{}
		)

		// 1. Yeast match (weight 0.25)
		candYeast, hasCandYeast := normalizedYeast(cand.Ingredients.Yeast)
		totalWeight += weightYeast
		if hasTargetYeast && hasCandYeast && targetYeast == candYeast {
			weightedSum += weightYeast
			reasons = append(reasons, "same yeast")
		}

		// 2. Hops Jaccard similarity (weight 0.30)
		candHops := uniqueHopMap(cand.Ingredients.Hops)
		hopSim, sharedHops := jaccardWithShared(targetHops, candHops)
		totalWeight += weightHops
		weightedSum += weightHops * hopSim
		if len(sharedHops) > 0 {
			reasons = append(reasons, formatSharedReason("hop", "hops", sharedHops))
		}

		// 3. Malt Jaccard similarity (weight 0.15)
		candMalts := uniqueMaltMap(cand.Ingredients.Malt)
		maltSim, sharedMalts := jaccardWithShared(targetMalts, candMalts)
		totalWeight += weightMalt
		weightedSum += weightMalt * maltSim
		if len(sharedMalts) > 0 {
			reasons = append(reasons, formatSharedReason("malt", "malts", sharedMalts))
		}

		// 4. Numeric proximity: ABV (always available), IBU & EBC (re-normalized when nil)
		abvDiff := math.Abs(target.ABV - cand.ABV)
		totalWeight += weightABV
		weightedSum += weightABV * numericProximity(abvDiff, abvScale)
		if abvDiff <= closeABVThreshold {
			reasons = append(reasons, "close ABV")
		}

		if target.IBU != nil && cand.IBU != nil {
			ibuDiff := math.Abs(*target.IBU - *cand.IBU)
			totalWeight += weightIBU
			weightedSum += weightIBU * numericProximity(ibuDiff, ibuScale)
			if ibuDiff <= closeIBUThreshold {
				reasons = append(reasons, "close IBU")
			}
		}

		if target.EBC != nil && cand.EBC != nil {
			ebcDiff := math.Abs(*target.EBC - *cand.EBC)
			totalWeight += weightEBC
			weightedSum += weightEBC * numericProximity(ebcDiff, ebcScale)
			if ebcDiff <= closeEBCThreshold {
				reasons = append(reasons, "close EBC")
			}
		}

		score := roundToDecimals(weightedSum/totalWeight, 3)

		scored = append(scored, SimilarBeerItem{
			BeerSummary:  toBeerSummary(cand),
			Score:        score,
			MatchReasons: reasons,
		})
	}

	slices.SortFunc(scored, func(a, b SimilarBeerItem) int {
		if c := cmp.Compare(b.Score, a.Score); c != 0 {
			return c
		}
		return compareInt(a.ID, b.ID)
	})

	if len(scored) > limit {
		scored = scored[:limit]
	}

	return SimilarBeersResult{
		Target:  toBeerSummary(target),
		Similar: scored,
	}, nil
}

func uniqueHopMap(hops []model.Hop) map[string]string {
	out := make(map[string]string, len(hops))
	for _, h := range hops {
		if key, display, ok := normalizeIngredientName(h.Name); ok {
			if _, exists := out[key]; !exists {
				out[key] = display
			}
		}
	}
	return out
}

func uniqueMaltMap(malts []model.Malt) map[string]string {
	out := make(map[string]string, len(malts))
	for _, m := range malts {
		if key, display, ok := normalizeIngredientName(m.Name); ok {
			if _, exists := out[key]; !exists {
				out[key] = display
			}
		}
	}
	return out
}

func jaccardWithShared(a, b map[string]string) (float64, []string) {
	if len(a) == 0 && len(b) == 0 {
		return 0, nil
	}

	var shared []string
	for key, display := range a {
		if _, ok := b[key]; ok {
			shared = append(shared, display)
		}
	}
	slices.SortFunc(shared, compareCaseInsensitive)

	unionSize := len(a) + len(b) - len(shared)
	if unionSize == 0 {
		return 0, nil
	}
	return float64(len(shared)) / float64(unionSize), shared
}

func numericProximity(diff, scale float64) float64 {
	if scale <= 0 {
		return 0
	}
	p := 1.0 - diff/scale
	if p < 0 {
		return 0
	}
	return p
}

func formatSharedReason(singular, plural string, names []string) string {
	n := len(names)
	label := plural
	if n == 1 {
		label = singular
	}

	displayed := names
	suffix := ""
	if n > maxSharedNamesInReason {
		displayed = names[:maxSharedNamesInReason]
		suffix = ", ..."
	}
	return fmt.Sprintf("%d shared %s: %s%s", n, label, strings.Join(displayed, ", "), suffix)
}

func roundToDecimals(v float64, decimals int) float64 {
	factor := math.Pow(10, float64(decimals))
	return math.Round(v*factor) / factor
}
