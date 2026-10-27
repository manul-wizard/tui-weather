package weather

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"slices"
	"sync"
	"time"
)

// UserAgent identifies the program to weather services, as several of them
// ask clients to do.
var UserAgent = "tui-weather (+https://github.com/manul-wizard/tui-weather)"

// Provider fetches a forecast from one weather service. A provider may fill
// only some parts of the forecast.
type Provider interface {
	Name() string
	Fetch(ctx context.Context, client *http.Client, loc Location) (*Forecast, error)
}

var providers = []Provider{OpenMeteo{}, Wttr{}, SevenTimer{}}

// ProviderNames lists the names of all known providers, in their default
// order of preference.
func ProviderNames() []string {
	names := make([]string, len(providers))
	for i, p := range providers {
		names[i] = p.Name()
	}
	return names
}

// ProvidersByName returns the providers with the given names, keeping their
// order.
func ProvidersByName(names []string) ([]Provider, error) {
	var out []Provider
	for _, name := range names {
		i := slices.IndexFunc(providers, func(p Provider) bool { return p.Name() == name })
		if i < 0 {
			return nil, fmt.Errorf("unknown provider %q, expected one of %v", name, ProviderNames())
		}
		out = append(out, providers[i])
	}
	return out, nil
}

// NewClient returns an HTTP client that gives up quickly on unreachable
// hosts, so that one blocked service does not hold up the others.
func NewClient(timeout time.Duration) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = (&net.Dialer{Timeout: 4 * time.Second}).DialContext
	transport.TLSHandshakeTimeout = 4 * time.Second
	return &http.Client{Timeout: timeout, Transport: transport}
}

// FetchAll asks all providers at once and merges their answers: each part of
// the forecast comes from the first provider in the list that has it.
func FetchAll(ctx context.Context, client *http.Client, list []Provider, loc Location) (*Forecast, error) {
	type result struct {
		forecast *Forecast
		err      error
	}
	results := make([]result, len(list))
	var wg sync.WaitGroup
	for i, p := range list {
		wg.Go(func() {
			f, err := p.Fetch(ctx, client, loc)
			results[i] = result{f, err}
		})
	}
	wg.Wait()

	merged := &Forecast{Location: loc, FetchedAt: time.Now()}
	var errs []error
	for i, r := range results {
		if r.err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", list[i].Name(), r.err))
			continue
		}
		merged.fill(r.forecast, list[i].Name())
	}
	if merged.Empty() {
		if len(errs) == 0 {
			errs = append(errs, errors.New("no providers configured"))
		}
		return nil, errors.Join(errs...)
	}
	addSunTimes(merged)
	return merged, nil
}

// addSunTimes fills in sunrise and sunset for days whose provider lacks them.
func addSunTimes(f *Forecast) {
	for i := range f.Days {
		d := &f.Days[i]
		if d.Sunrise == nil {
			d.Sunrise, d.Sunset = SunTimes(d.Date, f.Location.Latitude, f.Location.Longitude)
		}
	}
}

func getJSON(ctx context.Context, client *http.Client, url string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s", resp.Status)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(v); err != nil {
		return fmt.Errorf("bad response: %w", err)
	}
	return nil
}
