package mcp

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/alxiw/punkmcp/internal/model"
	"github.com/alxiw/punkmcp/internal/service"
)

type mockBeerStore struct {
	beers []model.Beer
	byID  map[int]model.Beer
}

func newMockBeerStore(beers []model.Beer) *mockBeerStore {
	byID := make(map[int]model.Beer, len(beers))
	cloned := make([]model.Beer, len(beers))
	for i, b := range beers {
		c := b.Clone()
		cloned[i] = c
		byID[c.ID] = c
	}
	return &mockBeerStore{
		beers: cloned,
		byID:  byID,
	}
}

func (m *mockBeerStore) GetAll() []model.Beer {
	out := make([]model.Beer, len(m.beers))
	for i, b := range m.beers {
		out[i] = b.Clone()
	}
	return out
}

func (m *mockBeerStore) GetByID(id int) (model.Beer, bool) {
	b, ok := m.byID[id]
	if !ok {
		return model.Beer{}, false
	}
	return b.Clone(), true
}

func (m *mockBeerStore) Count() int {
	return len(m.beers)
}

func f64Ptr(v float64) *float64 { return &v }
func strPtr(v string) *string   { return &v }
func intPtr(v int) *int         { return &v }

func sampleBeers() []model.Beer {
	return []model.Beer{
		{
			ID:          1,
			Name:        "Punk IPA 2007 - 2010",
			Tagline:     "Post Modern Classic. Spiky. Tropical. Hoppy.",
			FirstBrewed: "04/2007",
			Description: "Our flagship IPA loaded with new-world hops.",
			ABV:         6.0,
			IBU:         f64Ptr(60),
			EBC:         f64Ptr(17),
			PH:          f64Ptr(4.4),
			Volume:      model.Volume{Value: 20, Unit: "litres"},
			BoilVolume:  model.Volume{Value: 25, Unit: "litres"},
			Method: model.Method{
				MashTemp: []model.MashTemp{
					{Temp: model.Temperature{Value: f64Ptr(65), Unit: "celsius"}, Duration: intPtr(75)},
				},
				Fermentation: model.Fermentation{
					Temp: model.Temperature{Value: f64Ptr(19), Unit: "celsius"},
				},
			},
			Ingredients: model.Ingredients{
				Malt: []model.Malt{
					{Name: "Extra Pale", Amount: model.Amount{Value: 5.3, Unit: "kilograms"}},
					{Name: "Caramalt", Amount: model.Amount{Value: 0.4, Unit: "kilograms"}},
				},
				Hops: []model.Hop{
					{Name: "Ahtanum", Amount: model.Amount{Value: 17.5, Unit: "grams"}, Add: "start", Attribute: strPtr("bitter")},
					{Name: "Chinook", Amount: model.Amount{Value: 15, Unit: "grams"}, Add: "middle", Attribute: strPtr("flavour")},
					{Name: "Simcoe", Amount: model.Amount{Value: 25, Unit: "grams"}, Add: "end", Attribute: strPtr("aroma")},
				},
				Yeast: strPtr("Wyeast 1056 - American Ale"),
			},
			FoodPairing:   []string{"Spicy carne asada", "Gorgonzola burger"},
			BrewersTips:   "Dry hop aggressively after primary fermentation.",
			ContributedBy: "Sam Mason <samjbmason>",
		},
		{
			ID:          2,
			Name:        "Trashy Blonde",
			Tagline:     "You Know You Shouldn't",
			FirstBrewed: "04/2008",
			Description: "A titillating, neurotic, peroxide punk of a Pale Ale.",
			ABV:         4.1,
			IBU:         f64Ptr(41.5),
			EBC:         f64Ptr(15),
			PH:          f64Ptr(4.4),
			Volume:      model.Volume{Value: 20, Unit: "litres"},
			BoilVolume:  model.Volume{Value: 25, Unit: "litres"},
			Method: model.Method{
				MashTemp: []model.MashTemp{
					{Temp: model.Temperature{Value: f64Ptr(69), Unit: "celsius"}, Duration: nil},
				},
				Fermentation: model.Fermentation{
					Temp: model.Temperature{Value: f64Ptr(18), Unit: "celsius"},
				},
			},
			Ingredients: model.Ingredients{
				Malt: []model.Malt{
					{Name: "Extra Pale", Amount: model.Amount{Value: 3.25, Unit: "kilograms"}},
					{Name: "Munich", Amount: model.Amount{Value: 0.2, Unit: "kilograms"}},
				},
				Hops: []model.Hop{
					{Name: "Amarillo", Amount: model.Amount{Value: 13.8, Unit: "grams"}, Add: "start", Attribute: strPtr("bitter")},
					{Name: "Simcoe", Amount: model.Amount{Value: 13.8, Unit: "grams"}, Add: "end", Attribute: strPtr("aroma")},
				},
				Yeast: strPtr("Wyeast 1056 - American Ale"),
			},
			FoodPairing:   []string{"Fresh crab with lemon", "Garlic butter dipping sauce"},
			BrewersTips:   "Be careful not to collect too much wort from the mash.",
			ContributedBy: "Sam Mason <samjbmason>",
		},
		{
			ID:          3,
			Name:        "Imperial Stout",
			Tagline:     "Dark Roast & Velvet.",
			FirstBrewed: "2012",
			Description: "Rich, dark imperial stout with roasted barley notes.",
			ABV:         10.5,
			IBU:         nil, // intentionally nil to exercise nullable JSON schema handling
			EBC:         f64Ptr(140),
			PH:          nil,
			Volume:      model.Volume{Value: 20, Unit: "litres"},
			BoilVolume:  model.Volume{Value: 25, Unit: "litres"},
			Method: model.Method{
				MashTemp: []model.MashTemp{
					{Temp: model.Temperature{Value: f64Ptr(67), Unit: "celsius"}, Duration: intPtr(60)},
				},
				Fermentation: model.Fermentation{
					Temp: model.Temperature{Value: f64Ptr(21), Unit: "celsius"},
				},
				Twist: strPtr("Add cocoa nibs in secondary"),
			},
			Ingredients: model.Ingredients{
				Malt: []model.Malt{
					{Name: "Maris Otter Extra Pale", Amount: model.Amount{Value: 8.5, Unit: "kilograms"}},
					{Name: "Roasted Barley", Amount: model.Amount{Value: 0.9, Unit: "kilograms"}},
				},
				Hops: []model.Hop{
					{Name: "Magnum", Amount: model.Amount{Value: 40, Unit: "grams"}, Add: "start", Attribute: strPtr("bitter")},
				},
				Yeast: strPtr("Wyeast 1272 - American Ale II"),
			},
			FoodPairing:   []string{"Dark chocolate mousse", "Smoked brisket"},
			BrewersTips:   "Age on oak chips for extra complexity.",
			ContributedBy: "BrewDog",
		},
	}
}

