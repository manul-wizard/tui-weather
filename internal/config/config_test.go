package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/manul-wizard/tui-weather/internal/weather"
)

func TestLoadCreatesTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "config.toml")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cfg, Default()) {
		t.Errorf("got %+v, want the defaults", cfg)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("the file was not created: %v", err)
	}
}

func TestSaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	cfg := Default()
	cfg.Language, cfg.Units, cfg.Icons = "ru", "imperial", "nerd"
	cfg.Providers = []string{"7timer", "wttr"}
	// A whole-number coordinate and quotes in the name must survive.
	cfg.Location = weather.Location{Name: `Kyiv "centre"`, Latitude: 50, Longitude: 30.5234, Timezone: "Europe/Kyiv"}
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, cfg) {
		t.Errorf("got %+v, want %+v", got, cfg)
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "# Weather icons:") {
		t.Errorf("the comments were lost:\n%s", data)
	}
}

func TestMissingSettingsKeepTheirDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	os.WriteFile(path, []byte("icons = \"ascii\"\n"), 0o644)
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	want := Default()
	want.Icons = "ascii"
	if !reflect.DeepEqual(cfg, want) {
		t.Errorf("got %+v, want %+v", cfg, want)
	}
}

func TestInvalidSettings(t *testing.T) {
	for name, content := range map[string]string{
		"language":  `language = "de"`,
		"units":     `units = "kelvin"`,
		"icons":     `icons = "fancy"`,
		"providers": `providers = ["open-meteo", "yr"]`,
		"empty":     `providers = []`,
		"timeout":   `timeout_seconds = 0`,
		"latitude":  "[location]\nlatitude = 95.0",
		"syntax":    `language = `,
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.toml")
			os.WriteFile(path, []byte(content+"\n"), 0o644)
			if _, err := Load(path); err == nil {
				t.Errorf("%s: want an error", content)
			}
		})
	}
}
