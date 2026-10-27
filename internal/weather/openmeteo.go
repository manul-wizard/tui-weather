package weather

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// OpenMeteo reads https://open-meteo.com: current weather, 48 hours and
// 7 days. It is the most complete provider.
type OpenMeteo struct{}

// Name implements Provider.
func (OpenMeteo) Name() string { return "open-meteo" }

var openMeteoURL = "https://api.open-meteo.com/v1/forecast"

type openMeteoResponse struct {
	Timezone         string `json:"timezone"`
	UTCOffsetSeconds int    `json:"utc_offset_seconds"`
	Current          struct {
		Time        string  `json:"time"`
		Temp        float64 `json:"temperature_2m"`
		FeelsLike   float64 `json:"apparent_temperature"`
		Humidity    float64 `json:"relative_humidity_2m"`
		WeatherCode int     `json:"weather_code"`
		Wind        float64 `json:"wind_speed_10m"`
		IsDay       int     `json:"is_day"`
	} `json:"current"`
	Hourly struct {
		Time         []string   `json:"time"`
		Temp         []float64  `json:"temperature_2m"`
		WeatherCode  []int      `json:"weather_code"`
		PrecipChance []*float64 `json:"precipitation_probability"`
		IsDay        []int      `json:"is_day"`
	} `json:"hourly"`
	Daily struct {
		Time         []string   `json:"time"`
		WeatherCode  []int      `json:"weather_code"`
		Max          []float64  `json:"temperature_2m_max"`
		Min          []float64  `json:"temperature_2m_min"`
		PrecipChance []*float64 `json:"precipitation_probability_max"`
		Sunrise      []string   `json:"sunrise"`
		Sunset       []string   `json:"sunset"`
	} `json:"daily"`
}

// Fetch implements Provider.
func (OpenMeteo) Fetch(ctx context.Context, client *http.Client, loc Location) (*Forecast, error) {
	q := url.Values{}
	q.Set("latitude", fmt.Sprintf("%.4f", loc.Latitude))
	q.Set("longitude", fmt.Sprintf("%.4f", loc.Longitude))
	q.Set("current", "temperature_2m,apparent_temperature,relative_humidity_2m,weather_code,wind_speed_10m,is_day")
	q.Set("hourly", "temperature_2m,weather_code,precipitation_probability,is_day")
	q.Set("daily", "weather_code,temperature_2m_max,temperature_2m_min,precipitation_probability_max,sunrise,sunset")
	q.Set("timezone", "auto")
	q.Set("forecast_days", "7")
	q.Set("forecast_hours", "48")

	var r openMeteoResponse
	if err := getJSON(ctx, client, openMeteoURL+"?"+q.Encode(), &r); err != nil {
		return nil, err
	}
	return r.forecast(loc)
}

func (r *openMeteoResponse) forecast(loc Location) (*Forecast, error) {
	tz, err := time.LoadLocation(r.Timezone)
	if err != nil {
		tz = time.FixedZone(r.Timezone, r.UTCOffsetSeconds)
	}
	parse := func(s string) (time.Time, error) {
		if len(s) == len("2006-01-02") {
			return time.ParseInLocation("2006-01-02", s, tz)
		}
		return time.ParseInLocation("2006-01-02T15:04", s, tz)
	}

	f := &Forecast{Location: loc}

	if r.Current.Time != "" {
		t, err := parse(r.Current.Time)
		if err != nil {
			return nil, err
		}
		f.Current = &Current{
			Time:      t,
			Condition: fromWMO(r.Current.WeatherCode),
			IsDay:     r.Current.IsDay == 1,
			Temp:      r.Current.Temp,
			FeelsLike: ptr(r.Current.FeelsLike),
			Humidity:  ptr(r.Current.Humidity),
			Wind:      ptr(r.Current.Wind),
		}
	}

	h := r.Hourly
	if len(h.Temp) != len(h.Time) || len(h.WeatherCode) != len(h.Time) {
		return nil, fmt.Errorf("hourly arrays differ in length")
	}
	for i, s := range h.Time {
		t, err := parse(s)
		if err != nil {
			return nil, err
		}
		hour := Hour{Time: t, Condition: fromWMO(h.WeatherCode[i]), Temp: h.Temp[i], IsDay: true}
		if i < len(h.IsDay) {
			hour.IsDay = h.IsDay[i] == 1
		}
		if i < len(h.PrecipChance) {
			hour.PrecipChance = h.PrecipChance[i]
		}
		f.Hours = append(f.Hours, hour)
	}

	d := r.Daily
	if len(d.Max) != len(d.Time) || len(d.Min) != len(d.Time) || len(d.WeatherCode) != len(d.Time) {
		return nil, fmt.Errorf("daily arrays differ in length")
	}
	for i, s := range d.Time {
		date, err := parse(s)
		if err != nil {
			return nil, err
		}
		day := Day{Date: date, Condition: fromWMO(d.WeatherCode[i]), Min: d.Min[i], Max: d.Max[i]}
		if i < len(d.PrecipChance) {
			day.PrecipChance = d.PrecipChance[i]
		}
		if i < len(d.Sunrise) && i < len(d.Sunset) {
			rise, err1 := parse(d.Sunrise[i])
			set, err2 := parse(d.Sunset[i])
			if err1 == nil && err2 == nil {
				day.Sunrise, day.Sunset = &rise, &set
			}
		}
		f.Days = append(f.Days, day)
	}
	return f, nil
}