func setupInMemoryClient(t *testing.T, store *mockBeerStore) *gomcp.ClientSession {
	t.Helper()

	ctx := context.Background()
	svc := service.New(store, service.WithRandSource(rand.NewPCG(42, 7)))
	server := NewServer(svc, "1.0.0-test")

	serverTransport, clientTransport := gomcp.NewInMemoryTransports()

	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server.Connect: %v", err)
	}
	t.Cleanup(func() {
		_ = serverSession.Close()
	})

	client := gomcp.NewClient(&gomcp.Implementation{
		Name:    "punkmcp-test-client",
		Version: "1.0.0",
	}, nil)

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	t.Cleanup(func() {
		_ = clientSession.Close()
	})

	return clientSession
}

func extractTextContent(t *testing.T, res *gomcp.CallToolResult) string {
	t.Helper()
	if len(res.Content) == 0 {
		t.Fatalf("expected non-empty Content in CallToolResult")
	}
	tc, ok := res.Content[0].(*gomcp.TextContent)
	if !ok {
		t.Fatalf("expected first content item to be *mcp.TextContent, got %T", res.Content[0])
	}
	return tc.Text
}

func TestListTools(t *testing.T) {
	t.Parallel()

	session := setupInMemoryClient(t, newMockBeerStore(sampleBeers()))
	listRes, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	expectedTools := map[string]bool{
		"search_beers":       true,  // idempotent
		"get_beer_recipe":    true,  // idempotent
		"get_random_beer":    false, // non-idempotent
		"list_ingredients":   true,  // idempotent
		"compare_beers":      true,  // idempotent
		"find_similar_beers": true,  // idempotent
		"get_catalog_stats":  true,  // idempotent
	}

	if len(listRes.Tools) != len(expectedTools) {
		t.Fatalf("got %d tools, want %d", len(listRes.Tools), len(expectedTools))
	}

	seen := make(map[string]bool, len(listRes.Tools))
	for _, tool := range listRes.Tools {
		wantIdempotent, ok := expectedTools[tool.Name]
		if !ok {
			t.Errorf("unexpected tool registered: %q", tool.Name)
			continue
		}
		seen[tool.Name] = true

		if strings.TrimSpace(tool.Description) == "" {
			t.Errorf("tool %q has empty description", tool.Name)
		}
		if tool.InputSchema == nil {
			t.Errorf("tool %q has nil InputSchema", tool.Name)
		} else {
			schemaMap, ok := tool.InputSchema.(map[string]any)
			if !ok || schemaMap["type"] != "object" {
				t.Errorf("tool %q InputSchema is not a JSON object schema: %#v", tool.Name, tool.InputSchema)
			}
		}
		if tool.OutputSchema == nil {
			t.Errorf("tool %q has nil OutputSchema", tool.Name)
		}
		if tool.Annotations == nil {
			t.Errorf("tool %q has nil Annotations", tool.Name)
			continue
		}
		if !tool.Annotations.ReadOnlyHint {
			t.Errorf("tool %q ReadOnlyHint = false, want true", tool.Name)
		}
		if tool.Annotations.DestructiveHint == nil || *tool.Annotations.DestructiveHint {
			t.Errorf("tool %q DestructiveHint = %v, want false", tool.Name, tool.Annotations.DestructiveHint)
		}
		if tool.Annotations.OpenWorldHint == nil || *tool.Annotations.OpenWorldHint {
			t.Errorf("tool %q OpenWorldHint = %v, want false", tool.Name, tool.Annotations.OpenWorldHint)
		}
		if tool.Annotations.IdempotentHint != wantIdempotent {
			t.Errorf("tool %q IdempotentHint = %v, want %v", tool.Name, tool.Annotations.IdempotentHint, wantIdempotent)
		}
	}

	for name := range expectedTools {
		if !seen[name] {
			t.Errorf("expected tool %q was not listed", name)
		}
	}
}

