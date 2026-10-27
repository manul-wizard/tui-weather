package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/manul-wizard/tui-weather/internal/config"
	"github.com/manul-wizard/tui-weather/internal/weather"
)

// App is what the interactive mode needs to start.
type App struct {
	Config     config.Config
	ConfigPath string
	// Location overrides the one in Config without saving it, as the --city
	// flag does.
	Location *weather.Location
	Options  weather.Options
	// Search, when set, opens the city search with this query.
	Search string
}

// Run starts the interactive mode and blocks until the user quits.
func Run(a App) error {
	m := newModel(a)
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}

type screen int

const (
	menuScreen screen = iota
	viewScreen
	searchScreen
	resultsScreen
)

type view int

const (
	nowView view = iota
	hourlyView
	dailyView
	viewCount
)

type menuItem int

const (
	itemNow menuItem = iota
	itemHourly
	itemDaily
	itemCity
	itemRefresh
	itemQuit
	itemCount
)

type forecastMsg struct {
	forecast *weather.Forecast
	err      error
}

type placesMsg struct {
	query  string
	places []weather.Location
	err    error
}

type model struct {
	app      App
	text     Text
	renderer Renderer

	screen  screen
	view    view
	cursor  int
	width   int
	height  int
	status  string
	loading bool

	forecast *weather.Forecast
	err      error

	spinner  spinner.Model
	viewport viewport.Model
	input    textinput.Model

	searching bool
	places    []weather.Location
	place     int
	searchErr string
}

func newModel(a App) model {
	text := TextFor(a.Config.Language)
	sp := spinner.New(spinner.WithSpinner(spinner.Dot))
	in := textinput.New()
	in.Prompt = text.CityPrompt + ": "
	in.Placeholder = text.CityPlaceholder
	in.CharLimit = 80

	m := model{
		app:  a,
		text: text,
		renderer: Renderer{
			Text: text, Icons: IconsFor(a.Config.Icons), Imperial: a.Config.Units == "imperial",
		},
		spinner:  sp,
		viewport: viewport.New(80, 20),
		input:    in,
		loading:  true,
	}
	if a.Search != "" {
		m.screen = searchScreen
		m.input.SetValue(a.Search)
		m.input.Focus()
		m.searching = true
	}
	return m
}

func (m model) location() weather.Location {
	if m.app.Location != nil {
		return *m.app.Location
	}
	return m.app.Config.Location
}

func (m model) fetch(refresh bool) tea.Cmd {
	o := m.app.Options
	o.Location = m.location()
	o.Refresh = refresh
	return func() tea.Msg {
		f, err := weather.Get(context.Background(), o)
		return forecastMsg{f, err}
	}
}

func (m model) search(query string) tea.Cmd {
	lang := m.app.Config.Language
	timeout := m.app.Options.Timeout
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		places, err := weather.Search(ctx, weather.NewClient(timeout), query, lang)
		return placesMsg{query, places, err}
	}
}

func (m model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.spinner.Tick, m.fetch(false)}
	if m.searching {
		cmds = append(cmds, m.search(m.input.Value()), textinput.Blink)
	}
	return tea.Batch(cmds...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.resizeViewport()
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case forecastMsg:
		m.loading = false
		m.forecast, m.err = msg.forecast, msg.err
		m.refreshViewport()
		return m, nil

	case placesMsg:
		if msg.query != m.input.Value() {
			return m, nil // an answer to an older query
		}
		m.searching = false
		m.searchErr = ""
		switch {
		case msg.err != nil:
			m.searchErr = m.text.SearchFailed + ": " + msg.err.Error()
		case len(msg.places) == 0:
			m.searchErr = fmt.Sprintf(m.text.NotFound, msg.query)
		default:
			m.places, m.place = msg.places, 0
			m.screen = resultsScreen
			m.input.Blur()
		}
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		switch m.screen {
		case menuScreen:
			return m.updateMenu(msg)
		case viewScreen:
			return m.updateView(msg)
		case searchScreen:
			return m.updateSearch(msg)
		case resultsScreen:
			return m.updateResults(msg)
		}
	}
	return m, nil
}

func (m model) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		return m, tea.Quit
	case "up", "k":
		m.cursor = (m.cursor + int(itemCount) - 1) % int(itemCount)
	case "down", "j":
		m.cursor = (m.cursor + 1) % int(itemCount)
	case "1", "2", "3", "4", "5", "6":
		m.cursor = int(msg.String()[0] - '1')
		return m.choose()
	case "enter", " ":
		return m.choose()
	}
	return m, nil
}

func (m model) choose() (tea.Model, tea.Cmd) {
	switch menuItem(m.cursor) {
	case itemNow, itemHourly, itemDaily:
		m.screen, m.view = viewScreen, view(m.cursor)
		m.refreshViewport()
	case itemCity:
		m.screen = searchScreen
		m.input.SetValue("")
		m.searchErr = ""
		return m, tea.Batch(m.input.Focus(), textinput.Blink)
	case itemRefresh:
		m.loading, m.status = true, ""
		return m, tea.Batch(m.spinner.Tick, m.fetch(true))
	case itemQuit:
		return m, tea.Quit
	}
	return m, nil
}

