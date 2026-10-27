package weather

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

var barcelona = Location{Name: "Barcelona", Latitude: 41.3888, Longitude: 2.159, Timezone: "Europe/Madrid"}

func readFixture(t *testing.T, name string, v any) {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatal(err)
	}
}

func TestConditionCodes(t *testing.T) {
	for _, tc := range []struct {
		name string
		got  Condition
		want Condition
	}{
		{"WMO 0", fromWMO(0), Clear},
		{"WMO 2", fromWMO(2), PartlyCloudy},
		{"WMO 63", fromWMO(63), Rain},
		{"WMO 66", fromWMO(66), Sleet},
		{"WMO 99", fromWMO(99), Thunder},
		{"WMO 7", fromWMO(7), Unknown},
		{"WWO 113", fromWWO(113), Clear},
		{"WWO 296", fromWWO(296), LightRain},
		{"WWO 338", fromWWO(338), Snow},
		{"WWO 389", fromWWO(389), Thunder},
	} {
		if tc.got != tc.want {
			t.Errorf("%s = %v, want %v", tc.name, tc.got, tc.want)
		}
	}

	for kind, want := range map[string]struct {
		cond  Condition
		isDay bool
	}{
		"clearday":     {Clear, true},
		"pcloudynight": {PartlyCloudy, false},
		"mcloudy":      {Cloudy, true},
		"humidnight":   {Fog, false},
		"ishowerday":   {LightRain, true},
		"rainsnow":     {Sleet, true},
		"tsrainnight":  {Thunder, false},
		"whatever":     {Unknown, true},
	} {
		cond, isDay := from7Timer(kind)
		if cond != want.cond || isDay != want.isDay {
			t.Errorf("from7Timer(%q) = %v, %v; want %v, %v", kind, cond, isDay, want.cond, want.isDay)
		}
	}
}

func TestOpenMeteo(t *testing.T) {
	var r openMeteoResponse
	readFixture(t, "open-meteo.json", &r)
	f, err := r.forecast(barcelona)
	if err != nil {
		t.Fatal(err)
	}

	c := f.Current
	if c == nil || c.Temp != 19.4 || c.Condition != PartlyCloudy || !c.IsDay || *c.FeelsLike != 18.2 || *c.Wind != 11.3 {
		t.Errorf("current = %+v", c)
	}
	if want := "2026-10-27 10:45 +0100"; c.Time.Format("2006-01-02 15:04 -0700") != want {
		t.Errorf("current time = %v, want %s", c.Time, want)
	}

	if len(f.Hours) != 48 {
		t.Fatalf("got %d hours, want 48", len(f.Hours))
	}
	if h := f.Hours[31]; h.Condition != Thunder || *h.PrecipChance != 80 || !h.IsDay {
		t.Errorf("hour 31 = %+v", h)
	}
	if f.Hours[47].PrecipChance != nil {
		t.Errorf("a null precipitation chance should stay unknown")
	}

	if len(f.Days) != 7 {
		t.Fatalf("got %d days, want 7", len(f.Days))
	}
	d := f.Days[1]
	if d.Condition != Rain || d.Min != 13.5 || d.Max != 19 || *d.PrecipChance != 70 {
		t.Errorf("day 1 = %+v", d)
	}
	if d.Sunrise.Format("15:04") != "07:17" || d.Date.Format("2006-01-02") != "2026-10-28" {
		t.Errorf("day 1 date %v, sunrise %v", d.Date, d.Sunrise)
	}
}

func TestWttr(t *testing.T) {
	var r wttrResponse
	readFixture(t, "wttr-j2.json", &r)
	now := time.Date(2026, 10, 27, 9, 42, 0, 0, time.UTC)
	f, err := r.forecast(barcelona, now)
	if err != nil {
		t.Fatal(err)
	}
	c := f.Current
	if c.Temp != 19 || c.Condition != Clear || !c.IsDay || *c.Humidity != 79 || *c.Wind != 10 || *c.FeelsLike != 19 {
		t.Errorf("current = %+v", c)
	}
	// Observed at 09:33 UTC, shown in Barcelona time.
	if c.Time.Format("15:04 MST") != "10:33 CET" {
		t.Errorf("observed at %v", c.Time)
	}
	if f.Hours != nil || f.Days != nil {
		t.Errorf("wttr should only give the current weather")
	}

	// Just after midnight UTC, an observation from before midnight belongs
	// to the day before.
	f, _ = r.forecast(barcelona, time.Date(2026, 10, 27, 0, 5, 0, 0, time.UTC))
	if f.Current.Time.UTC().Day() != 26 {
		t.Errorf("observed at %v, want October 26 UTC", f.Current.Time.UTC())
	}
}

func TestSevenTimer(t *testing.T) {
	var civil sevenTimerCivil
	var light sevenTimerCivilLight
	readFixture(t, "7timer-civil.json", &civil)
	readFixture(t, "7timer-civillight.json", &light)

	// The fixture starts at 2026-10-27 00:00 UTC in 3-hour steps.
	now := time.Date(2026, 10, 27, 10, 0, 0, 0, time.UTC)
	f, err := civil.forecast(barcelona, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Hours) != len(civil.Dataseries) {
		t.Errorf("got %d hours, want %d", len(f.Hours), len(civil.Dataseries))
	}
	if f.Current == nil || !f.Current.Time.Equal(time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("current should be the 09:00 UTC step, got %+v", f.Current)
	}
	if f.Current.Humidity == nil || f.Current.Wind == nil {
		t.Errorf("current lacks humidity or wind: %+v", f.Current)
	}

	// Far from any step there is no current weather.
	f, _ = civil.forecast(barcelona, now.AddDate(0, 1, 0))
	if f.Current != nil {
		t.Errorf("current = %+v, want none a month later", f.Current)
	}

	days := light.days(barcelona)
	if len(days) != len(light.Dataseries) || days[0].Date.Format("2006-01-02") != "2026-10-27" {
		t.Errorf("days = %+v", days)
	}
}

func TestSunTimes(t *testing.T) {
	// London on the summer solstice: sunrise 04:43, sunset 21:21 BST.
	tz, _ := time.LoadLocation("Europe/London")
	rise, set := SunTimes(time.Date(2026, 6, 21, 15, 0, 0, 0, tz), 51.5074, -0.1278)
	near := func(got *time.Time, h, m int) bool {
		want := time.Date(2026, 6, 21, h, m, 0, 0, tz)
		return got != nil && got.Sub(want).Abs() <= 3*time.Minute
	}
	if !near(rise, 4, 43) || !near(set, 21, 21) {
		t.Errorf("sunrise %v, sunset %v", rise, set)
	}

	// Polar night in Tromsø.
	rise, set = SunTimes(time.Date(2026, 12, 21, 0, 0, 0, 0, time.UTC), 69.65, 18.96)
	if rise != nil || set != nil {
		t.Errorf("Tromsø in December: sunrise %v, sunset %v; want none", rise, set)
	}
	if isDaytime(time.Date(2026, 12, 21, 12, 0, 0, 0, time.UTC), 69.65, 18.96) {
		t.Errorf("Tromsø in December should count as night")
	}
}
