package ui

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/manul-wizard/tui-weather/internal/weather"
)

// Renderer turns a forecast into text.
type Renderer struct {
	Text     Text
	Icons    Icons
	Imperial bool
	// Now returns the current time; tests replace it.
	Now func() time.Time
}

var (
	titleStyle = lipgloss.NewStyle().Bold(true)
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	warnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
)

// tempColors go from frost to heat; each applies up to its limit in °C.
var tempColors = []struct {
	limit float64
	color lipgloss.Color
}{
	{-10, "#5f87ff"}, {0, "#87afff"}, {8, "#5fd7d7"}, {15, "#87d787"},
	{22, "#d7d75f"}, {28, "#ffaf5f"}, {math.Inf(1), "#ff5f5f"},
}

func tempColor(c float64) lipgloss.Color {
	for _, t := range tempColors {
		if c < t.limit {
			return t.color
		}
	}
	return tempColors[len(tempColors)-1].color
}

func conditionColor(c weather.Condition, isDay bool) lipgloss.Color {
	switch c {
	case weather.Clear, weather.PartlyCloudy:
		if !isDay {
			return "#afafd7"
		}
		return "#ffd75f"
	case weather.Cloudy, weather.Fog:
		return "#a8a8a8"
	case weather.LightRain, weather.Rain, weather.HeavyRain:
		return "#5fafff"
	case weather.Sleet, weather.Snow:
		return "#d7ffff"
	case weather.Thunder:
		return "#d787ff"
	}
	return "#808080"
}

func (r Renderer) now(tz *time.Location) time.Time {
	if r.Now != nil {
		return r.Now().In(tz)
	}
	return time.Now().In(tz)
}

func (r Renderer) temp(c float64) string {
	v := c
	if r.Imperial {
		v = c*9/5 + 32
	}
	return fmt.Sprintf("%d°", int(math.Round(v)))
}

func (r Renderer) coloredTemp(c float64) string {
	return lipgloss.NewStyle().Foreground(tempColor(c)).Render(r.temp(c))
}

func (r Renderer) wind(kmh float64) string {
	if r.Imperial {
		return fmt.Sprintf("%d mph", int(math.Round(kmh*0.621371)))
	}
	return fmt.Sprintf("%d %s", int(math.Round(kmh)), r.Text.KmH)
}

func (r Renderer) icon(c weather.Condition, isDay bool) string {
	return lipgloss.NewStyle().Foreground(conditionColor(c, isDay)).Render(r.Icons.Condition(c, isDay))
}

// pad right-pads s to width w terminal cells.
func pad(s string, w int) string {
	if gap := w - lipgloss.Width(s); gap > 0 {
		return s + strings.Repeat(" ", gap)
	}
	return s
}

// padLeft left-pads s to width w terminal cells.
func padLeft(s string, w int) string {
	if gap := w - lipgloss.Width(s); gap > 0 {
		return strings.Repeat(" ", gap) + s
	}
	return s
}

