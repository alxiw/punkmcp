package mcp

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/alxiw/punkmcp/internal/model"
	"github.com/alxiw/punkmcp/internal/service"
)

// SearchBeersInput defines the typed arguments for the search_beers MCP tool.
type SearchBeersInput struct {
	Name         string   `json:"name,omitempty" jsonschema:"Case-insensitive substring match against the beer name"`
	ABVMin       *float64 `json:"abv_min,omitempty" jsonschema:"Minimum alcohol by volume (ABV) percentage, inclusive"`
	ABVMax       *float64 `json:"abv_max,omitempty" jsonschema:"Maximum alcohol by volume (ABV) percentage, inclusive"`
	IBUMin       *float64 `json:"ibu_min,omitempty" jsonschema:"Minimum International Bitterness Units (IBU), inclusive"`
	IBUMax       *float64 `json:"ibu_max,omitempty" jsonschema:"Maximum International Bitterness Units (IBU), inclusive"`
	EBCMin       *float64 `json:"ebc_min,omitempty" jsonschema:"Minimum European Brewery Convention (EBC) color value, inclusive"`
	EBCMax       *float64 `json:"ebc_max,omitempty" jsonschema:"Maximum European Brewery Convention (EBC) color value, inclusive"`
	Hops         string   `json:"hops,omitempty" jsonschema:"Case-insensitive substring match against hop names (call list_ingredients to discover exact names)"`
	Malt         string   `json:"malt,omitempty" jsonschema:"Case-insensitive substring match against malt names (call list_ingredients to discover exact names)"`
	Yeast        string   `json:"yeast,omitempty" jsonschema:"Case-insensitive substring match against yeast strain names (call list_ingredients to discover exact names)"`
	Food         string   `json:"food,omitempty" jsonschema:"Case-insensitive substring match against recommended food pairings"`
	BrewedBefore string   `json:"brewed_before,omitempty" jsonschema:"Filter beers first brewed on or before this date in MM/YYYY or YYYY format (e.g. 04/2015 or 2015)"`
	BrewedAfter  string   `json:"brewed_after,omitempty" jsonschema:"Filter beers first brewed on or after this date in MM/YYYY or YYYY format (e.g. 01/2010 or 2010)"`
	SortBy       string   `json:"sort_by,omitempty" jsonschema:"Sort field: id (default), abv, ibu, ebc, first_brewed, or name"`
	Order        string   `json:"order,omitempty" jsonschema:"Sort direction: asc (default) or desc"`
	Limit        int      `json:"limit,omitempty" jsonschema:"Maximum number of compact beer summaries to return (default 10, max 25)"`
	Offset       int      `json:"offset,omitempty" jsonschema:"Number of matching results to skip for pagination (default 0)"`
}

// GetBeerRecipeInput defines the typed arguments for the get_beer_recipe MCP tool.
type GetBeerRecipeInput struct {
	ID int `json:"id" jsonschema:"Numeric ID of the beer recipe to retrieve"`
}

// GetRandomBeerInput defines the typed arguments for the get_random_beer MCP tool.
type GetRandomBeerInput struct {
	ABVMin       *float64 `json:"abv_min,omitempty" jsonschema:"Minimum alcohol by volume (ABV) percentage, inclusive"`
	ABVMax       *float64 `json:"abv_max,omitempty" jsonschema:"Maximum alcohol by volume (ABV) percentage, inclusive"`
	IBUMin       *float64 `json:"ibu_min,omitempty" jsonschema:"Minimum International Bitterness Units (IBU), inclusive"`
	IBUMax       *float64 `json:"ibu_max,omitempty" jsonschema:"Maximum International Bitterness Units (IBU), inclusive"`
	EBCMin       *float64 `json:"ebc_min,omitempty" jsonschema:"Minimum European Brewery Convention (EBC) color value, inclusive"`
	EBCMax       *float64 `json:"ebc_max,omitempty" jsonschema:"Maximum European Brewery Convention (EBC) color value, inclusive"`
	Food         string   `json:"food,omitempty" jsonschema:"Case-insensitive substring match against recommended food pairings"`
	BrewedBefore string   `json:"brewed_before,omitempty" jsonschema:"Filter beers first brewed on or before this date in MM/YYYY or YYYY format (e.g. 04/2015 or 2015)"`
	BrewedAfter  string   `json:"brewed_after,omitempty" jsonschema:"Filter beers first brewed on or after this date in MM/YYYY or YYYY format (e.g. 01/2010 or 2010)"`
}

// ListIngredientsInput defines the typed arguments for the list_ingredients MCP tool.
type ListIngredientsInput struct {
	Type  string `json:"type" jsonschema:"Ingredient category to list: hops, malt, or yeast"`
	Query string `json:"query,omitempty" jsonschema:"Optional case-insensitive substring filter on ingredient names"`
	Limit int    `json:"limit,omitempty" jsonschema:"Maximum number of unique ingredients to return (default 20, max 100)"`
}

