# PunkMCP

## Project Overview
**PunkMCP** is a lightweight, production-ready Model Context Protocol (MCP) server written in Go for **PunkAPI** (a digital archive of 415 BrewDog DIY Dog v8 beer recipes).

The service runs as a standalone Docker container alongside `PunkAPI` on a VPS (communicating over an internal Docker network) and can also be executed locally as a standalone binary via `stdio`.

## Upstream API & Data Contract
Before defining data models or writing the HTTP client, **always verify the exact JSON schema, nullable fields, endpoint paths, and pagination parameters** using the official project documentation:
- **Swagger Docs:** `https://punkapi.dev/docs`
- **OpenAPI:** `https://punkapi.dev/api/v3/openapi.json`
- **Repository & README:** `https://github.com/alxiw/punkapi`

Key data characteristics:
- The catalog contains ~415 mostly static beer recipes.
- Several numeric and string fields in the DIY Dog dataset can be `null` (e.g., `ibu`, `ebc`, `srm`, `ph`, `attenuation_level`, `twist`, `image`). Use pointer types (`*float64`, `*string`, etc.) where appropriate so JSON unmarshaling never fails and missing values are handled cleanly.
- The base URL must be configurable via `PUNKAPI_BASE_URL` (defaulting to `https://punkapi.dev`, overridable to an internal Docker URL like `http://punkapi:8080`).

## Tech Stack & Engineering Standards
- **Language:** Go 1.24+ (idiomatic, clean, minimal external dependencies).
- **MCP SDK:** Official `github.com/modelcontextprotocol/go-sdk`.
- **Project Layout:** Standard Go layout (`cmd/punkmcp`, `internal/...`).
- **Testing:** Table-driven unit tests (`table-driven tests`) for all business logic, caching, and data parsing. Mock external HTTP calls using `net/http/httptest`.
- **LLM Token Efficiency:** Search and list tools must return compact DTOs (e.g., `id`, `name`, `tagline`, `description`, `abv`, `ibu`, `ebc`, `first_brewed`, `food_pairing`) with `total_found` and pagination (`limit`/`offset`). Only `get_beer_recipe` returns the full recipe payload.
- **Error Handling & Hints:** Never panic on bad input. When search yields no results, return a helpful hint suggesting broader filters or `list_ingredients`.

## Target Architecture
- `cmd/punkmcp/` — Entrypoint, CLI flags / env config (`--transport=stdio|http`, `--addr` defaulting to `:8080`, `PUNKAPI_BASE_URL`, cache TTL), graceful shutdown.
- `internal/model/` — Domain structs for upstream PunkAPI JSON and compact MCP response DTOs.
- `internal/source/` — `DataSource` interface (`LoadBeers(ctx context.Context) ([]model.Beer, error)`) and `HTTPSource` implementation with pagination. (Designed so a `FileSource` can be added later without changing upper layers.)
- `internal/cache/` — Thread-safe (`sync.RWMutex`) in-memory store holding all recipes with startup preload and background TTL refresh.
- `internal/service/` — Pure business logic decoupled from MCP: filtering, sorting, ingredient aggregation, comparison, similarity scoring, and catalog statistics.
- `internal/mcp/` — MCP server setup, typed tool registration (`mcp.AddTool`), and transport wiring (`stdio` and `Streamable HTTP`).

## Planned MCP Tools (7 Total)
1. `search_beers` — Multi-criteria search & filter (`name`, `abv`/`ibu`/`ebc` ranges, `hops`, `malt`, `yeast`, `food`, `brewed_before`/`after`, `sort_by`, `order`, `limit`, `offset`). Returns compact cards + `total_found`.
2. `get_beer_recipe` — Full recipe details by `id`.
3. `get_random_beer` — Random beer with optional filters (`abv`/`ibu`/`ebc` ranges, `food`, `brewed_before`/`after`).
4. `list_ingredients` — Unique ingredients (`hops`, `malt`, `yeast`) with recipe counts and optional substring filter.
5. `compare_beers` — Side-by-side comparison of 2–4 beers by `ids` (key metrics, shared ingredients, unique ingredients).
6. `find_similar_beers` — Top-N similar recipes by `id` based on yeast, hops/malt overlap, and ABV/IBU/EBC proximity.
7. `get_catalog_stats` — Catalog-wide analytics (count, min/max/avg ABV/IBU/EBC, top hops/yeast, yearly distribution).

## Implementation Roadmap & Workflow Rules
We are building this project strictly in **4 stages**:
1. **Stage 1 — Core & Data Layer:** Project scaffolding, domain models, `DataSource` interface, paginated `HTTPSource`, thread-safe in-memory cache with TTL, and unit tests.
2. **Stage 2 — Search & Analytics Engine:** Pure business logic in `internal/service` for all 7 tool capabilities + thorough table-driven unit tests.
3. **Stage 3 — MCP Layer & Transports:** `modelcontextprotocol/go-sdk` integration, all 7 tools with clear LLM-facing descriptions/schemas, `stdio` & `Streamable HTTP` transports, graceful shutdown.
4. **Stage 4 — Infrastructure & Packaging:** Multi-stage `Dockerfile` (`scratch` or `distroless`), `docker-compose.yml` example, GitHub Actions CI (lint, test, build), and `README.md`.

### Strict Rules for the Agent
- **Work strictly on the current stage.** Do not implement future stages or add placeholder stubs for later stages unless explicitly asked.
- **Verify before assuming.** Check the upstream API docs/README before writing JSON tags or pagination logic.
- **Run `go test ./...` and `go vet ./...`** after making changes and ensure everything passes cleanly.
