package ui

import "github.com/manul-wizard/tui-weather/internal/weather"

// Icons is a set of weather symbols.
type Icons struct {
	day, night                                map[weather.Condition]string
	Humidity, Wind, Sunrise, Sunset, Umbrella string
}

// Condition returns the symbol of a condition at day or night.
func (i Icons) Condition(c weather.Condition, isDay bool) string {
	if !isDay {
		if s, ok := i.night[c]; ok {
			return s
		}
	}
	if s, ok := i.day[c]; ok {
		return s
	}
	return i.day[weather.Unknown]
}

// IconsFor returns an icon set by its name in the settings.
func IconsFor(name string) Icons {
	switch name {
	case "nerd":
		return nerdIcons
	case "ascii":
		return asciiIcons
	}
	return emojiIcons
}

// nerdIcons are Weather Icons glyphs as packed into Nerd Fonts.
var nerdIcons = Icons{
	day: map[weather.Condition]string{
		weather.Unknown:      "",
		weather.Clear:        "",
		weather.PartlyCloudy: "",
		weather.Cloudy:       "",
		weather.Fog:          "",
		weather.LightRain:    "",
		weather.Rain:         "",
		weather.HeavyRain:    "",
		weather.Sleet:        "",
		weather.Snow:         "",
		weather.Thunder:      "",
	},
	night: map[weather.Condition]string{
		weather.Clear:        "",
		weather.PartlyCloudy: "",
	},
	Humidity: "", Wind: "", Sunrise: "", Sunset: "", Umbrella: "",
}

var emojiIcons = Icons{
	day: map[weather.Condition]string{
		weather.Unknown:      "❔",
		weather.Clear:        "☀️",
		weather.PartlyCloudy: "⛅",
		weather.Cloudy:       "☁️",
		weather.Fog:          "🌫️",
		weather.LightRain:    "🌦️",
		weather.Rain:         "🌧️",
		weather.HeavyRain:    "🌧️",
		weather.Sleet:        "🌨️",
		weather.Snow:         "❄️",
		weather.Thunder:      "⛈️",
	},
	night: map[weather.Condition]string{
		weather.Clear:     "🌙",
		weather.LightRain: "🌧️",
	},
	Humidity: "💧", Wind: "💨", Sunrise: "🌅", Sunset: "🌇", Umbrella: "☔",
}

var asciiIcons = Icons{
	day: map[weather.Condition]string{
		weather.Unknown:      "??",
		weather.Clear:        "O ",
		weather.PartlyCloudy: "o~",
		weather.Cloudy:       "~~",
		weather.Fog:          "==",
		weather.LightRain:    "' ",
		weather.Rain:         "''",
		weather.HeavyRain:    "||",
		weather.Sleet:        "*'",
		weather.Snow:         "**",
		weather.Thunder:      "/!",
	},
	night: map[weather.Condition]string{
		weather.Clear:        "C ",
		weather.PartlyCloudy: "C~",
	},
	Humidity: "hum", Wind: "wind", Sunrise: "rise", Sunset: "set", Umbrella: "rain",
}
