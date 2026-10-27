package weather

import (
	"math"
	"time"
)

// SunTimes returns sunrise and sunset on the given date at the given point,
// accurate to a minute or two. Both are nil during polar day and night.
//
// It follows the sunrise equation from
// https://en.wikipedia.org/wiki/Sunrise_equation.
func SunTimes(date time.Time, lat, lon float64) (sunrise, sunset *time.Time) {
	const j2000 = 2451545.0
	rad := math.Pi / 180

	// The equation counts days from midnight UTC; rounding up lands on the
	// noon of the date.
	midnight := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	n := math.Ceil(julianDay(midnight) - j2000 + 0.0008)
	meanNoon := n - lon/360
	anomaly := math.Mod(357.5291+0.98560028*meanNoon, 360)
	center := 1.9148*math.Sin(anomaly*rad) + 0.02*math.Sin(2*anomaly*rad) + 0.0003*math.Sin(3*anomaly*rad)
	longitude := math.Mod(anomaly+center+180+102.9372, 360)
	transit := j2000 + meanNoon + 0.0053*math.Sin(anomaly*rad) - 0.0069*math.Sin(2*longitude*rad)
	declination := math.Asin(math.Sin(longitude*rad) * math.Sin(23.4397*rad))

	cosHourAngle := (math.Sin(-0.833*rad) - math.Sin(lat*rad)*math.Sin(declination)) /
		(math.Cos(lat*rad) * math.Cos(declination))
	if cosHourAngle < -1 || cosHourAngle > 1 {
		return nil, nil
	}
	hourAngle := math.Acos(cosHourAngle) / rad

	rise := fromJulianDay(transit - hourAngle/360).In(date.Location())
	set := fromJulianDay(transit + hourAngle/360).In(date.Location())
	return &rise, &set
}

// isDaytime reports whether t falls between sunrise and sunset. During polar
// day and night it guesses from the season.
func isDaytime(t time.Time, lat, lon float64) bool {
	rise, set := SunTimes(t, lat, lon)
	if rise == nil {
		summer := t.Month() >= time.April && t.Month() <= time.September
		return summer == (lat > 0)
	}
	return !t.Before(*rise) && t.Before(*set)
}

func julianDay(t time.Time) float64 {
	return float64(t.Unix())/86400 + 2440587.5
}

func fromJulianDay(jd float64) time.Time {
	return time.Unix(int64(math.Round((jd-2440587.5)*86400)), 0)
}
