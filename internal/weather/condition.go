package weather

import "strings"

// fromWMO maps a WMO weather interpretation code, as used by Open-Meteo.
func fromWMO(code int) Condition {
	switch code {
	case 0:
		return Clear
	case 1, 2:
		return PartlyCloudy
	case 3:
		return Cloudy
	case 45, 48:
		return Fog
	case 51, 53, 55:
		return LightRain
	case 61, 63, 80, 81:
		return Rain
	case 65, 82:
		return HeavyRain
	case 56, 57, 66, 67:
		return Sleet
	case 71, 73, 75, 77, 85, 86:
		return Snow
	case 95, 96, 99:
		return Thunder
	}
	return Unknown
}

// fromWWO maps a World Weather Online condition code, as used by wttr.in.
func fromWWO(code int) Condition {
	switch code {
	case 113:
		return Clear
	case 116:
		return PartlyCloudy
	case 119, 122:
		return Cloudy
	case 143, 248, 260:
		return Fog
	case 176, 263, 266, 293, 296, 353:
		return LightRain
	case 299, 302, 356:
		return Rain
	case 305, 308, 359:
		return HeavyRain
	case 179, 182, 185, 281, 284, 311, 314, 317, 320, 350, 362, 365, 374, 377:
		return Sleet
	case 227, 230, 323, 326, 329, 332, 335, 338, 368, 371:
		return Snow
	case 200, 386, 389, 392, 395:
		return Thunder
	}
	return Unknown
}

// from7Timer maps a 7Timer weather type such as "pcloudyday" or "tsrain".
// It also reports whether the type is a daytime one; types without a day or
// night suffix count as daytime.
func from7Timer(kind string) (Condition, bool) {
	isDay := !strings.HasSuffix(kind, "night")
	kind = strings.TrimSuffix(strings.TrimSuffix(kind, "day"), "night")
	switch kind {
	case "clear":
		return Clear, isDay
	case "pcloudy":
		return PartlyCloudy, isDay
	case "mcloudy", "cloudy":
		return Cloudy, isDay
	case "humid":
		return Fog, isDay
	case "lightrain", "oshower", "ishower":
		return LightRain, isDay
	case "rain":
		return Rain, isDay
	case "rainsnow":
		return Sleet, isDay
	case "lightsnow", "snow":
		return Snow, isDay
	case "ts", "tsrain":
		return Thunder, isDay
	}
	return Unknown, isDay
}
