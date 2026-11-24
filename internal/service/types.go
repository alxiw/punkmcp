package service

// BeerSummary is a compact token-efficient representation of a beer recipe for search and similarity results.
type BeerSummary struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Tagline     string   `json:"tagline"`
	Description string   `json:"description"`
	ABV         float64  `json:"abv"`
	IBU         *float64 `json:"ibu"`
	EBC         *float64 `json:"ebc"`
	FirstBrewed string   `json:"first_brewed"`
	FoodPairing []string `json:"food_pairing"`
}

// SearchParams defines optional filtering, sorting, and pagination criteria for SearchBeers.
type SearchParams struct {
	Name         string   `json:"name,omitempty"`
	ABVMin       *float64 `json:"abv_min,omitempty"`
	ABVMax       *float64 `json:"abv_max,omitempty"`
	IBUMin       *float64 `json:"ibu_min,omitempty"`
	IBUMax       *float64 `json:"ibu_max,omitempty"`
	EBCMin       *float64 `json:"ebc_min,omitempty"`
	EBCMax       *float64 `json:"ebc_max,omitempty"`
	Hops         string   `json:"hops,omitempty"`
	Malt         string   `json:"malt,omitempty"`
	Yeast        string   `json:"yeast,omitempty"`
	Food         string   `json:"food,omitempty"`
	BrewedBefore string   `json:"brewed_before,omitempty"`
	BrewedAfter  string   `json:"brewed_after,omitempty"`
	SortBy       string   `json:"sort_by,omitempty"`
	Order        string   `json:"order,omitempty"`
	Limit        int      `json:"limit,omitempty"`
	Offset       int      `json:"offset,omitempty"`
}

// SearchResult holds paginated compact beer summaries and total match metadata.
type SearchResult struct {
	TotalFound int           `json:"total_found"`
	Limit      int           `json:"limit"`
	Offset     int           `json:"offset"`
	Beers      []BeerSummary `json:"beers"`
	Hint       string        `json:"hint,omitempty"`
}

// RandomParams defines optional filters for GetRandomBeer.
type RandomParams struct {
	ABVMin       *float64 `json:"abv_min,omitempty"`
	ABVMax       *float64 `json:"abv_max,omitempty"`
	IBUMin       *float64 `json:"ibu_min,omitempty"`
	IBUMax       *float64 `json:"ibu_max,omitempty"`
	EBCMin       *float64 `json:"ebc_min,omitempty"`
	EBCMax       *float64 `json:"ebc_max,omitempty"`
	Food         string   `json:"food,omitempty"`
	BrewedBefore string   `json:"brewed_before,omitempty"`
	BrewedAfter  string   `json:"brewed_after,omitempty"`
}

// IngredientParams defines filters for ListIngredients.
type IngredientParams struct {
	Type  string `json:"type"`
	Query string `json:"query,omitempty"`
	Limit int    `json:"limit,omitempty"`
}

// IngredientStat reports a canonical ingredient name and the number of unique recipes using it.
type IngredientStat struct {
	Name        string `json:"name"`
	RecipeCount int    `json:"recipe_count"`
}

// IngredientListResult contains unique ingredient usage counts for a requested ingredient category.
type IngredientListResult struct {
	Type        string           `json:"type"`
	TotalUnique int              `json:"total_unique"`
	Ingredients []IngredientStat `json:"ingredients"`
}

// BeerComparison holds a side-by-side comparison of 2 to 4 beers.
type BeerComparison struct {
	Beers        []BeerComparisonItem      `json:"beers"`
	Shared       SharedIngredients         `json:"shared"`
	UniqueByBeer map[int]UniqueIngredients `json:"unique_by_beer"`
}

// BeerComparisonItem summarizes key metrics and deduplicated ingredients for a compared beer.
type BeerComparisonItem struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Tagline     string   `json:"tagline"`
	FirstBrewed string   `json:"first_brewed"`
	ABV         float64  `json:"abv"`
	IBU         *float64 `json:"ibu"`
	EBC         *float64 `json:"ebc"`
	PH          *float64 `json:"ph"`
	Yeast       *string  `json:"yeast"`
	Hops        []string `json:"hops"`
	Malts       []string `json:"malts"`
	FoodPairing []string `json:"food_pairing"`
}

// SharedIngredients lists ingredients common to every beer in a comparison set.
type SharedIngredients struct {
	Hops      []string `json:"hops"`
	Malts     []string `json:"malts"`
	SameYeast bool     `json:"same_yeast"`
}

// UniqueIngredients lists hops and malts present in one beer and absent from all others in the comparison set.
type UniqueIngredients struct {
	Hops  []string `json:"hops"`
	Malts []string `json:"malts"`
}

// SimilarBeerItem pairs a compact beer summary with its normalized similarity score and match reasons.
type SimilarBeerItem struct {
	BeerSummary
	Score        float64  `json:"score"`
	MatchReasons []string `json:"match_reasons"`
}

// SimilarBeersResult contains the target beer summary and its closest catalog matches.
type SimilarBeersResult struct {
	Target  BeerSummary       `json:"target"`
	Similar []SimilarBeerItem `json:"similar"`
}

// MetricStats holds summary statistics for a numeric catalog metric.
type MetricStats struct {
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
	Avg   float64 `json:"avg"`
	Count int     `json:"count"`
}

// YearCount reports how many catalog recipes were first brewed in a given calendar year.
type YearCount struct {
	Year  int `json:"year"`
	Count int `json:"count"`
}

// CatalogStats provides catalog-wide aggregate analytics.
type CatalogStats struct {
	TotalBeers int              `json:"total_beers"`
	ABV        MetricStats      `json:"abv"`
	IBU        MetricStats      `json:"ibu"`
	EBC        MetricStats      `json:"ebc"`
	TopHops    []IngredientStat `json:"top_hops"`
	TopYeasts  []IngredientStat `json:"top_yeasts"`
	ByYear     []YearCount      `json:"by_year"`
}