func TestCallToolsEndToEnd(t *testing.T) {
	t.Parallel()

	session := setupInMemoryClient(t, newMockBeerStore(sampleBeers()))
	ctx := context.Background()

	t.Run("search_beers", func(t *testing.T) {
		res, err := session.CallTool(ctx, &gomcp.CallToolParams{
			Name: "search_beers",
			Arguments: map[string]any{
				"hops":    "simcoe",
				"sort_by": "abv",
				"order":   "desc",
				"limit":   5,
			},
		})
		if err != nil {
			t.Fatalf("CallTool(search_beers): %v", err)
		}
		if res.IsError {
			t.Fatalf("unexpected tool error: %s", extractTextContent(t, res))
		}
		if res.StructuredContent == nil {
			t.Fatalf("expected StructuredContent to be populated")
		}

		var out service.SearchResult
		if err := json.Unmarshal([]byte(extractTextContent(t, res)), &out); err != nil {
			t.Fatalf("unmarshal text content: %v", err)
		}
		if out.TotalFound != 2 || len(out.Beers) != 2 {
			t.Fatalf("got total_found=%d len=%d, want 2", out.TotalFound, len(out.Beers))
		}
		if out.Beers[0].ID != 1 || out.Beers[1].ID != 2 {
			t.Fatalf("unexpected beer order: got [%d, %d], want [1, 2]", out.Beers[0].ID, out.Beers[1].ID)
		}
	})

	t.Run("search_beers_empty_returns_hint", func(t *testing.T) {
		res, err := session.CallTool(ctx, &gomcp.CallToolParams{
			Name: "search_beers",
			Arguments: map[string]any{
				"name": "nonexistent-beer-xyz",
			},
		})
		if err != nil {
			t.Fatalf("CallTool(search_beers): %v", err)
		}
		if res.IsError {
			t.Fatalf("empty search should not set IsError")
		}

		var out service.SearchResult
		if err := json.Unmarshal([]byte(extractTextContent(t, res)), &out); err != nil {
			t.Fatalf("unmarshal text content: %v", err)
		}
		if out.TotalFound != 0 || out.Hint == "" {
			t.Fatalf("expected total_found=0 with helpful hint, got %+v", out)
		}
	})

	t.Run("get_beer_recipe", func(t *testing.T) {
		res, err := session.CallTool(ctx, &gomcp.CallToolParams{
			Name:      "get_beer_recipe",
			Arguments: map[string]any{"id": 3},
		})
		if err != nil {
			t.Fatalf("CallTool(get_beer_recipe): %v", err)
		}
		if res.IsError {
			t.Fatalf("unexpected tool error: %s", extractTextContent(t, res))
		}
		if res.StructuredContent == nil {
			t.Fatalf("expected StructuredContent to be populated")
		}

		var beer model.Beer
		if err := json.Unmarshal([]byte(extractTextContent(t, res)), &beer); err != nil {
			t.Fatalf("unmarshal text content: %v", err)
		}
		if beer.ID != 3 || beer.Name != "Imperial Stout" || beer.IBU != nil {
			t.Fatalf("unexpected beer recipe: %+v", beer)
		}
		if beer.Method.Twist == nil || *beer.Method.Twist != "Add cocoa nibs in secondary" {
			t.Fatalf("expected method twist to be preserved, got %v", beer.Method.Twist)
		}
	})

	t.Run("get_random_beer", func(t *testing.T) {
		res, err := session.CallTool(ctx, &gomcp.CallToolParams{
			Name: "get_random_beer",
			Arguments: map[string]any{
				"abv_min": 5.0,
				"abv_max": 7.0,
			},
		})
		if err != nil {
			t.Fatalf("CallTool(get_random_beer): %v", err)
		}
		if res.IsError {
			t.Fatalf("unexpected tool error: %s", extractTextContent(t, res))
		}
		if res.StructuredContent == nil {
			t.Fatalf("expected StructuredContent to be populated")
		}

		var beer model.Beer
		if err := json.Unmarshal([]byte(extractTextContent(t, res)), &beer); err != nil {
			t.Fatalf("unmarshal text content: %v", err)
		}
		if beer.ID != 1 {
			t.Fatalf("got random beer ID %d, want 1", beer.ID)
		}
	})

	t.Run("list_ingredients", func(t *testing.T) {
		res, err := session.CallTool(ctx, &gomcp.CallToolParams{
			Name: "list_ingredients",
			Arguments: map[string]any{
				"type":  "hops",
				"limit": 10,
			},
		})
		if err != nil {
			t.Fatalf("CallTool(list_ingredients): %v", err)
		}
		if res.IsError {
			t.Fatalf("unexpected tool error: %s", extractTextContent(t, res))
		}
		if res.StructuredContent == nil {
			t.Fatalf("expected StructuredContent to be populated")
		}

		var out service.IngredientListResult
		if err := json.Unmarshal([]byte(extractTextContent(t, res)), &out); err != nil {
			t.Fatalf("unmarshal text content: %v", err)
		}
		if out.Type != "hops" || out.TotalUnique != 5 || len(out.Ingredients) != 5 {
			t.Fatalf("unexpected list_ingredients result: %+v", out)
		}
		if out.Ingredients[0].Name != "Simcoe" || out.Ingredients[0].RecipeCount != 2 {
			t.Fatalf("expected top hop Simcoe (2), got %+v", out.Ingredients[0])
		}
	})

	t.Run("compare_beers", func(t *testing.T) {
		res, err := session.CallTool(ctx, &gomcp.CallToolParams{
			Name: "compare_beers",
			Arguments: map[string]any{
				"ids": []int{1, 2},
			},
		})
		if err != nil {
			t.Fatalf("CallTool(compare_beers): %v", err)
		}
		if res.IsError {
			t.Fatalf("unexpected tool error: %s", extractTextContent(t, res))
		}
		if res.StructuredContent == nil {
			t.Fatalf("expected StructuredContent to be populated")
		}

		var out CompareBeersOutput
		if err := json.Unmarshal([]byte(extractTextContent(t, res)), &out); err != nil {
			t.Fatalf("unmarshal text content: %v", err)
		}
		if len(out.Beers) != 2 || !out.Shared.SameYeast {
			t.Fatalf("unexpected compare_beers output: %+v", out)
		}
		if len(out.Shared.Hops) != 1 || out.Shared.Hops[0] != "Simcoe" {
			t.Fatalf("expected shared hop Simcoe, got %v", out.Shared.Hops)
		}
		if _, ok := out.UniqueByBeer["1"]; !ok {
			t.Fatalf("expected unique_by_beer entry for \"1\", got %+v", out.UniqueByBeer)
		}
		if _, ok := out.UniqueByBeer["2"]; !ok {
			t.Fatalf("expected unique_by_beer entry for \"2\", got %+v", out.UniqueByBeer)
		}
	})

	t.Run("find_similar_beers", func(t *testing.T) {
		res, err := session.CallTool(ctx, &gomcp.CallToolParams{
			Name: "find_similar_beers",
			Arguments: map[string]any{
				"id":    1,
				"limit": 2,
			},
		})
		if err != nil {
			t.Fatalf("CallTool(find_similar_beers): %v", err)
		}
		if res.IsError {
			t.Fatalf("unexpected tool error: %s", extractTextContent(t, res))
		}
		if res.StructuredContent == nil {
			t.Fatalf("expected StructuredContent to be populated")
		}

		var out service.SimilarBeersResult
		if err := json.Unmarshal([]byte(extractTextContent(t, res)), &out); err != nil {
			t.Fatalf("unmarshal text content: %v", err)
		}
		if out.Target.ID != 1 || len(out.Similar) != 2 {
			t.Fatalf("unexpected find_similar_beers result: %+v", out)
		}
		if out.Similar[0].ID != 2 || out.Similar[0].Score <= out.Similar[1].Score {
			t.Fatalf("expected beer 2 to rank above beer 3, got %+v", out.Similar)
		}
	})

	t.Run("get_catalog_stats", func(t *testing.T) {
		res, err := session.CallTool(ctx, &gomcp.CallToolParams{
			Name:      "get_catalog_stats",
			Arguments: map[string]any{},
		})
		if err != nil {
			t.Fatalf("CallTool(get_catalog_stats): %v", err)
		}
		if res.IsError {
			t.Fatalf("unexpected tool error: %s", extractTextContent(t, res))
		}
		if res.StructuredContent == nil {
			t.Fatalf("expected StructuredContent to be populated")
		}

		var out service.CatalogStats
		if err := json.Unmarshal([]byte(extractTextContent(t, res)), &out); err != nil {
			t.Fatalf("unmarshal text content: %v", err)
		}
		if out.TotalBeers != 3 || out.IBU.Count != 2 || len(out.ByYear) != 3 {
			t.Fatalf("unexpected catalog stats: %+v", out)
		}
	})
}

