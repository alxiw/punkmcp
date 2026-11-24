# PunkMCP

A lightweight [Model Context Protocol (MCP)](https://modelcontextprotocol.io) server written in Go for [**PunkAPI**](https://punkapi.dev) ([GitHub](https://github.com/alxiw/punkapi)) — the digital archive of 415 BrewDog DIY Dog v8 beer recipes.

PunkMCP preloads the catalog from PunkAPI into memory and exposes search, filtering, ingredient discovery, side-by-side comparison, similarity matching, and catalog analytics to MCP-compatible AI assistants.

## Highlights

- **Single static Go binary** — built on Go 1.25+ and the official [`modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk), packaged in a minimal `distroless` non-root container.
- **Fast in-memory cache** — preloads ~415 recipes from PunkAPI v3 on startup and refreshes in the background on a configurable TTL (`24h` by default) with zero per-request upstream latency.
- **Token-efficient responses** — search, ingredient, and similarity tools return compact DTOs with pagination; only `get_beer_recipe` and `get_random_beer` return full brewing schedules.
- **Dual transport support** — runs locally over `stdio` or as a network service over MCP `Streamable HTTP` (`/mcp`, plus `/healthz` for internal health checks).

## Tools Reference

| Tool | Purpose | Key Parameters |
| :--- | :--- | :--- |
| `search_beers` | Search, filter, and sort beers; returns compact cards with `total_found` and pagination | `name`, `abv_min`/`abv_max`, `ibu_min`/`ibu_max`, `ebc_min`/`ebc_max`, `hops`, `malt`, `yeast`, `food`, `brewed_before`, `brewed_after`, `sort_by` (`id`, `abv`, `ibu`, `ebc`, `first_brewed`, `name`), `order` (`asc`, `desc`), `limit` (default `10`, max `25`), `offset` |
| `get_beer_recipe` | Retrieve the full brewing recipe (malt/hop schedules, mash & fermentation temps, target gravity, pH, brewer's tips) | `id` (required) |
| `get_random_beer` | Return a random full beer recipe with optional constraints | `abv_min`/`abv_max`, `ibu_min`/`ibu_max`, `ebc_min`/`ebc_max`, `food`, `brewed_before`, `brewed_after` |
| `list_ingredients` | List unique hops, malts, or yeasts with recipe counts, sorted by popularity | `type` (required: `hops`, `malt`, `yeast`), `query`, `limit` (default `20`, max `100`) |
| `compare_beers` | Compare 2–4 beers side by side (key metrics, shared ingredients, unique ingredients per beer) | `ids` (required: array of 2–4 unique beer IDs) |
| `find_similar_beers` | Find top-N most similar beers by yeast, hop/malt overlap, and ABV/IBU/EBC proximity | `id` (required), `limit` (default `5`, max `15`) |
| `get_catalog_stats` | Catalog-wide analytics (count, min/max/avg ABV/IBU/EBC, top hops & yeasts, yearly distribution) | *(none)* |

## Configuration

CLI flags take precedence over environment variables.

| CLI Flag | Environment Variable | Default | Description |
| :--- | :--- | :--- | :--- |
| `-transport` | `MCP_TRANSPORT` | `stdio` (`http` in Docker) | MCP transport mode: `stdio` or `http` |
| `-addr` | `MCP_ADDR` | `:8080` | HTTP listen address when `-transport=http` |
| `-punkapi-base-url` | `PUNKAPI_BASE_URL` | `https://punkapi.dev` | Upstream PunkAPI v3 base URL |
| `-cache-ttl` | `CACHE_TTL` | `24h` | Background catalog refresh interval (Go duration, e.g. `12h`, `24h`) |

When running in `http` mode, PunkMCP serves:
- `/mcp` — MCP Streamable HTTP endpoint (`POST`, `GET`, `DELETE`)
- `/healthz` — JSON health check (`GET`), returning `200 OK` with `{"status":"ok","beers_loaded":415}` once the catalog is loaded (`503 Service Unavailable` if empty).

## Quick Start

### Local (`go run` / `go build`)

Run directly in `stdio` mode (default):

```bash
go run ./cmd/punkmcp
```

Run in `Streamable HTTP` mode on `:8080`:

```bash
go run ./cmd/punkmcp -transport=http -addr=:8080
```

Or build a standalone binary:

```bash
go build -o punkmcp ./cmd/punkmcp
./punkmcp -transport=http -addr=:8080
```

Verify the HTTP server health check:

```bash
curl http://127.0.0.1:8080/healthz
```

### Docker

Build and run standalone (fetching from `https://punkapi.dev`):

```bash
docker build -t punkmcp:dev .
docker run --rm -p 127.0.0.1:8080:8080 punkmcp:dev
```

Deploy alongside `PunkAPI` on a shared Docker network (`punk_net`):

```bash
docker network create punk_net
docker compose up -d --build
```

An Nginx reverse proxy snippet for exposing `/mcp` over HTTPS (with streaming/SSE buffering disabled) is provided in [`deploy/nginx/punkmcp.conf`](deploy/nginx/punkmcp.conf).

## MCP Client Setup

### Remote Streamable HTTP (`https://punkapi.dev/mcp`)

Use this configuration with MCP clients that support remote Streamable HTTP servers (e.g., Cursor, Windsurf, Claude Code, OpenCode):

```json
{
  "mcpServers": {
    "punkmcp": {
      "url": "https://punkapi.dev/mcp"
    }
  }
}
```

For OpenCode (`opencode.json`):

```json
{
  "mcp": {
    "punkmcp": {
      "type": "remote",
      "url": "https://punkapi.dev/mcp",
      "enabled": true
    }
  }
}
```

### Local `stdio` Binary

Build the binary (`go build -o /path/to/punkmcp ./cmd/punkmcp`) and register it with Claude Desktop, Cursor, or any `stdio`-compatible MCP client:

```json
{
  "mcpServers": {
    "punkmcp": {
      "command": "/path/to/punkmcp",
      "args": ["-transport=stdio"],
      "env": {
        "PUNKAPI_BASE_URL": "https://punkapi.dev"
      }
    }
  }
}
```

For OpenCode (`opencode.json`):

```json
{
  "mcp": {
    "punkmcp": {
      "type": "local",
      "command": ["/path/to/punkmcp", "-transport=stdio"],
      "enabled": true
    }
  }
}
```

## Development

Run the full test suite with the race detector, static analysis, and formatting check:

```bash
go test -race -v ./...
go vet ./...
gofmt -l .
```

### Project Layout

```text
cmd/punkmcp/          CLI entrypoint, flag/env config, graceful shutdown
internal/model/       Upstream PunkAPI JSON models and compact DTOs
internal/source/      DataSource interface and paginated PunkAPI HTTP client
internal/cache/       Thread-safe in-memory catalog store with background TTL refresh
internal/service/     Pure search, filter, ingredient, comparison, similarity, and stats logic
internal/mcp/         MCP server, typed tool handlers, and Streamable HTTP / healthz wiring
deploy/nginx/         Nginx reverse proxy configuration snippet
```
