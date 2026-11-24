package source

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/alxiw/punkmcp/internal/model"
)

const (
	// DefaultBaseURL is the public PunkAPI v3 base URL.
	DefaultBaseURL = "https://punkapi.dev"

	// DefaultRequestTimeout is the default timeout for each HTTP page request.
	DefaultRequestTimeout = 10 * time.Second

	// MinPerPage and MaxPerPage are the query bounds enforced by PunkAPI v3 /api/v3/beers.
	MinPerPage = 10
	MaxPerPage = 80

	beersEndpointPath   = "/api/v3/beers"
	maxErrorBodySnippet = 512
)

// HTTPSource fetches all beer recipes from a PunkAPI v3 HTTP endpoint using pagination.
type HTTPSource struct {
	baseURL string
	client  *http.Client
	perPage int
}

// HTTPOption configures an HTTPSource.
type HTTPOption func(*HTTPSource)

// WithHTTPClient sets a custom HTTP client for requests.
func WithHTTPClient(client *http.Client) HTTPOption {
	return func(s *HTTPSource) {
		if client != nil {
			s.client = client
		}
	}
}

// WithPerPage sets the page size used when paginating /api/v3/beers.
// Values outside [MinPerPage, MaxPerPage] are clamped to valid PunkAPI v3 bounds.
func WithPerPage(perPage int) HTTPOption {
	return func(s *HTTPSource) {
		switch {
		case perPage < MinPerPage:
			s.perPage = MinPerPage
		case perPage > MaxPerPage:
			s.perPage = MaxPerPage
		default:
			s.perPage = perPage
		}
	}
}

// NewHTTPSource creates a new PunkAPI HTTP data source.
// If baseURL is empty, DefaultBaseURL is used.
func NewHTTPSource(baseURL string, opts ...HTTPOption) *HTTPSource {
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmed == "" {
		trimmed = DefaultBaseURL
	}

	s := &HTTPSource{
		baseURL: trimmed,
		client: &http.Client{
			Timeout: DefaultRequestTimeout,
		},
		perPage: MaxPerPage,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}

	return s
}

// LoadBeers fetches all pages sequentially from /api/v3/beers until an empty page
// or a page with fewer items than perPage is returned.
func (s *HTTPSource) LoadBeers(ctx context.Context) ([]model.Beer, error) {
	allBeers := make([]model.Beer, 0)

	for page := 1; ; page++ {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("load beers canceled before page %d: %w", page, err)
		}

		pageBeers, err := s.fetchPage(ctx, page)
		if err != nil {
			return nil, err
		}

		if len(pageBeers) == 0 {
			break
		}

		allBeers = append(allBeers, pageBeers...)

		if len(pageBeers) < s.perPage {
			break
		}
	}

	return allBeers, nil
}

func (s *HTTPSource) fetchPage(ctx context.Context, page int) ([]model.Beer, error) {
	endpoint, err := s.pageURL(page)
	if err != nil {
		return nil, fmt.Errorf("build URL for page %d: %w", page, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create request for page %d: %w", page, err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request beers page %d: %w", page, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodySnippet, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodySnippet))
		snippet := strings.TrimSpace(string(bodySnippet))
		if snippet != "" {
			return nil, fmt.Errorf("punkapi returned HTTP %d for page %d: %s", resp.StatusCode, page, snippet)
		}
		return nil, fmt.Errorf("punkapi returned HTTP %d for page %d", resp.StatusCode, page)
	}

	var beers []model.Beer
	if err := json.NewDecoder(resp.Body).Decode(&beers); err != nil {
		return nil, fmt.Errorf("decode beers page %d: %w", page, err)
	}

	return beers, nil
}

func (s *HTTPSource) pageURL(page int) (string, error) {
	u, err := url.Parse(s.baseURL + beersEndpointPath)
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Set("page", strconv.Itoa(page))
	q.Set("per_page", strconv.Itoa(s.perPage))
	u.RawQuery = q.Encode()

	return u.String(), nil
}
