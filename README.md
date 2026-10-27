# tui-weather

[![CI](https://github.com/manul-wizard/tui-weather/actions/workflows/ci.yml/badge.svg)](https://github.com/manul-wizard/tui-weather/actions/workflows/ci.yml)
[![License: Apache-2.0](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

The weather in your terminal: right now, hour by hour and for the week, with a small menu to switch between them and to change the city.

- **No account, no API key.** Forecasts come from free public services.
- **Keeps working when a service is down or blocked.** It asks several services at once and takes each part of the forecast from the first one that answers.
- **Works offline for a while.** The last forecast is cached and shown, marked as old, when nothing answers.
- **Your look.** Nerd Font, emoji or plain ASCII icons; English or Russian; metric or imperial.

## Install

Download an archive for your system from [Releases](https://github.com/manul-wizard/tui-weather/releases) and put `tui-weather` somewhere on your `PATH`, or build it with [Go](https://go.dev/dl/):

```sh
go install github.com/manul-wizard/tui-weather@latest
```

## Use

```sh
tui-weather                  # the interactive menu
tui-weather now              # print the current weather
tui-weather hourly           # print the next 24 hours
tui-weather daily            # print the coming days
tui-weather line             # one line, for a shell prompt or a status bar
tui-weather city Barcelona   # find a city and make it the default
tui-weather config           # where the settings file is
```

Every command takes `--city <name>` to look at another place once without changing the default, and `--refresh` to skip the cache.

In the menu, pick a view with the arrows and Enter or with its number. In a view, `←` and `→` switch between now, hourly and daily, `r` refreshes, `Esc` goes back to the menu and `q` quits.

## Settings

The settings file is created on the first run: `%AppData%\tui-weather\config.toml` on Windows, `~/.config/tui-weather/config.toml` on Linux and `~/Library/Application Support/tui-weather/config.toml` on macOS. Every setting is explained in the file itself.

```toml
language = "en"        # en or ru
units = "metric"       # metric or imperial
icons = "emoji"        # nerd, emoji or ascii
cache_minutes = 15
timeout_seconds = 10
providers = ["open-meteo", "wttr", "7timer"]

[location]
name = "Barcelona, Catalonia, Spain"
latitude = 41.3888
longitude = 2.159
timezone = "Europe/Madrid"
```

The `nerd` icons need a [Nerd Font](https://www.nerdfonts.com) in the terminal, such as Cascadia Code NF or JetBrainsMono Nerd Font.

To change the city, use the menu or `tui-weather city <name>`: both look the name up and let you pick the right place when there are several. You can also edit the coordinates by hand.

## Weather services

| Service | Gives | |
|---|---|---|
| [Open-Meteo](https://open-meteo.com) | now, hourly for 48 hours, daily for 7 days; also finds cities | Data under [CC BY 4.0](https://open-meteo.com/en/license) |
| [wttr.in](https://wttr.in) | now | |
| [7Timer!](https://www.7timer.info) | 3-hour steps and daily for about a week | |

They are asked in parallel, and each part of the forecast comes from the first service in `providers` that has it. Sunrise and sunset are worked out from the coordinates when a service does not give them.

## Development

```sh
go test ./...
go vet ./...
go run . now
```

Releases are built by [GoReleaser](https://goreleaser.com) when a `v*` tag is pushed.

## License

[Apache License 2.0](LICENSE). A [Manul Wizard](https://github.com/manul-wizard) project.
