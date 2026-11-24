package service

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"

	"github.com/alxiw/punkmcp/internal/model"
)

var (
	// ErrNotFound indicates a requested beer or matching recipe does not exist.
	ErrNotFound = errors.New("not found")
	// ErrInvalidInput indicates caller-provided parameters failed validation.
	ErrInvalidInput = errors.New("invalid input")
)

// BeerStore defines the minimal read-only catalog store required by Service.
// It is satisfied by *cache.Cache.
type BeerStore interface {
	GetAll() []model.Beer
	GetByID(id int) (model.Beer, bool)
}

// Option configures a Service instance.
type Option func(*Service)

// WithRandSource injects a deterministic math/rand/v2.Source for GetRandomBeer.
// Calls to the underlying RNG are synchronized so Service remains safe for concurrent use.
func WithRandSource(src rand.Source) Option {
	return func(s *Service) {
		if src != nil {
			s.rng = rand.New(src)
		}
	}
}

// Service implements pure search, comparison, similarity, and analytics logic over a BeerStore.
type Service struct {
	store  BeerStore
	randMu sync.Mutex
	rng    *rand.Rand
}

// New constructs a new Service backed by store.
func New(store BeerStore, opts ...Option) *Service {
	s := &Service{
		store: store,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}
	return s
}

func (s *Service) allBeers() []model.Beer {
	if s.store == nil {
		return nil
	}
	return s.store.GetAll()
}

func (s *Service) beerByID(id int) (model.Beer, bool) {
	if s.store == nil {
		return model.Beer{}, false
	}
	return s.store.GetByID(id)
}

func (s *Service) intN(n int) int {
	if s.rng == nil {
		return rand.IntN(n)
	}
	s.randMu.Lock()
	defer s.randMu.Unlock()
	return s.rng.IntN(n)
}

// GetBeerRecipe returns the full recipe for the beer with the given ID, or ErrNotFound.
func (s *Service) GetBeerRecipe(id int) (model.Beer, error) {
	beer, ok := s.beerByID(id)
	if !ok {
		return model.Beer{}, fmt.Errorf("%w: beer %d", ErrNotFound, id)
	}
	return beer.Clone(), nil
}

// GetRandomBeer filters the catalog by optional criteria and selects one matching beer uniformly at random.
func (s *Service) GetRandomBeer(params RandomParams) (model.Beer, error) {
	dateRange, err := parseBrewDateRange(params.BrewedAfter, params.BrewedBefore)
	if err != nil {
		return model.Beer{}, err
	}

	foodNeedle := strings.ToLower(strings.TrimSpace(params.Food))

	all := s.allBeers()
	matched := make([]model.Beer, 0, len(all))
	for _, b := range all {
		if !matchesFloatRange(b.ABV, params.ABVMin, params.ABVMax) {
			continue
		}
		if !matchesPtrFloatRange(b.IBU, params.IBUMin, params.IBUMax) {
			continue
		}
		if !matchesPtrFloatRange(b.EBC, params.EBCMin, params.EBCMax) {
			continue
		}
		if foodNeedle != "" && !matchesAnyString(b.FoodPairing, foodNeedle) {
			continue
		}
		if dateRange.active() {
			d, parseErr := parseBrewDate(b.FirstBrewed)
			if !dateRange.matches(d, parseErr == nil) {
				continue
			}
		}
		matched = append(matched, b)
	}

	if len(matched) == 0 {
		return model.Beer{}, fmt.Errorf("%w: no beers match the given filters", ErrNotFound)
	}

	// Sort by ID before indexing so results are 100% deterministic for a given RNG seed
	// regardless of store iteration order.
	slices.SortFunc(matched, func(a, b model.Beer) int {
		return compareInt(a.ID, b.ID)
	})

	pick := matched[s.intN(len(matched))]
	return pick.Clone(), nil
}

func toBeerSummary(b model.Beer) BeerSummary {
	foodPairing := []string{}
	if len(b.FoodPairing) > 0 {
		foodPairing = make([]string, len(b.FoodPairing))
		copy(foodPairing, b.FoodPairing)
	}

	return BeerSummary{
		ID:          b.ID,
		Name:        b.Name,
		Tagline:     b.Tagline,
		Description: b.Description,
		ABV:         b.ABV,
		IBU:         cloneFloat64Ptr(b.IBU),
		EBC:         cloneFloat64Ptr(b.EBC),
		FirstBrewed: b.FirstBrewed,
		FoodPairing: foodPairing,
	}
}

func cloneFloat64Ptr(p *float64) *float64 {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func cloneStringPtr(p *string) *string {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func compareInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}