func (r Renderer) dayLabel(date, today time.Time) string {
	label := fmt.Sprintf("%s %02d.%02d", r.Text.Weekdays[date.Weekday()], date.Day(), date.Month())
	if sameDay(date, today) {
		return titleStyle.Render(label)
	}
	return label
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// Header is the place name, with a warning when the forecast is old.
func (r Renderer) Header(f *weather.Forecast) string {
	s := titleStyle.Render(f.Location.Name)
	if f.Stale {
		at := f.FetchedAt.In(f.Location.TZ()).Format("02.01 15:04")
		s += "\n" + warnStyle.Render(fmt.Sprintf(r.Text.Offline, at))
	}
	return s
}

// Footer says when the forecast was fetched and where from.
func (r Renderer) Footer(f *weather.Forecast) string {
	at := f.FetchedAt.In(f.Location.TZ()).Format("15:04")
	src := f.Sources
	var from string
	if src.Current == src.Hours && src.Hours == src.Days {
		from = src.Current
	} else {
		var parts []string
		for _, p := range []struct{ label, name string }{
			{r.Text.SourceNow, src.Current}, {r.Text.SourceHourly, src.Hours}, {r.Text.SourceDaily, src.Days},
		} {
			if p.name != "" {
				parts = append(parts, p.label+": "+p.name)
			}
		}
		from = strings.Join(parts, ", ")
	}
	return dimStyle.Render(fmt.Sprintf(r.Text.Updated, at) + " · " + from)
}

// Current renders the weather right now and a summary of today.
func (r Renderer) Current(f *weather.Forecast) string {
	c := f.Current
	if c == nil {
		return dimStyle.Render(r.Text.NoCurrent)
	}
	var b strings.Builder
	temp := lipgloss.NewStyle().Bold(true).Foreground(tempColor(c.Temp)).Render(r.temp(c.Temp))
	fmt.Fprintf(&b, "%s  %s  %s\n", r.icon(c.Condition, c.IsDay), temp, r.Text.Conditions[c.Condition])
	if c.FeelsLike != nil {
		b.WriteString(dimStyle.Render(fmt.Sprintf(r.Text.FeelsLike, r.temp(*c.FeelsLike))) + "\n")
	}

	var details []string
	if c.Humidity != nil {
		details = append(details, fmt.Sprintf("%s %d%%", r.Icons.Humidity, int(math.Round(*c.Humidity))))
	}
	if c.Wind != nil {
		details = append(details, r.Icons.Wind+" "+r.wind(*c.Wind))
	}
	if len(details) > 0 {
		b.WriteString("\n" + strings.Join(details, "   ") + "\n")
	}

	tz := f.Location.TZ()
	now := r.now(tz)
	if i := slices.IndexFunc(f.Days, func(d weather.Day) bool { return sameDay(d.Date, now) }); i >= 0 {
		d := f.Days[i]
		var today []string
		if d.Sunrise != nil && d.Sunset != nil {
			today = append(today,
				r.Icons.Sunrise+" "+d.Sunrise.In(tz).Format("15:04"),
				r.Icons.Sunset+" "+d.Sunset.In(tz).Format("15:04"))
		}
		if len(today) > 0 {
			b.WriteString(strings.Join(today, "   ") + "\n")
		}
		line := fmt.Sprintf("%s: %s … %s", r.Text.Today, r.coloredTemp(d.Min), r.coloredTemp(d.Max))
		if d.PrecipChance != nil {
			line += fmt.Sprintf("   %s %d%%", r.Icons.Umbrella, int(math.Round(*d.PrecipChance)))
		}
		b.WriteString("\n" + line + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// upcomingHours keeps the steps from the current one up to a day ahead.
func upcomingHours(hours []weather.Hour, now time.Time) []weather.Hour {
	step := time.Hour
	if len(hours) > 1 {
		step = hours[1].Time.Sub(hours[0].Time)
	}
	var out []weather.Hour
	for _, h := range hours {
		if !h.Time.Add(step).After(now) {
			continue
		}
		if len(out) > 0 && !h.Time.Before(out[0].Time.Add(24*time.Hour)) {
			break
		}
		out = append(out, h)
	}
	return out
}

// minBarSpan is the smallest temperature range, in °C, that the hourly bars
// stretch over.
const minBarSpan = 12.0

// Hourly renders the next 24 hours.
func (r Renderer) Hourly(f *weather.Forecast) string {
	tz := f.Location.TZ()
	now := r.now(tz)
	hours := upcomingHours(f.Hours, now)
	if len(hours) == 0 {
		return dimStyle.Render(r.Text.NoHours)
	}

	lo, hi := hours[0].Temp, hours[0].Temp
	for _, h := range hours {
		lo, hi = min(lo, h.Temp), max(hi, h.Temp)
	}
	// Keep the bars from blowing a two-degree swing up to the full width.
	if span := hi - lo; span < minBarSpan {
		lo -= (minBarSpan - span) / 2
		hi = lo + minBarSpan
	}
	condWidth := 0
	for _, h := range hours {
		condWidth = max(condWidth, lipgloss.Width(r.Text.Conditions[h.Condition]))
	}

	var b strings.Builder
	var lastDay time.Time
	for i, h := range hours {
		t := h.Time.In(tz)
		if i == 0 || !sameDay(t, lastDay) {
			if i > 0 {
				b.WriteString("\n")
			}
			b.WriteString(r.dayHeading(t, now) + "\n")
			lastDay = t
		}
		bar := lipgloss.NewStyle().Foreground(tempColor(h.Temp)).Render(strings.Repeat("▬", scale(h.Temp, lo, hi, 1, 10)))
		row := fmt.Sprintf("  %s  %s  %s %s  %s",
			t.Format("15:04"), r.icon(h.Condition, h.IsDay),
			padLeft(r.coloredTemp(h.Temp), 4), pad(bar, 10),
			pad(r.Text.Conditions[h.Condition], condWidth))
		if h.PrecipChance != nil && *h.PrecipChance > 0 {
			row += fmt.Sprintf("  %s %d%%", r.Icons.Umbrella, int(math.Round(*h.PrecipChance)))
		}
		b.WriteString(strings.TrimRight(row, " ") + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func (r Renderer) dayHeading(t, now time.Time) string {
	var name string
	switch {
	case sameDay(t, now):
		name = r.Text.Today
	case sameDay(t, now.AddDate(0, 0, 1)):
		name = r.Text.Tomorrow
	default:
		name = r.Text.Weekdays[t.Weekday()]
	}
	return titleStyle.Render(fmt.Sprintf("%s, %02d.%02d", name, t.Day(), t.Month()))
}

// scale maps v from [lo, hi] to an integer in [from, to].
func scale(v, lo, hi float64, from, to int) int {
	if hi <= lo {
		return (from + to) / 2
	}
	return from + int(math.Round((v-lo)/(hi-lo)*float64(to-from)))
}

// Daily renders the days from today on, with a bar that places each day's
// range on the scale of the whole week.
func (r Renderer) Daily(f *weather.Forecast) string {
	tz := f.Location.TZ()
	now := r.now(tz)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, tz)
	var days []weather.Day
	for _, d := range f.Days {
		if !d.Date.In(tz).Before(today) {
			days = append(days, d)
		}
	}
	if len(days) == 0 {
		return dimStyle.Render(r.Text.NoDays)
	}

	lo, hi := days[0].Min, days[0].Max
	for _, d := range days {
		lo, hi = min(lo, d.Min), max(hi, d.Max)
	}
	const barWidth = 16

	var b strings.Builder
	for _, d := range days {
		date := d.Date.In(tz)
		from, to := scale(d.Min, lo, hi, 0, barWidth-1), scale(d.Max, lo, hi, 0, barWidth-1)
		var bar strings.Builder
		for i := range barWidth {
			if i < from || i > to {
				bar.WriteString(dimStyle.Render("─"))
				continue
			}
			t := lo + (hi-lo)*float64(i)/float64(barWidth-1)
			bar.WriteString(lipgloss.NewStyle().Foreground(tempColor(t)).Render("━"))
		}
		row := fmt.Sprintf("%s  %s  %s %s %s  %s",
			pad(r.dayLabel(date, now), 8), r.icon(d.Condition, true),
			padLeft(r.coloredTemp(d.Min), 4), bar.String(), pad(r.coloredTemp(d.Max), 4),
			r.Text.Conditions[d.Condition])
		if d.PrecipChance != nil && *d.PrecipChance > 0 {
			row += fmt.Sprintf("  %s %d%%", r.Icons.Umbrella, int(math.Round(*d.PrecipChance)))
		}
		b.WriteString(row + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// Line is a one-line summary for shell prompts and status bars.
func (r Renderer) Line(f *weather.Forecast) string {
	city, _, _ := strings.Cut(f.Location.Name, ",")
	c := f.Current
	if c == nil {
		return city + " " + r.Icons.Condition(weather.Unknown, true)
	}
	return fmt.Sprintf("%s %s %s %s", city, r.icon(c.Condition, c.IsDay), r.coloredTemp(c.Temp), r.Text.Conditions[c.Condition])
}
