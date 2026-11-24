package source

import (
	"context"

	"github.com/alxiw/punkmcp/internal/model"
)

// DataSource loads the full catalog of beer recipes from an upstream source.
type DataSource interface {
	LoadBeers(ctx context.Context) ([]model.Beer, error)
}
