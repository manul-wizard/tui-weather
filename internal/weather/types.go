// Package weather fetches forecasts from several free providers and merges
// them into one provider-neutral Forecast.
package weather

import "time"

// Condition is a provider-neutral weather condition.
type Condition int

// Conditions, roughly ordered from calm to severe.
const (
	Unknown Condition = iota
	Clear
	PartlyCloudy
	Cloudy
	Fog
	LightRain
	Rain
	HeavyRain
	Sleet
	Snow
	Thunder
)

// Location is a named point on the map.
type Location struct {
	Name      string  `toml:"name" json:"name"`
	Latitude  float64 `toml:"latitude" json:"latitude"`
	Longitude float64 `toml:"longitude" json:"longitude"`
	// Timezone is an IANA name such as "Europe/Madrid". Times are shown in
	// it; when empty, the local timezone of the computer is used.
	Timezone string `toml:"timezone" json:"timezone"`
}

// TZ returns the timezone of the location.
func (l Location) TZ() *time.Location {
	if l.Timezone != "" {
		if tz, err := time.LoadLocation(l.Timezone); err == nil {
			return tz
		}
	}
	return time.Local
}

// Current is the weather right now.
type Current struct {
	Time      time.Time `json:"time"`
	Condition Condition `json:"condition"`
	IsDay     bool      `json:"is_day"`
	Temp      float64   `json:"temp"`                 // °C
	FeelsLike *float64  `json:"feels_like,omitempty"` // °C
	Humidity  *float64  `json:"humidity,omitempty"`   // %
	Wind      *float64  `json:"wind,omitempty"`       // km/h
}

// Hour is one step of the short-term forecast. Some providers give one step
// per hour, others one per three hours.
type Hour struct {
	Time         time.Time `json:"time"`
	Condition    Condition `json:"condition"`
	IsDay        bool      `json:"is_day"`
	Temp         float64   `json:"temp"`
	PrecipChance *float64  `json:"precip_chance,omitempty"` // %
}

// Day is one day of the forecast.
type Day struct {
	Date         time.Time  `json:"date"` // midnight in the location's timezone
	Condition    Condition  `json:"condition"`
	Min          float64    `json:"min"`
	Max          float64    `json:"max"`
	PrecipChance *float64   `json:"precip_chance,omitempty"`
	Sunrise      *time.Time `json:"sunrise,omitempty"`
	Sunset       *time.Time `json:"sunset,omitempty"`
}

// Sources names the provider each part of a forecast came from.
type Sources struct {
	Current string `json:"current,omitempty"`
	Hours   string `json:"hours,omitempty"`
	Days    string `json:"days,omitempty"`
}

// Forecast is everything known about the weather at one location.
type Forecast struct {
	Location  Location  `json:"location"`
	Current   *Current  `json:"current,omitempty"`
	Hours     []Hour    `json:"hours,omitempty"`
	Days      []Day     `json:"days,omitempty"`
	Sources   Sources   `json:"sources"`
	FetchedAt time.Time `json:"fetched_at"`

	// Stale is set when no provider answered and the forecast came from an
	// old cache entry.
	Stale bool `json:"-"`
}

// Complete reports whether every part of the forecast is present.
func (f *Forecast) Complete() bool {
	return f.Current != nil && len(f.Hours) > 0 && len(f.Days) > 0
}

// Empty reports whether no part of the forecast is present.
func (f *Forecast) Empty() bool {
	return f.Current == nil && len(f.Hours) == 0 && len(f.Days) == 0
}

// fill copies into f the parts it lacks from src.
func (f *Forecast) fill(src *Forecast, provider string) {
	if f.Current == nil && src.Current != nil {
		f.Current, f.Sources.Current = src.Current, provider
	}
	if len(f.Hours) == 0 && len(src.Hours) > 0 {
		f.Hours, f.Sources.Hours = src.Hours, provider
	}
	if len(f.Days) == 0 && len(src.Days) > 0 {
		f.Days, f.Sources.Days = src.Days, provider
	}
}

func ptr(v float64) *float64 { return &v }