// CompareBeersInput defines the typed arguments for the compare_beers MCP tool.
type CompareBeersInput struct {
	IDs []int `json:"ids" jsonschema:"List of 2 to 4 unique numeric beer IDs to compare side by side"`
}

// CompareBeersOutput mirrors service.BeerComparison using string map keys
// for JSON Schema 2020-12 compatibility while producing identical JSON output.
type CompareBeersOutput struct {
	Beers        []service.BeerComparisonItem         `json:"beers"`
	Shared       service.SharedIngredients            `json:"shared"`
	UniqueByBeer map[string]service.UniqueIngredients `json:"unique_by_beer"`
}

// FindSimilarBeersInput defines the typed arguments for the find_similar_beers MCP tool.
type FindSimilarBeersInput struct {
	ID    int `json:"id" jsonschema:"Numeric ID of the target beer to find similar recipes for"`
	Limit int `json:"limit,omitempty" jsonschema:"Maximum number of similar beers to return (default 5, max 15)"`
}

// GetCatalogStatsInput defines the empty input object for the get_catalog_stats MCP tool.
type GetCatalogStatsInput struct{}

func registerTools(server *gomcp.Server, svc *service.Service) {
	readOnlyIdempotent := &gomcp.ToolAnnotations{
		ReadOnlyHint:    true,
		DestructiveHint: boolPtr(false),
		IdempotentHint:  true,
		OpenWorldHint:   boolPtr(false),
	}

	readOnlyNonIdempotent := &gomcp.ToolAnnotations{
		ReadOnlyHint:    true,
		DestructiveHint: boolPtr(false),
		IdempotentHint:  false,
		OpenWorldHint:   boolPtr(false),
	}

	gomcp.AddTool(server, &gomcp.Tool{
		Name:        "search_beers",
		Title:       "Search Beers",
		Description: "Search and filter the BrewDog DIY Dog / PunkAPI catalog by name, ABV/IBU/EBC ranges, hops, malt, yeast, food pairing, or brew date. Returns compact beer summaries with total_found and pagination metadata (call get_beer_recipe with a beer ID for full brewing details). Use list_ingredients to discover exact hop, malt, or yeast names.",
		Annotations: readOnlyIdempotent,
	}, func(_ context.Context, _ *gomcp.CallToolRequest, in SearchBeersInput) (*gomcp.CallToolResult, service.SearchResult, error) {
		if svc == nil {
			return nil, service.SearchResult{}, errors.New("beer service is not initialized")
		}
		res, err := svc.SearchBeers(service.SearchParams{
			Name:         in.Name,
			ABVMin:       in.ABVMin,
			ABVMax:       in.ABVMax,
			IBUMin:       in.IBUMin,
			IBUMax:       in.IBUMax,
			EBCMin:       in.EBCMin,
			EBCMax:       in.EBCMax,
			Hops:         in.Hops,
			Malt:         in.Malt,
			Yeast:        in.Yeast,
			Food:         in.Food,
			BrewedBefore: in.BrewedBefore,
			BrewedAfter:  in.BrewedAfter,
			SortBy:       in.SortBy,
			Order:        in.Order,
			Limit:        in.Limit,
			Offset:       in.Offset,
		})
		if err != nil {
			return nil, service.SearchResult{}, mapServiceError(err, "")
		}
		return nil, res, nil
	})

	gomcp.AddTool(server, &gomcp.Tool{
		Name:        "get_beer_recipe",
		Title:       "Get Beer Recipe",
		Description: "Retrieve the complete brewing recipe for a single beer by its numeric ID, including full hop and malt schedules, yeast strain, mash temperatures, fermentation temperature, target gravity, pH, food pairings, and brewer's tips. Use search_beers first to discover beer IDs.",
		Annotations: readOnlyIdempotent,
	}, func(_ context.Context, _ *gomcp.CallToolRequest, in GetBeerRecipeInput) (*gomcp.CallToolResult, model.Beer, error) {
		if svc == nil {
			return nil, model.Beer{}, errors.New("beer service is not initialized")
		}
		beer, err := svc.GetBeerRecipe(in.ID)
		if err != nil {
			return nil, model.Beer{}, mapServiceError(err, "Use search_beers to find valid beer IDs.")
		}
		return nil, beer, nil
	})

	gomcp.AddTool(server, &gomcp.Tool{
		Name:        "get_random_beer",
		Title:       "Get Random Beer",
		Description: "Return one random full beer recipe from the catalog, optionally constrained by ABV/IBU/EBC ranges, food pairing, or brew date. Useful for recommendations, inspiration, or surprise beer picks.",
		Annotations: readOnlyNonIdempotent,
	}, func(_ context.Context, _ *gomcp.CallToolRequest, in GetRandomBeerInput) (*gomcp.CallToolResult, model.Beer, error) {
		if svc == nil {
			return nil, model.Beer{}, errors.New("beer service is not initialized")
		}
		beer, err := svc.GetRandomBeer(service.RandomParams{
			ABVMin:       in.ABVMin,
			ABVMax:       in.ABVMax,
			IBUMin:       in.IBUMin,
			IBUMax:       in.IBUMax,
			EBCMin:       in.EBCMin,
			EBCMax:       in.EBCMax,
			Food:         in.Food,
			BrewedBefore: in.BrewedBefore,
			BrewedAfter:  in.BrewedAfter,
		})
		if err != nil {
			return nil, model.Beer{}, mapServiceError(err, "Try broadening numeric ranges or removing some filters.")
		}
		return nil, beer, nil
	})

	gomcp.AddTool(server, &gomcp.Tool{
		Name:        "list_ingredients",
		Title:       "List Ingredients",
		Description: "List unique hops, malts, or yeasts used across the beer catalog along with recipe counts, sorted by popularity. Use this to discover exact ingredient names before filtering with search_beers.",
		Annotations: readOnlyIdempotent,
	}, func(_ context.Context, _ *gomcp.CallToolRequest, in ListIngredientsInput) (*gomcp.CallToolResult, service.IngredientListResult, error) {
		if svc == nil {
			return nil, service.IngredientListResult{}, errors.New("beer service is not initialized")
		}
		res, err := svc.ListIngredients(service.IngredientParams{
			Type:  in.Type,
			Query: in.Query,
			Limit: in.Limit,
		})
		if err != nil {
			return nil, service.IngredientListResult{}, mapServiceError(err, "")
		}
		return nil, res, nil
	})

	gomcp.AddTool(server, &gomcp.Tool{
		Name:        "compare_beers",
		Title:       "Compare Beers",
		Description: "Compare 2 to 4 beers side by side by their numeric IDs, showing key metrics (ABV, IBU, EBC, pH), deduplicated hops and malts, ingredients shared by all compared beers, and ingredients unique to each beer.",
		Annotations: readOnlyIdempotent,
	}, func(_ context.Context, _ *gomcp.CallToolRequest, in CompareBeersInput) (*gomcp.CallToolResult, CompareBeersOutput, error) {
		if svc == nil {
			return nil, CompareBeersOutput{}, errors.New("beer service is not initialized")
		}
		res, err := svc.CompareBeers(in.IDs)
		if err != nil {
			return nil, CompareBeersOutput{}, mapServiceError(err, "Use search_beers to find valid beer IDs.")
		}
		return nil, toCompareBeersOutput(res), nil
	})

	gomcp.AddTool(server, &gomcp.Tool{
		Name:        "find_similar_beers",
		Title:       "Find Similar Beers",
		Description: "Find the top-N most similar beers in the catalog to a given beer ID based on yeast, hop and malt overlap, and ABV/IBU/EBC proximity. Returns compact beer summaries with similarity scores (0 to 1) and match reasons.",
		Annotations: readOnlyIdempotent,
	}, func(_ context.Context, _ *gomcp.CallToolRequest, in FindSimilarBeersInput) (*gomcp.CallToolResult, service.SimilarBeersResult, error) {
		if svc == nil {
			return nil, service.SimilarBeersResult{}, errors.New("beer service is not initialized")
		}
		res, err := svc.FindSimilarBeers(in.ID, in.Limit)
		if err != nil {
			return nil, service.SimilarBeersResult{}, mapServiceError(err, "Use search_beers to find valid beer IDs.")
		}
		return nil, res, nil
	})

	gomcp.AddTool(server, &gomcp.Tool{
		Name:        "get_catalog_stats",
		Title:       "Get Catalog Stats",
		Description: "Get catalog-wide analytics across all loaded beer recipes, including total beer count, min/max/average ABV, IBU, and EBC, top 10 hops and yeasts, and yearly recipe distribution.",
		Annotations: readOnlyIdempotent,
	}, func(_ context.Context, _ *gomcp.CallToolRequest, _ GetCatalogStatsInput) (*gomcp.CallToolResult, service.CatalogStats, error) {
		if svc == nil {
			return nil, service.CatalogStats{}, errors.New("beer service is not initialized")
		}
		return nil, svc.GetCatalogStats(), nil
	})
}

func toCompareBeersOutput(in service.BeerComparison) CompareBeersOutput {
	unique := make(map[string]service.UniqueIngredients, len(in.UniqueByBeer))
	for id, u := range in.UniqueByBeer {
		unique[strconv.Itoa(id)] = u
	}
	return CompareBeersOutput{
		Beers:        in.Beers,
		Shared:       in.Shared,
		UniqueByBeer: unique,
	}
}

func mapServiceError(err error, notFoundHint string) error {
	switch {
	case errors.Is(err, service.ErrNotFound):
		if notFoundHint != "" {
			return fmt.Errorf("%s. %s", err.Error(), notFoundHint)
		}
		return fmt.Errorf("%s", err.Error())
	case errors.Is(err, service.ErrInvalidInput):
		return fmt.Errorf("%s", err.Error())
	default:
		return err
	}
}

func boolPtr(v bool) *bool {
	return &v
}