func (m model) updateView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "esc", "backspace":
		m.screen = menuScreen
		return m, nil
	case "left", "h", "shift+tab":
		m.view = (m.view + viewCount - 1) % viewCount
		m.refreshViewport()
		return m, nil
	case "right", "l", "tab":
		m.view = (m.view + 1) % viewCount
		m.refreshViewport()
		return m, nil
	case "r":
		m.loading, m.status = true, ""
		return m, tea.Batch(m.spinner.Tick, m.fetch(true))
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = menuScreen
		m.input.Blur()
		return m, nil
	case "enter":
		query := strings.TrimSpace(m.input.Value())
		if query == "" {
			return m, nil
		}
		m.input.SetValue(query)
		m.searching, m.searchErr = true, ""
		return m, tea.Batch(m.spinner.Tick, m.search(query))
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m model) updateResults(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "backspace":
		m.screen = searchScreen
		return m, m.input.Focus()
	case "up", "k":
		m.place = (m.place + len(m.places) - 1) % len(m.places)
	case "down", "j":
		m.place = (m.place + 1) % len(m.places)
	case "enter":
		loc := m.places[m.place]
		m.app.Config.Location = loc
		m.app.Location = nil
		if err := config.Save(m.app.ConfigPath, m.app.Config); err != nil {
			m.status = err.Error()
		} else {
			m.status = fmt.Sprintf(m.text.Saved, loc.Name)
		}
		m.screen, m.view = viewScreen, nowView
		m.forecast, m.err, m.loading = nil, nil, true
		m.refreshViewport()
		return m, tea.Batch(m.spinner.Tick, m.fetch(false))
	}
	return m, nil
}

const (
	headerLines = 3 // title, place, blank
	footerLines = 2 // blank, hints
)

func (m *model) resizeViewport() {
	m.viewport.Width = max(m.width, 20)
	m.viewport.Height = max(m.height-headerLines-footerLines-1, 3)
	m.refreshViewport()
}

func (m *model) refreshViewport() {
	m.viewport.SetContent(Indent(m.content(), "  "))
	m.viewport.GotoTop()
}

func (m model) content() string {
	if m.forecast == nil {
		return ""
	}
	r := m.renderer
	var body string
	switch m.view {
	case nowView:
		body = r.Current(m.forecast)
	case hourlyView:
		body = r.Hourly(m.forecast)
	case dailyView:
		body = r.Daily(m.forecast)
	}
	return body + "\n\n" + r.Footer(m.forecast)
}

// Indent prefixes every non-empty line of s.
func Indent(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if l != "" {
			lines[i] = prefix + l
		}
	}
	return strings.Join(lines, "\n")
}

var (
	appTitle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ffd75f"))
	activeStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ffd75f"))
	tabStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5f5f"))
)

func (m model) View() string {
	var b strings.Builder
	b.WriteString(" " + appTitle.Render("tui-weather") + "  " + m.statusLine() + "\n")
	place := m.location().Name
	if m.forecast != nil {
		place = m.renderer.Header(m.forecast)
	} else {
		place = titleStyle.Render(place)
	}
	b.WriteString(Indent(place, " ") + "\n\n")

	var body, hints string
	switch m.screen {
	case menuScreen:
		body, hints = m.menuView(), m.text.HintMenu
	case viewScreen:
		body, hints = m.tabs()+"\n\n"+m.viewport.View(), m.text.HintView
	case searchScreen:
		body, hints = m.searchView(), m.text.HintSearch
	case resultsScreen:
		body, hints = m.resultsView(), m.text.HintResults
	}
	b.WriteString(body + "\n\n " + dimStyle.Render(hints))
	return b.String()
}

func (m model) statusLine() string {
	switch {
	case m.loading || m.searching:
		label := m.text.Loading
		if m.searching {
			label = m.text.Searching
		}
		if m.status != "" {
			label += " · " + m.status
		}
		return m.spinner.View() + dimStyle.Render(label)
	case m.err != nil:
		return errorStyle.Render(m.text.FetchFailed)
	case m.status != "":
		return dimStyle.Render(m.status)
	}
	return ""
}

func (m model) menuView() string {
	labels := []string{m.text.Now, m.text.Hourly, m.text.Daily, m.text.ChangeCity, m.text.Refresh, m.text.Quit}
	var b strings.Builder
	if m.forecast != nil && m.forecast.Current != nil {
		b.WriteString("  " + m.renderer.Line(m.forecast) + "\n\n")
	}
	if m.err != nil {
		b.WriteString(Indent(errorStyle.Render(m.err.Error()), "  ") + "\n\n")
	}
	for i, l := range labels {
		if i == m.cursor {
			b.WriteString(activeStyle.Render(fmt.Sprintf("  › %d  %s", i+1, l)) + "\n")
		} else {
			b.WriteString(fmt.Sprintf("    %s  %s", dimStyle.Render(fmt.Sprint(i+1)), l) + "\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m model) tabs() string {
	names := []string{m.text.Now, m.text.Hourly, m.text.Daily}
	parts := make([]string, len(names))
	for i, n := range names {
		if view(i) == m.view {
			parts[i] = activeStyle.Render("[" + n + "]")
		} else {
			parts[i] = tabStyle.Render(" " + n + " ")
		}
	}
	s := "  " + strings.Join(parts, " ")
	if m.err != nil && m.forecast == nil {
		s += "\n\n" + Indent(errorStyle.Render(m.err.Error()), "  ")
	}
	return s
}

func (m model) searchView() string {
	s := "  " + m.input.View()
	if m.searchErr != "" {
		s += "\n\n  " + errorStyle.Render(m.searchErr)
	}
	return s
}

func (m model) resultsView() string {
	var b strings.Builder
	for i, p := range m.places {
		coords := dimStyle.Render(fmt.Sprintf("  %.2f, %.2f · %s", p.Latitude, p.Longitude, p.Timezone))
		if i == m.place {
			b.WriteString(activeStyle.Render("  › "+p.Name) + coords + "\n")
		} else {
			b.WriteString("    " + p.Name + coords + "\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}
