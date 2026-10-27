package weather

import (
	"context"
	"math"
	"net/http"
	"net/url"
	"strings"
)

var geocodingURL = "https://geocoding-api.open-meteo.com/v1/search"

// Search finds places by name with the Open-Meteo geocoding API. Names come
// back in the given language when the service knows them.
func Search(ctx context.Context, client *http.Client, name, language string) ([]Location, error) {
	q := url.Values{}
	q.Set("name", strings.TrimSpace(name))
	q.Set("count", "10")
	q.Set("language", language)
	q.Set("format", "json")

	var r struct {
		Results []struct {
			Name      string  `json:"name"`
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
			Timezone  string  `json:"timezone"`
			Admin1    string  `json:"admin1"`
			Country   string  `json:"country"`
		} `json:"results"`
	}
	if err := getJSON(ctx, client, geocodingURL+"?"+q.Encode(), &r); err != nil {
		return nil, err
	}

	places := make([]Location, 0, len(r.Results))
	for _, p := range r.Results {
		parts := []string{p.Name}
		for _, s := range []string{p.Admin1, p.Country} {
			if s != "" && s != parts[len(parts)-1] {
				parts = append(parts, s)
			}
		}
		places = append(places, Location{
			Name:      strings.Join(parts, ", "),
			Latitude:  round(p.Latitude, 4),
			Longitude: round(p.Longitude, 4),
			Timezone:  p.Timezone,
		})
	}
	return places, nil
}

func round(v float64, digits int) float64 {
	scale := math.Pow(10, float64(digits))
	return math.Round(v*scale) / scale
}
