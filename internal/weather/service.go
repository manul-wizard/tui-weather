package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Options say where and how to get a forecast.
type Options struct {
	Location  Location
	Providers []string
	Timeout   time.Duration
	// CacheTTL is how long a cached forecast is used without asking the
	// providers again. Zero turns the cache off.
	CacheTTL time.Duration
	// CacheFile is where the last forecast is kept. Empty turns the cache off.
	CacheFile string
	// Refresh skips a fresh cache entry, but an old one is still used when no
	// provider answers.
	Refresh bool
}

type cacheEntry struct {
	Key      string    `json:"key"`
	Forecast *Forecast `json:"forecast"`
}

// DefaultCacheFile returns the cache path in the user's cache directory.
func DefaultCacheFile() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "tui-weather", "forecast.json")
}

// Get returns a forecast from the cache or from the providers.
func Get(ctx context.Context, o Options) (*Forecast, error) {
	list, err := ProvidersByName(o.Providers)
	if err != nil {
		return nil, err
	}
	key := fmt.Sprintf("%.3f,%.3f|%s", o.Location.Latitude, o.Location.Longitude, strings.Join(o.Providers, ","))
	cached := readCache(o.CacheFile, key)
	if cached != nil && !o.Refresh && o.CacheTTL > 0 && time.Since(cached.FetchedAt) < o.CacheTTL {
		cached.Location = o.Location
		return cached, nil
	}

	ctx, cancel := context.WithTimeout(ctx, o.Timeout)
	defer cancel()
	f, err := FetchAll(ctx, NewClient(o.Timeout), list, o.Location)
	if err != nil {
		if cached != nil {
			cached.Location = o.Location
			cached.Stale = true
			return cached, nil
		}
		return nil, err
	}
	if o.CacheTTL > 0 {
		writeCache(o.CacheFile, cacheEntry{Key: key, Forecast: f})
	}
	return f, nil
}

func readCache(path, key string) *Forecast {
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var e cacheEntry
	if json.Unmarshal(data, &e) != nil || e.Key != key || e.Forecast == nil {
		return nil
	}
	return e.Forecast
}

// writeCache stores the entry. Failing to write a cache is not worth
// bothering the user about, so errors are dropped.
func writeCache(path string, e cacheEntry) {
	if path == "" {
		return
	}
	data, err := json.Marshal(e)
	if err != nil {
		return
	}
	if os.MkdirAll(filepath.Dir(path), 0o755) != nil {
		return
	}
	tmp := path + ".tmp"
	if os.WriteFile(tmp, data, 0o644) != nil {
		return
	}
	if os.Rename(tmp, path) != nil {
		_ = os.Remove(tmp)
	}
}
