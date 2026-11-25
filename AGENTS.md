# PunkMCP

Lightweight Go 1.25+ Model Context Protocol (MCP) server for **PunkAPI** (415 BrewDog DIY Dog v8 beer recipes), built on `github.com/modelcontextprotocol/go-sdk`. Runs over `stdio` or Streamable HTTP (`/mcp`, `/healthz`).

## Architecture

* `cmd/punkmcp/` — Entrypoint, CLI flags / env config (`MCP_TRANSPORT`, `MCP_ADDR`, `PUNKAPI_BASE_URL`, `CACHE_TTL`), graceful shutdown.
* `internal/model/` — Upstream PunkAPI JSON models and compact MCP response DTOs.
* `internal/source/` — `DataSource` interface and paginated `HTTPSource` client (`https://punkapi.dev` by default; `http://punkapi:8000` in Docker).
* `internal/cache/` — Thread-safe (`sync.RWMutex`) in-memory catalog store with startup preload and background TTL refresh.
* `internal/service/` — Pure business logic decoupled from MCP (search, filtering, sorting, ingredients, comparison, similarity, stats).
* `internal/mcp/` — MCP server, typed tool handlers (`mcp.AddTool`), and Streamable HTTP / `/healthz` wiring.
* `deploy/nginx/` — Nginx reverse proxy snippet for `/mcp`.

## Implemented Tools (7)

`search_beers`, `get_beer_recipe`, `get_random_beer`, `list_ingredients`, `compare_beers`, `find_similar_beers`, `get_catalog_stats`.

## Engineering Invariants

* **Upstream contract:** Verify any API changes against `https://punkapi.dev/api/v3/openapi.json` (`https://punkapi.dev/docs`). Nullable fields (`ibu`, `ebc`, `srm`, `ph`, `attenuation_level`, `twist`, `image`) must remain pointer types (`*float64`, `*string`, etc.).
* **Token efficiency:** Search, list, and similarity tools return compact DTOs with pagination; only `get_beer_recipe` and `get_random_beer` return full brewing schedules.
* **Safety & UX:** Never panic on invalid input. Return actionable hints (e.g., suggesting broader filters or `list_ingredients`) when queries yield no results.
* **Layering:** Keep business logic in `internal/service` free of MCP SDK dependencies.
* **Tests:** Add table-driven unit tests for all new logic; mock upstream HTTP calls with `net/http/httptest`.

## Verification

Run before committing (mirrors `.github/workflows/ci.yml`):

```bash
gofmt -l .
go vet ./...
go test -race -count=1 ./...
go build ./cmd/punkmcp
```