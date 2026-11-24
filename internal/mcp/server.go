package mcp

import (
	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/alxiw/punkmcp/internal/service"
)

const (
	serverName         = "punkmcp"
	serverTitle        = "PunkMCP"
	defaultVersion     = "dev"
	serverInstructions = "PunkMCP provides read-only search, comparison, similarity, and analytics tools for the BrewDog DIY Dog / PunkAPI catalog (~415 beer recipes). Start with search_beers for compact summaries, list_ingredients to discover exact hop/malt/yeast names, and get_beer_recipe to fetch a full brewing recipe by ID."
)

// NewServer constructs an MCP server exposing all 7 PunkMCP tools backed by svc.
func NewServer(svc *service.Service, version string) *gomcp.Server {
	if version == "" {
		version = defaultVersion
	}

	server := gomcp.NewServer(
		&gomcp.Implementation{
			Name:        serverName,
			Title:       serverTitle,
			Description: "Model Context Protocol server for the BrewDog DIY Dog / PunkAPI beer recipe catalog",
			Version:     version,
		},
		&gomcp.ServerOptions{
			Instructions: serverInstructions,
		},
	)

	registerTools(server, svc)
	return server
}
