package weather

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeServices points every provider at a local server that answers with the
// fixtures, except for the services listed as down.
func fakeServices(t *testing.T, down ...string) {
	t.Helper()
	isDown := func(name string) bool {
		for _, d := range down {
			if d == name {
				return true
			}
		}
		return false
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var name, file string
		switch {
		case strings.HasPrefix(r.URL.Path, "/open-meteo"):
			name, file = "open-meteo", "open-meteo.json"
		case strings.HasPrefix(r.URL.Path, "/wttr"):
			name, file = "wttr", "wttr-j2.json"
		case strings.HasSuffix(r.URL.Path, "/civil.php"):
			name, file = "7timer", "7timer-civil.json"
		case strings.HasSuffix(r.URL.Path, "/civillight.php"):
			name, file = "7timer", "7timer-civillight.json"
		case strings.HasPrefix(r.URL.Path, "/search"):
			name, file = "geocoding", "geocoding.json"
		}
		if file == "" || isDown(name) {
			http.Error(w, "down", http.StatusServiceUnavailable)
			return
		}
		data, err := os.ReadFile(filepath.Join("testdata", file))
		if err != nil {
			t.Error(err)
		}
		w.Write(data)
	}))
	t.Cleanup(srv.Close)

	saved := []string{openMeteoURL, wttrURL, sevenTimerURL, geocodingURL}
	openMeteoURL, wttrURL, sevenTimerURL, geocodingURL = srv.URL+"/open-meteo", srv.URL+"/wttr", srv.URL+"/7timer", srv.URL+"/search"
	t.Cleanup(func() {
		openMeteoURL, wttrURL, sevenTimerURL, geocodingURL = saved[0], saved[1], saved[2], saved[3]
	})
}

func options(t *testing.T) Options {
	return Options{
		Location:  barcelona,
		Providers: ProviderNames(),
		Timeout:   5 * time.Second,
		CacheTTL:  15 * time.Minute,
		CacheFile: filepath.Join(t.TempDir(), "forecast.json"),
	}
}

func TestGetPrefersTheFirstProvider(t *testing.T) {
	fakeServices(t)
	f, err := Get(context.Background(), options(t))
	if err != nil {
		t.Fatal(err)
	}
	want := Sources{Current: "open-meteo", Hours: "open-meteo", Days: "open-meteo"}
	if f.Sources != want {
		t.Errorf("sources = %+v, want %+v", f.Sources, want)
	}
}

func TestGetFillsGapsFromOtherProviders(t *testing.T) {
	fakeServices(t, "open-meteo")
	f, err := Get(context.Background(), options(t))
	if err != nil {
		t.Fatal(err)
	}
	want := Sources{Current: "wttr", Hours: "7timer", Days: "7timer"}
	if f.Sources != want {
		t.Errorf("sources = %+v, want %+v", f.Sources, want)
	}
	// 7Timer has no sunrise; it is worked out from the coordinates.
	if f.Days[0].Sunrise == nil {
		t.Errorf("days lack sunrise")
	}
}

func TestGetFailsWhenAllProvidersFail(t *testing.T) {
	fakeServices(t, "open-meteo", "wttr", "7timer")
	_, err := Get(context.Background(), options(t))
	if err == nil {
		t.Fatal("want an error")
	}
	for _, name := range ProviderNames() {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error %q does not mention %s", err, name)
		}
	}
}

func TestGetUsesTheCache(t *testing.T) {
	o := options(t)
	fakeServices(t)
	if _, err := Get(context.Background(), o); err != nil {
		t.Fatal(err)
	}

	// A fresh cache entry is used without asking anyone.
	fakeServices(t, "open-meteo", "wttr", "7timer")
	f, err := Get(context.Background(), o)
	if err != nil || f.Stale {
		t.Fatalf("got %v, stale %v; want the cached forecast", err, f != nil && f.Stale)
	}

	// When a refresh fails, the old forecast is shown and marked stale.
	o.Refresh = true
	f, err = Get(context.Background(), o)
	if err != nil || !f.Stale {
		t.Fatalf("got %v, stale %v; want the stale forecast", err, f != nil && f.Stale)
	}

	// Another place does not reuse the entry.
	o.Location = Location{Name: "Elsewhere", Latitude: 10, Longitude: 10}
	if _, err := Get(context.Background(), o); err == nil {
		t.Errorf("want an error for a place that is not cached")
	}
}

func TestSearch(t *testing.T) {
	fakeServices(t)
	places, err := Search(context.Background(), http.DefaultClient, "Barcelona", "en")
	if err != nil {
		t.Fatal(err)
	}
	if len(places) != 3 {
		t.Fatalf("got %d places, want 3", len(places))
	}
	want := Location{Name: "Barcelona, Catalonia, Spain", Latitude: 41.3888, Longitude: 2.159, Timezone: "Europe/Madrid"}
	if places[0] != want {
		t.Errorf("first place = %+v, want %+v", places[0], want)
	}
	// A country that is its own city is not repeated.
	if places[2].Name != "Monaco" {
		t.Errorf("third place = %q, want Monaco", places[2].Name)
	}
}
