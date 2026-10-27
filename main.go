// Command tui-weather shows the weather in the terminal.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"
	"time"
	_ "time/tzdata" // timezones on systems without a zoneinfo database, such as Windows

	"github.com/mattn/go-isatty"

	"github.com/manul-wizard/tui-weather/internal/config"
	"github.com/manul-wizard/tui-weather/internal/ui"
	"github.com/manul-wizard/tui-weather/internal/weather"
)

// version is set at build time by the release workflow.
var version = ""

const usage = `tui-weather shows the weather in the terminal.

Usage:
  tui-weather [flags]               open the interactive menu
  tui-weather now [flags]           print the current weather
  tui-weather hourly [flags]        print the next 24 hours
  tui-weather daily [flags]         print the coming days
  tui-weather line [flags]          print one line, for shell prompts
  tui-weather city <name> [flags]   find a city and make it the default
  tui-weather config                print the path of the settings file

Flags:
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	command := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		command, args = args[0], args[1:]
	}

	fs := flag.NewFlagSet("tui-weather", flag.ContinueOnError)
	fs.SetOutput(stderr)
	configPath := fs.String("config", config.DefaultPath(), "settings `file`")
	city := fs.String("city", "", "show this `city` instead of the default one, without saving it")
	refresh := fs.Bool("refresh", false, "ask the weather services even if the cached forecast is fresh")
	showVersion := fs.Bool("version", false, "print the version")
	fs.Usage = func() {
		fmt.Fprint(stderr, usage)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if *showVersion {
		fmt.Fprintln(stdout, "tui-weather", buildVersion())
		return 0
	}
	weather.UserAgent = fmt.Sprintf("tui-weather/%s (+https://github.com/manul-wizard/tui-weather)", buildVersion())

	if command == "config" {
		fmt.Fprintln(stdout, *configPath)
		return 0
	}
	if command == "help" {
		fs.Usage()
		return 0
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintln(stderr, "tui-weather:", err)
		return 1
	}
	opts := weather.Options{
		Location:  cfg.Location,
		Providers: cfg.Providers,
		Timeout:   time.Duration(cfg.TimeoutSeconds) * time.Second,
		CacheTTL:  time.Duration(cfg.CacheMinutes) * time.Minute,
		CacheFile: weather.DefaultCacheFile(),
		Refresh:   *refresh,
	}

	var override *weather.Location
	if *city != "" {
		loc, err := findCity(*city, cfg.Language, opts.Timeout)
		if err != nil {
			fmt.Fprintln(stderr, "tui-weather:", err)
			return 1
		}
		override, opts.Location = &loc, loc
	}

	interactive := isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
	switch command {
	case "":
		if interactive {
			return runApp(ui.App{Config: cfg, ConfigPath: *configPath, Location: override, Options: opts}, stderr)
		}
		command = "now"
	case "city":
		query := strings.Join(fs.Args(), " ")
		if query == "" {
			fmt.Fprintln(stderr, "tui-weather: city needs a name, for example: tui-weather city Barcelona")
			return 2
		}
		if !interactive {
			fmt.Fprintln(stderr, "tui-weather: city needs a terminal to choose from the places found")
			return 1
		}
		return runApp(ui.App{Config: cfg, ConfigPath: *configPath, Options: opts, Search: query}, stderr)
	}

	r := ui.Renderer{
		Text:     ui.TextFor(cfg.Language),
		Icons:    ui.IconsFor(cfg.Icons),
		Imperial: cfg.Units == "imperial",
	}
	var render func(*weather.Forecast) string
	switch command {
	case "now":
		render = r.Current
	case "hourly":
		render = r.Hourly
	case "daily":
		render = r.Daily
	case "line":
		render = r.Line
	default:
		fmt.Fprintf(stderr, "tui-weather: unknown command %q\n\n", command)
		fs.Usage()
		return 2
	}

	f, err := weather.Get(context.Background(), opts)
	if err != nil {
		fmt.Fprintln(stderr, "tui-weather: could not get the forecast:")
		fmt.Fprintln(stderr, err)
		return 1
	}
	if command == "line" {
		fmt.Fprintln(stdout, render(f))
		return 0
	}
	fmt.Fprintf(stdout, "%s\n\n%s\n\n%s\n", r.Header(f), ui.Indent(render(f), "  "), r.Footer(f))
	return 0
}

func runApp(app ui.App, stderr io.Writer) int {
	if err := ui.Run(app); err != nil {
		fmt.Fprintln(stderr, "tui-weather:", err)
		return 1
	}
	return 0
}

// findCity returns the best match for a city name.
func findCity(name, language string, timeout time.Duration) (weather.Location, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	places, err := weather.Search(ctx, weather.NewClient(timeout), name, language)
	if err != nil {
		return weather.Location{}, fmt.Errorf("could not search for %q: %w", name, err)
	}
	if len(places) == 0 {
		return weather.Location{}, fmt.Errorf("no place called %q found", name)
	}
	return places[0], nil
}

func buildVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}