func TestToolErrorHandling(t *testing.T) {
	t.Parallel()

	session := setupInMemoryClient(t, newMockBeerStore(sampleBeers()))
	ctx := context.Background()

	tests := []struct {
		name        string
		tool        string
		args        map[string]any
		wantContain string
	}{
		{
			name:        "get_beer_recipe unknown id",
			tool:        "get_beer_recipe",
			args:        map[string]any{"id": 999},
			wantContain: "not found",
		},
		{
			name:        "get_random_beer no matches",
			tool:        "get_random_beer",
			args:        map[string]any{"abv_min": 50.0},
			wantContain: "not found",
		},
		{
			name:        "search_beers invalid date",
			tool:        "search_beers",
			args:        map[string]any{"brewed_after": "2020-01-01"},
			wantContain: "MM/YYYY",
		},
		{
			name:        "search_beers invalid sort_by",
			tool:        "search_beers",
			args:        map[string]any{"sort_by": "invalid_field"},
			wantContain: "unsupported sort_by",
		},
		{
			name:        "list_ingredients invalid type",
			tool:        "list_ingredients",
			args:        map[string]any{"type": "spices"},
			wantContain: "unsupported ingredient type",
		},
		{
			name:        "compare_beers single id",
			tool:        "compare_beers",
			args:        map[string]any{"ids": []int{1}},
			wantContain: "between 2 and 4",
		},
		{
			name:        "compare_beers unknown id",
			tool:        "compare_beers",
			args:        map[string]any{"ids": []int{1, 999}},
			wantContain: "not found",
		},
		{
			name:        "find_similar_beers unknown id",
			tool:        "find_similar_beers",
			args:        map[string]any{"id": 999},
			wantContain: "not found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := session.CallTool(ctx, &gomcp.CallToolParams{
				Name:      tc.tool,
				Arguments: tc.args,
			})
			if err != nil {
				t.Fatalf("expected MCP tool error result (IsError=true), got protocol error: %v", err)
			}
			if !res.IsError {
				t.Fatalf("expected IsError=true, got false (content: %s)", extractTextContent(t, res))
			}
			msg := extractTextContent(t, res)
			if !strings.Contains(strings.ToLower(msg), strings.ToLower(tc.wantContain)) {
				t.Fatalf("error message %q does not contain %q", msg, tc.wantContain)
			}
		})
	}
}

