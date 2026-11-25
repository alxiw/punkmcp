# 🍺 PunkMCP

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)](https://go.dev) [![MCP](https://img.shields.io/badge/MCP-Streamable%20HTTP%20%26%20stdio-blue)](https://modelcontextprotocol.io) [![Docker](https://img.shields.io/badge/GHCR-punkmcp-2496ED?logo=docker&logoColor=white)](https://github.com/alxiw/punkmcp/pkgs/container/punkmcp) [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A lightweight [Model Context Protocol (MCP)](https://modelcontextprotocol.io) server in Go for **[PunkAPI](https://punkapi.dev)** ([GitHub](https://github.com/alxiw/punkapi)) — the digital archive of 415 BrewDog DIY Dog v8 beer recipes.

Give your AI assistant instant access to search, filter, compare, and analyze craft beer recipes — with zero upstream latency.

## ✨ What you can ask your AI

* *"Find hoppy IPAs under 6% ABV brewed with Citra hops that pair well with tacos."*
* *"Give me the full brewing recipe for Punk IPA, including hop schedule and mash temp."*
* *"Compare Punk IPA and Elvis Juice side by side — what ingredients do they share?"*
* *"What are the 5 most similar beers to Trashy Blonde?"*
* *"Which hops and yeasts are used most often across the entire catalog?"*

## ⚡ Highlights

* **Zero-latency in-memory cache** — preloads all 415 recipes on startup and refreshes in the background (`24h` TTL).
* **Token-efficient** — list/search tools return compact cards with pagination; full brewing schedules are fetched only when needed.
* **Dual transport** — works remotely over **Streamable HTTP** (`/mcp`) or locally over **`stdio`**.
* **Tiny footprint** — single static Go binary built on the official [`modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk), packaged in a `distroless` non-root image.

## 🔌 Connect to the Public Server

The fastest way to try PunkMCP — no local setup required.

**Cursor, Windsurf, Claude Code, etc.:**

```json
{
  "mcpServers": {
    "punkmcp": {
      "url": "https://punkapi.dev/mcp"
    }
  }
}
```

**OpenCode (`opencode.json`):**

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

You can check if the public server is up at any time:

```bash
curl https://punkapi.dev/mcp/healthz
# {"status":"ok","beers_loaded":415}
```

## 🛠️ Tools

| Tool | Description | Key Parameters |
| :--- | :--- | :--- |
| `search_beers` | Search, filter, and sort beers (compact cards + pagination) | `name`, `abv`/`ibu`/`ebc` ranges, `hops`, `malt`, `yeast`, `food`, `brewed_before`/`after`, `sort_by`, `order`, `limit`, `offset` |
| `get_beer_recipe` | Full brewing recipe (malt/hop schedules, mash & fermentation temps, tips) | `id` *(required)* |
| `get_random_beer` | Random full recipe with optional filters | `abv`/`ibu`/`ebc` ranges, `food`, `brewed_before`/`after` |
| `list_ingredients` | Unique hops, malts, or yeasts sorted by popularity | `type` (`hops`, `malt`, `yeast`), `query`, `limit` |
| `compare_beers` | Side-by-side comparison of 2–4 beers (metrics, shared & unique ingredients) | `ids` *(2–4 beer IDs)* |
| `find_similar_beers` | Top-N similar beers by yeast, hop/malt overlap, and ABV/IBU/EBC | `id` *(required)*, `limit` |
| `get_catalog_stats` | Catalog-wide analytics (averages, top ingredients, yearly breakdown) | — |

## 🐳 Self-Hosting & Local Usage

### Docker (GHCR)

```bash
docker run -d --name punkmcp -p 127.0.0.1:8080:8080 ghcr.io/alxiw/punkmcp:latest
```

Or with Docker Compose:

```bash
docker compose up -d
```

Verify the server is ready:

```bash
curl http://127.0.0.1:8080/healthz
# {"status":"ok","beers_loaded":415}
```

> An Nginx reverse proxy snippet for exposing `/mcp` over HTTPS is available in [`deploy/nginx/punkmcp.conf`](deploy/nginx/punkmcp.conf).

### Local `stdio` Binary

```bash
go build -o /path/to/punkmcp ./cmd/punkmcp
```

Register in Claude Desktop, Cursor, or any `stdio` MCP client:

```json
{
  "mcpServers": {
    "punkmcp": {
      "command": "/path/to/punkmcp",
      "args": ["-transport=stdio"]
    }
  }
}
```

## ⚙️ Configuration

CLI flags take precedence over environment variables.

| Flag | Env Variable | Default | Description |
| :--- | :--- | :--- | :--- |
| `-transport` | `MCP_TRANSPORT` | `stdio` (`http` in Docker) | Transport mode: `stdio` or `http` |
| `-addr` | `MCP_ADDR` | `:8080` | HTTP listen address (`http` mode) |
| `-punkapi-base-url` | `PUNKAPI_BASE_URL` | `https://punkapi.dev` | Upstream PunkAPI v3 URL |
| `-cache-ttl` | `CACHE_TTL` | `24h` | Background catalog refresh interval |

## 📄 License

Distributed under the [MIT License](LICENSE).