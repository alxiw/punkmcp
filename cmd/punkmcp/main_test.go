package main

import (
	"io"
	"testing"
	"time"

	"github.com/alxiw/punkmcp/internal/cache"
	"github.com/alxiw/punkmcp/internal/source"
)

func TestParseConfig(t *testing.T) {
	t.Parallel()

	envMap := func(values map[string]string) func(string) string {
		return func(key string) string {
			return values[key]
		}
	}

	tests := []struct {
		name    string
		args    []string
		env     map[string]string
		want    config
		wantErr bool
	}{
		{
			name: "defaults",
			want: config{
				Transport:      "stdio",
				Addr:           ":8080",
				PunkAPIBaseURL: source.DefaultBaseURL,
				CacheTTL:       cache.DefaultTTL,
			},
		},
		{
			name: "env fallbacks",
			env: map[string]string{
				"MCP_TRANSPORT":    "HTTP",
				"MCP_ADDR":         "127.0.0.1:9090",
				"PUNKAPI_BASE_URL": "http://punkapi:8080",
				"CACHE_TTL":        "12h",
			},
			want: config{
				Transport:      "http",
				Addr:           "127.0.0.1:9090",
				PunkAPIBaseURL: "http://punkapi:8080",
				CacheTTL:       12 * time.Hour,
			},
		},
		{
			name: "flags override env",
			env: map[string]string{
				"MCP_TRANSPORT":    "http",
				"MCP_ADDR":         ":9000",
				"PUNKAPI_BASE_URL": "http://env-host",
				"CACHE_TTL":        "6h",
			},
			args: []string{
				"--transport=stdio",
				"--addr=:7070",
				"--punkapi-base-url=http://flag-host",
				"--cache-ttl=30m",
			},
			want: config{
				Transport:      "stdio",
				Addr:           ":7070",
				PunkAPIBaseURL: "http://flag-host",
				CacheTTL:       30 * time.Minute,
			},
		},
		{
			name:    "invalid transport",
			args:    []string{"--transport=sse"},
			wantErr: true,
		},
		{
			name:    "invalid env cache ttl",
			env:     map[string]string{"CACHE_TTL": "not-a-duration"},
			wantErr: true,
		},
		{
			name:    "non-positive flag cache ttl",
			args:    []string{"--cache-ttl=0s"},
			wantErr: true,
		},
		{
			name:    "empty addr in http mode",
			args:    []string{"--transport=http", "--addr="},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseConfig(tc.args, envMap(tc.env), io.Discard)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (config: %+v)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("parseConfig() = %+v, want %+v", got, tc.want)
			}
		})
	}
}
