// Package ui renders forecasts as text and runs the interactive menu.
package ui

import "github.com/manul-wizard/tui-weather/internal/weather"

// Text holds the interface strings of one language.
type Text struct {
	Now, Hourly, Daily, ChangeCity, Refresh, Quit string

	FeelsLike, Updated, Offline, Today, Tomorrow    string
	Loading, Searching, CityPrompt, CityPlaceholder string
	NotFound, Saved, FetchFailed, SearchFailed      string
	NoCurrent, NoHours, NoDays                      string
	SourceNow, SourceHourly, SourceDaily            string

	HintMenu, HintView, HintSearch, HintResults string

	Weekdays   [7]string // Sunday first, as in time.Weekday
	Conditions map[weather.Condition]string
	KmH        string
}

// TextFor returns the strings of a language, English when it is unknown.
func TextFor(lang string) Text {
	if lang == "ru" {
		return russian
	}
	return english
}

var english = Text{
	Now: "Now", Hourly: "Hourly", Daily: "Daily", ChangeCity: "Change city", Refresh: "Refresh", Quit: "Quit",

	FeelsLike: "feels like %s", Updated: "Updated %s", Offline: "Offline: showing the forecast from %s",
	Today: "Today", Tomorrow: "Tomorrow",
	Loading: "Loading the forecast…", Searching: "Searching…",
	CityPrompt: "City", CityPlaceholder: "for example, Barcelona",
	NotFound: "Nothing found for %q", Saved: "Saved: %s",
	FetchFailed: "Could not get the forecast", SearchFailed: "Search failed",
	NoCurrent: "No service reported the current weather.",
	NoHours:   "No service has an hourly forecast for this place.",
	NoDays:    "No service has a daily forecast for this place.",
	SourceNow: "now", SourceHourly: "hourly", SourceDaily: "daily",

	HintMenu:    "↑↓ choose · enter open · q quit",
	HintView:    "←→ switch · ↑↓ scroll · r refresh · esc menu · q quit",
	HintSearch:  "enter search · esc back",
	HintResults: "↑↓ choose · enter save · esc back",

	Weekdays: [7]string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"},
	Conditions: map[weather.Condition]string{
		weather.Unknown:      "Unknown",
		weather.Clear:        "Clear",
		weather.PartlyCloudy: "Partly cloudy",
		weather.Cloudy:       "Cloudy",
		weather.Fog:          "Fog",
		weather.LightRain:    "Light rain",
		weather.Rain:         "Rain",
		weather.HeavyRain:    "Heavy rain",
		weather.Sleet:        "Sleet",
		weather.Snow:         "Snow",
		weather.Thunder:      "Thunderstorm",
	},
	KmH: "km/h",
}

var russian = Text{
	Now: "Сейчас", Hourly: "По часам", Daily: "По дням", ChangeCity: "Сменить город", Refresh: "Обновить", Quit: "Выход",

	FeelsLike: "ощущается как %s", Updated: "Обновлено %s", Offline: "Нет связи: прогноз от %s",
	Today: "Сегодня", Tomorrow: "Завтра",
	Loading: "Загружаю прогноз…", Searching: "Ищу…",
	CityPrompt: "Город", CityPlaceholder: "например, Барселона",
	NotFound: "По запросу %q ничего не найдено", Saved: "Сохранено: %s",
	FetchFailed: "Не удалось получить прогноз", SearchFailed: "Не удалось выполнить поиск",
	NoCurrent: "Ни один сервис не сообщил текущую погоду.",
	NoHours:   "Ни у одного сервиса нет почасового прогноза для этого места.",
	NoDays:    "Ни у одного сервиса нет прогноза по дням для этого места.",
	SourceNow: "сейчас", SourceHourly: "по часам", SourceDaily: "по дням",

	HintMenu:    "↑↓ выбор · enter открыть · q выход",
	HintView:    "←→ вид · ↑↓ прокрутка · r обновить · esc меню · q выход",
	HintSearch:  "enter найти · esc назад",
	HintResults: "↑↓ выбор · enter сохранить · esc назад",

	Weekdays: [7]string{"Вс", "Пн", "Вт", "Ср", "Чт", "Пт", "Сб"},
	Conditions: map[weather.Condition]string{
		weather.Unknown:      "Неизвестно",
		weather.Clear:        "Ясно",
		weather.PartlyCloudy: "Переменная облачность",
		weather.Cloudy:       "Облачно",
		weather.Fog:          "Туман",
		weather.LightRain:    "Небольшой дождь",
		weather.Rain:         "Дождь",
		weather.HeavyRain:    "Ливень",
		weather.Sleet:        "Мокрый снег",
		weather.Snow:         "Снег",
		weather.Thunder:      "Гроза",
	},
	KmH: "км/ч",
}
