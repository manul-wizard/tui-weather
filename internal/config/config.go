// Package config loads and saves the settings file.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"text/template"

	"github.com/BurntSushi/toml"

	"github.com/manul-wizard/tui-weather/internal/weather"
)

// Config is the settings file.
type Config struct {
	Language       string           `toml:"language"`
	Units          string           `toml:"units"`
	Icons          string           `toml:"icons"`
	CacheMinutes   int              `toml:"cache_minutes"`
	TimeoutSeconds int              `toml:"timeout_seconds"`
	Providers      []string         `toml:"providers"`
	Location       weather.Location `toml:"location"`
}

// Allowed values of the enumerated settings.
var (
	Languages = []string{"en", "ru"}
	Units     = []string{"metric", "imperial"}
	IconSets  = []string{"nerd", "emoji", "ascii"}
)

// Default returns the settings used when there is no file yet.
func Default() Config {
	lang := systemLanguage()
	name := "Barcelona, Catalonia, Spain"
	if lang == "ru" {
		name = "Барселона, Каталония, Испания"
	}
	return Config{
		Language:       lang,
		Units:          "metric",
		Icons:          "emoji",
		CacheMinutes:   15,
		TimeoutSeconds: 10,
		Providers:      weather.ProviderNames(),
		Location: weather.Location{
			Name:      name,
			Latitude:  41.3888,
			Longitude: 2.159,
			Timezone:  "Europe/Madrid",
		},
	}
}

// DefaultPath returns the settings path in the user's config directory.
func DefaultPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "tui-weather.toml"
	}
	return filepath.Join(dir, "tui-weather", "config.toml")
}

// Load reads the settings at path. When the file does not exist yet, it is
// created with the defaults. Settings missing from the file keep their
// defaults.
func Load(path string) (Config, error) {
	cfg := Default()
	_, err := toml.DecodeFile(path, &cfg)
	if errors.Is(err, fs.ErrNotExist) {
		return cfg, Save(path, cfg)
	}
	if err != nil {
		return cfg, fmt.Errorf("%s: %w", path, err)
	}
	if err := cfg.Validate(); err != nil {
		return cfg, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}

// Validate checks that every setting has an allowed value.
func (c Config) Validate() error {
	check := func(key, value string, allowed []string) error {
		if !slices.Contains(allowed, value) {
			return fmt.Errorf("%s = %q, expected one of: %s", key, value, strings.Join(allowed, ", "))
		}
		return nil
	}
	if err := check("language", c.Language, Languages); err != nil {
		return err
	}
	if err := check("units", c.Units, Units); err != nil {
		return err
	}
	if err := check("icons", c.Icons, IconSets); err != nil {
		return err
	}
	if len(c.Providers) == 0 {
		return fmt.Errorf("providers is empty, expected some of: %s", strings.Join(weather.ProviderNames(), ", "))
	}
	if _, err := weather.ProvidersByName(c.Providers); err != nil {
		return fmt.Errorf("providers: %w", err)
	}
	if c.TimeoutSeconds <= 0 {
		return fmt.Errorf("timeout_seconds must be positive")
	}
	if c.CacheMinutes < 0 {
		return fmt.Errorf("cache_minutes must not be negative")
	}
	l := c.Location
	if l.Latitude < -90 || l.Latitude > 90 || l.Longitude < -180 || l.Longitude > 180 {
		return fmt.Errorf("location: latitude must be within ±90 and longitude within ±180")
	}
	return nil
}

// fileTemplate keeps the comments in the file when the program saves it, for
// example after the city is changed from the menu.
var fileTemplate = template.Must(template.New("config").Funcs(template.FuncMap{
	"q":    quote,
	"f":    float,
	"list": func(s []string) string { return "[" + strings.Join(mapQuote(s), ", ") + "]" },
}).Parse(`# tui-weather settings. Edit by hand, or change the city from the menu.

# Interface language: en or ru.
language = {{q .Language}}

# metric (°C, km/h) or imperial (°F, mph).
units = {{q .Units}}

# Weather icons: nerd (needs a Nerd Font in the terminal), emoji or ascii.
icons = {{q .Icons}}

# How long a forecast is reused before asking the weather services again.
cache_minutes = {{.CacheMinutes}}

# How long to wait for the weather services.
timeout_seconds = {{.TimeoutSeconds}}

# Weather services, most preferred first. Each part of the forecast (now,
# hourly, daily) comes from the first service that has it.
#   open-meteo  everything, hourly; https://open-meteo.com
#   wttr        current weather only; https://wttr.in
#   7timer      3-hour steps and daily; https://www.7timer.info
providers = {{list .Providers}}

# The default city. "tui-weather city <name>" or the menu finds a place and
# fills this in; the name is only a label.
[location]
name = {{q .Location.Name}}
latitude = {{f .Location.Latitude}}
longitude = {{f .Location.Longitude}}
timezone = {{q .Location.Timezone}}
`))

// Save writes the settings to path, creating its directory.
func Save(path string, c Config) error {
	var buf bytes.Buffer
	if err := fileTemplate.Execute(&buf, c); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func quote(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

// float always writes a decimal point, so that TOML reads the value back as
// a float and not an integer.
func float(v float64) string {
	s := strconv.FormatFloat(v, 'f', -1, 64)
	if !strings.ContainsAny(s, ".eE") {
		s += ".0"
	}
	return s
}

func mapQuote(s []string) []string {
	out := make([]string, len(s))
	for i, v := range s {
		out[i] = quote(v)
	}
	return out
}
