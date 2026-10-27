package weather

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// SevenTimer reads https://www.7timer.info: a forecast in 3-hour steps and
// daily summaries for about a week. It is coarse, but small and reachable
// from networks that block the others.
type SevenTimer struct{}

// Name implements Provider.
func (SevenTimer) Name() string { return "7timer" }

var sevenTimerURL = "https://www.7timer.info/bin"

type sevenTimerCivil struct {
	Init       string `json:"init"` // "2026102700", UTC
	Dataseries []struct {
		Timepoint int    `json:"timepoint"` // hours after Init
		Temp      int    `json:"temp2m"`
		Humidity  string `json:"rh2m"` // "66%"
		Wind      struct {
			Speed int `json:"speed"` // Beaufort-like scale, 1 to 8
		} `json:"wind10m"`
		Weather string `json:"weather"`
	} `json:"dataseries"`
}

type sevenTimerCivilLight struct {
	Dataseries []struct {
		Date    int    `json:"date"` // 20261027
		Weather string `json:"weather"`
		Temp    struct {
			Max int `json:"max"`
			Min int `json:"min"`
		} `json:"temp2m"`
	} `json:"dataseries"`
}

// Fetch implements Provider.
func (SevenTimer) Fetch(ctx context.Context, client *http.Client, loc Location) (*Forecast, error) {
	query := fmt.Sprintf("?lon=%.3f&lat=%.3f&output=json", loc.Longitude, loc.Latitude)
	var civil sevenTimerCivil
	var light sevenTimerCivilLight
	errc := make(chan error, 1)
	go func() { errc <- getJSON(ctx, client, sevenTimerURL+"/civillight.php"+query, &light) }()
	if err := getJSON(ctx, client, sevenTimerURL+"/civil.php"+query, &civil); err != nil {
		return nil, err
	}
	lightErr := <-errc

	f, err := civil.forecast(loc, time.Now())
	if err != nil {
		return nil, err
	}
	if lightErr == nil {
		f.Days = light.days(loc)
	}
	return f, nil
}

// windKmh turns 7Timer's wind scale into a typical speed for its band.
var windKmh = [...]float64{0, 0, 7, 20, 34, 50, 75, 103, 120}

func (r *sevenTimerCivil) forecast(loc Location, now time.Time) (*Forecast, error) {
	start, err := time.ParseInLocation("2006010215", r.Init, time.UTC)
	if err != nil {
		return nil, fmt.Errorf("init time: %w", err)
	}
	tz := loc.TZ()
	f := &Forecast{Location: loc}
	var best time.Duration = -1
	for _, p := range r.Dataseries {
		t := start.Add(time.Duration(p.Timepoint) * time.Hour).In(tz)
		cond, isDay := from7Timer(p.Weather)
		f.Hours = append(f.Hours, Hour{Time: t, Condition: cond, IsDay: isDay, Temp: float64(p.Temp)})

		// The step closest to now stands in for the current weather.
		diff := t.Sub(now).Abs()
		if diff <= 90*time.Minute && (best < 0 || diff < best) {
			best = diff
			cur := &Current{Time: t, Condition: cond, IsDay: isDay, Temp: float64(p.Temp)}
			if h, err := strconv.ParseFloat(strings.TrimSuffix(p.Humidity, "%"), 64); err == nil {
				cur.Humidity = &h
			}
			if p.Wind.Speed >= 0 && p.Wind.Speed < len(windKmh) {
				cur.Wind = ptr(windKmh[p.Wind.Speed])
			}
			f.Current = cur
		}
	}
	return f, nil
}

func (r *sevenTimerCivilLight) days(loc Location) []Day {
	tz := loc.TZ()
	var days []Day
	for _, d := range r.Dataseries {
		date, err := time.ParseInLocation("20060102", strconv.Itoa(d.Date), tz)
		if err != nil {
			continue
		}
		cond, _ := from7Timer(d.Weather)
		days = append(days, Day{Date: date, Condition: cond, Min: float64(d.Temp.Min), Max: float64(d.Temp.Max)})
	}
	return days
}
