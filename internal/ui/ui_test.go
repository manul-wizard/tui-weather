package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/manul-wizard/tui-weather/internal/config"
	"github.com/manul-wizard/tui-weather/internal/weather"
)

func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.Ascii) // compare plain text
	os.Exit(m.Run())
}

var tz, _ = time.LoadLocation("Europe/Madrid")

func at(day, hour int) time.Time { return time.Date(2026, 10, day, hour, 0, 0, 0, tz) }

func forecast() *weather.Forecast {
	f := &weather.Forecast{
		Location: weather.Location{Name: "Barcelona, Catalonia, Spain", Latitude: 41.39, Longitude: 2.16, Timezone: "Europe/Madrid"},
		Current: &weather.Current{
			Time: at(27, 10), Condition: weather.PartlyCloudy, IsDay: true, Temp: 19.4,
			FeelsLike: new(18.2), Humidity: new(71.0), Wind: new(11.3),
		},
		Sources:   weather.Sources{Current: "wttr", Hours: "7timer", Days: "7timer"},
		FetchedAt: at(27, 10),
	}
	for h := range 48 {
		f.Hours = append(f.Hours, weather.Hour{
			Time: at(27, 0).Add(time.Duration(h) * time.Hour), Condition: weather.Clear, IsDay: true, Temp: 15,
		})
	}
	for d := range 5 {
		f.Days = append(f.Days, weather.Day{Date: at(26+d, 0), Condition: weather.Rain, Min: 12, Max: 20, PrecipChance: new(40.0)})
	}
	sunrise, sunset := at(27, 7), at(27, 18)
	f.Days[1].Sunrise, f.Days[1].Sunset = &sunrise, &sunset
	return f
}

func renderer(lang string) Renderer {
	return Renderer{Text: TextFor(lang), Icons: IconsFor("ascii"), Now: func() time.Time { return at(27, 10).Add(30 * time.Minute) }}
}

func TestCurrent(t *testing.T) {
	got := renderer("ru").Current(forecast())
	for _, want := range []string{"o~  19°  Переменная облачность", "ощущается как 18°", "hum 71%", "wind 11 км/ч", "rise 07:00", "set 18:00", "Сегодня: 12° … 20°", "rain 40%"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestImperial(t *testing.T) {
	r := renderer("en")
	r.Imperial = true
	got := r.Current(forecast())
	for _, want := range []string{" 67° ", "feels like 65°", "wind 7 mph"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestHourlyStartsAtTheCurrentHour(t *testing.T) {
	got := renderer("en").Hourly(forecast())
	lines := strings.Split(got, "\n")
	if lines[0] != "Today, 27.10" || !strings.HasPrefix(strings.TrimSpace(lines[1]), "10:00") {
		t.Errorf("should start with today at 10:00:\n%s", got)
	}
	if !strings.Contains(got, "Tomorrow, 28.10") {
		t.Errorf("should go on into tomorrow:\n%s", got)
	}
	if n := strings.Count(got, "Clear"); n != 24 {
		t.Errorf("got %d hours, want 24", n)
	}
}

func TestDailySkipsPastDays(t *testing.T) {
	got := renderer("en").Daily(forecast())
	if strings.Contains(got, "26.10") || !strings.HasPrefix(got, "Tue 27.10") {
		t.Errorf("should start today:\n%s", got)
	}
	if n := len(strings.Split(got, "\n")); n != 4 {
		t.Errorf("got %d days, want 4", n)
	}
}

func TestFooterAndLine(t *testing.T) {
	r := renderer("en")
	f := forecast()
	if got, want := r.Footer(f), "Updated 10:00 · now: wttr, hourly: 7timer, daily: 7timer"; got != want {
		t.Errorf("footer = %q, want %q", got, want)
	}
	f.Sources = weather.Sources{Current: "open-meteo", Hours: "open-meteo", Days: "open-meteo"}
	if got, want := r.Footer(f), "Updated 10:00 · open-meteo"; got != want {
		t.Errorf("footer = %q, want %q", got, want)
	}
	if got, want := r.Line(f), "Barcelona o~ 19° Partly cloudy"; got != want {
		t.Errorf("line = %q, want %q", got, want)
	}
	f.Stale = true
	if got := r.Header(f); !strings.Contains(got, "Offline: showing the forecast from 27.10 10:00") {
		t.Errorf("header = %q", got)
	}
}

func key(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func send(m tea.Model, msgs ...tea.Msg) tea.Model {
	for _, msg := range msgs {
		m, _ = m.Update(msg)
	}
	return m
}

func testApp(t *testing.T) App {
	cfg := config.Default()
	cfg.Language, cfg.Icons = "en", "ascii"
	return App{Config: cfg, ConfigPath: filepath.Join(t.TempDir(), "config.toml")}
}

func TestMenuNavigation(t *testing.T) {
	var m tea.Model = newModel(testApp(t))
	m = send(m, tea.WindowSizeMsg{Width: 100, Height: 40}, forecastMsg{forecast: forecast()})

	view := m.View()
	if !strings.Contains(view, "› 1  Now") || !strings.Contains(view, "Barcelona o~ 19°") {
		t.Errorf("menu:\n%s", view)
	}

	m = send(m, key("down"), key("enter"))
	if view := m.View(); !strings.Contains(view, "[Hourly]") || !strings.Contains(view, "Today, 27.10") {
		t.Errorf("enter on the second item should open the hourly view:\n%s", view)
	}

	m = send(m, key("right"))
	if view := m.View(); !strings.Contains(view, "[Daily]") {
		t.Errorf("right should switch to the daily view:\n%s", view)
	}

	m = send(m, key("esc"), key("4"))
	if view := m.View(); !strings.Contains(view, "City: ") {
		t.Errorf("4 should open the city search:\n%s", view)
	}

	// Typing q into the search box does not quit.
	_, cmd := m.Update(key("q"))
	if cmd != nil {
		if _, quit := cmd().(tea.QuitMsg); quit {
			t.Errorf("q in the search box quit the program")
		}
	}
}

func TestChoosingACitySavesIt(t *testing.T) {
	app := testApp(t)
	var m tea.Model = newModel(app)
	m = send(m, key("4"))
	for _, r := range "Mon" {
		m = send(m, key(string(r)))
	}
	m = send(m, placesMsg{query: "Mon", places: []weather.Location{
		{Name: "Monaco", Latitude: 43.73, Longitude: 7.42, Timezone: "Europe/Monaco"},
		{Name: "Montevideo, Uruguay", Latitude: -34.9, Longitude: -56.19, Timezone: "America/Montevideo"},
	}})
	m = send(m, key("down"), key("enter"))

	cfg, err := config.Load(app.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Location.Name != "Montevideo, Uruguay" {
		t.Errorf("saved %+v, want Montevideo", cfg.Location)
	}
	if view := m.View(); !strings.Contains(view, "Saved: Montevideo, Uruguay") {
		t.Errorf("view:\n%s", view)
	}
}
