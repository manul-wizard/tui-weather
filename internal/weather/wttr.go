package weather

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// Wttr reads the current weather from https://wttr.in. It asks for the
// compact j2 format only: the full one is large, and some networks cut large
// responses short.
type Wttr struct{}

// Name implements Provider.
func (Wttr) Name() string { return "wttr" }

var wttrURL = "https://wttr.in"

type wttrResponse struct {
	CurrentCondition []struct {
		ObservationTime string `json:"observation_time"` // "09:33 AM", UTC
		TempC           string `json:"temp_C"`
		FeelsLikeC      string `json:"FeelsLikeC"`
		Humidity        string `json:"humidity"`
		WindKmph        string `json:"windspeedKmph"`
		WeatherCode     string `json:"weatherCode"`
	} `json:"current_condition"`
}

// Fetch implements Provider.
func (Wttr) Fetch(ctx context.Context, client *http.Client, loc Location) (*Forecast, error) {
	u := fmt.Sprintf("%s/%.4f,%.4f?format=j2", wttrURL, loc.Latitude, loc.Longitude)
	var r wttrResponse
	if err := getJSON(ctx, client, u, &r); err != nil {
		return nil, err
	}
	return r.forecast(loc, time.Now())
}

func (r *wttrResponse) forecast(loc Location, now time.Time) (*Forecast, error) {
	if len(r.CurrentCondition) == 0 {
		return nil, fmt.Errorf("no current conditions")
	}
	c := r.CurrentCondition[0]
	temp, err := strconv.ParseFloat(c.TempC, 64)
	if err != nil {
		return nil, fmt.Errorf("temperature: %w", err)
	}
	code, _ := strconv.Atoi(c.WeatherCode)

	observed := now
	if t, err := time.Parse("03:04 PM", c.ObservationTime); err == nil {
		utc := now.UTC()
		observed = time.Date(utc.Year(), utc.Month(), utc.Day(), t.Hour(), t.Minute(), 0, 0, time.UTC)
		if observed.After(utc) {
			observed = observed.AddDate(0, 0, -1)
		}
	}
	observed = observed.In(loc.TZ())

	cur := &Current{
		Time:      observed,
		Condition: fromWWO(code),
		IsDay:     isDaytime(observed, loc.Latitude, loc.Longitude),
		Temp:      temp,
		FeelsLike: parseOptional(c.FeelsLikeC),
		Humidity:  parseOptional(c.Humidity),
		Wind:      parseOptional(c.WindKmph),
	}
	return &Forecast{Location: loc, Current: cur}, nil
}

func parseOptional(s string) *float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &v
}