func TestHTTPHealthz(t *testing.T) {
	t.Parallel()

	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

	tests := []struct {
		name       string
		counter    CatalogCounter
		wantCode   int
		wantStatus string
		wantCount  int
	}{
		{
			name:       "healthy with beers loaded",
			counter:    newMockBeerStore(sampleBeers()),
			wantCode:   http.StatusOK,
			wantStatus: "ok",
			wantCount:  3,
		},
		{
			name:       "unhealthy when empty",
			counter:    newMockBeerStore(nil),
			wantCode:   http.StatusServiceUnavailable,
			wantStatus: "unavailable",
			wantCount:  0,
		},
		{
			name:       "unhealthy when counter is nil",
			counter:    nil,
			wantCode:   http.StatusServiceUnavailable,
			wantStatus: "unavailable",
			wantCount:  0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := service.New(newMockBeerStore(sampleBeers()))
			mcpSrv := NewServer(svc, "1.0.0-test")
			handler := NewHTTPHandler(mcpSrv, tc.counter, discardLogger)

			req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tc.wantCode {
				t.Fatalf("GET /healthz status = %d, want %d", rec.Code, tc.wantCode)
			}
			if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
				t.Fatalf("Content-Type = %q, want application/json", ct)
			}

			var body healthResponse
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("decode healthz body: %v", err)
			}
			if body.Status != tc.wantStatus || body.BeersLoaded != tc.wantCount {
				t.Fatalf("healthz body = %+v, want status=%q beers_loaded=%d", body, tc.wantStatus, tc.wantCount)
			}
		})
	}
}

func TestStreamableHTTPEndpoint(t *testing.T) {
	t.Parallel()

	store := newMockBeerStore(sampleBeers())
	svc := service.New(store, service.WithRandSource(rand.NewPCG(42, 7)))
	mcpSrv := NewServer(svc, "1.0.0-test")
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

	handler := NewHTTPHandler(mcpSrv, store, discardLogger)

	// Verify NewHTTPServer configures streaming-safe timeouts.
	httpSrv := NewHTTPServer(":0", handler)
	if httpSrv.ReadHeaderTimeout <= 0 {
		t.Errorf("ReadHeaderTimeout = %s, want > 0", httpSrv.ReadHeaderTimeout)
	}
	if httpSrv.IdleTimeout <= 0 {
		t.Errorf("IdleTimeout = %s, want > 0", httpSrv.IdleTimeout)
	}
	if httpSrv.WriteTimeout != 0 {
		t.Errorf("WriteTimeout = %s, want 0 for streaming MCP compatibility", httpSrv.WriteTimeout)
	}

	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)

	// Verify /healthz over the test server.
	resp, err := ts.Client().Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /healthz status = %d, want 200", resp.StatusCode)
	}

	// Verify Streamable HTTP MCP client connection and tool execution.
	ctx := context.Background()
	client := gomcp.NewClient(&gomcp.Implementation{
		Name:    "punkmcp-http-test-client",
		Version: "1.0.0",
	}, nil)

	transport := &gomcp.StreamableClientTransport{
		Endpoint:   ts.URL + "/mcp",
		HTTPClient: ts.Client(),
	}

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("StreamableClientTransport Connect: %v", err)
	}
	t.Cleanup(func() {
		_ = session.Close()
	})

	toolsRes, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("HTTP ListTools: %v", err)
	}
	if len(toolsRes.Tools) != 7 {
		t.Fatalf("HTTP ListTools returned %d tools, want 7", len(toolsRes.Tools))
	}

	callRes, err := session.CallTool(ctx, &gomcp.CallToolParams{
		Name:      "get_beer_recipe",
		Arguments: map[string]any{"id": 1},
	})
	if err != nil {
		t.Fatalf("HTTP CallTool(get_beer_recipe): %v", err)
	}
	if callRes.IsError {
		t.Fatalf("unexpected HTTP tool error: %s", extractTextContent(t, callRes))
	}

	var beer model.Beer
	if err := json.Unmarshal([]byte(extractTextContent(t, callRes)), &beer); err != nil {
		t.Fatalf("unmarshal HTTP recipe response: %v", err)
	}
	if beer.ID != 1 || beer.Name != "Punk IPA 2007 - 2010" {
		t.Fatalf("unexpected beer over HTTP: %+v", beer)
	}
}
