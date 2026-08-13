// Package screens contains the TUI screens.
package screens

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"github.com/agmonetti/relm/internal/conn"
	"github.com/agmonetti/relm/internal/tui/styles"
)

// ConnectMsg is emitted when the user asks to connect.
type ConnectMsg struct {
	Cfg conn.ConnectionConfig
}

// SaveConnectionMsg is emitted when the user saves the current connection.
type SaveConnectionMsg struct {
	Cfg conn.ConnectionConfig
}

// DeleteConnectionMsg is emitted when the user deletes a saved connection.
type DeleteConnectionMsg struct {
	Name string
}

// logoASCII is the connect screen logo (figlet "relm", blocks font).
const logoASCII = `  _____  ______  _      __  __ 
 |  __ \|  ____|| |    |  \/  |
 | |__) | |__   | |    | \  / |
 |  _  /|  __|  | |    | |\/| |
 | | \ \| |____ | |____| |  | |
 |_|  \_\______|_\_____|_|  |_|`
// field is a form label + input. If isToggle is true, the field is a boolean
// (checkbox) and input is not used.
type field struct {
	label    string
	input    textinput.Model
	isToggle bool
	checked  bool
}

// ConnScreen is the state of the connection form. The engine is selected with
// ←/→ when the focus is on the selector; Tab moves across the fields.
type ConnScreen struct {
	driverIdx int
	saved     []conn.SavedConnection
	savedIdx  int
	focus     int // 0 = engine, 1..len(fields) = fields, len(fields)+1 = saved
	fields    []field
	err       string

	width  int
	height int
}

// NewConnScreen creates the screen with the saved connections loaded.
func NewConnScreen(saved []conn.SavedConnection) *ConnScreen {
	c := &ConnScreen{saved: saved}
	c.rebuildFields()
	c.applyFocus()
	return c
}

// fieldsVisible returns the fields for the selected engine.
func (c *ConnScreen) fieldsVisible() []*field {
	drv := c.driver()
	var out []*field
	for i := range c.fields {
		f := &c.fields[i]
		switch f.label {
		case "File":
			if drv == conn.DriverSQLite {
				out = append(out, f)
			}
		case "Read-only":
			if drv == conn.DriverSQLite {
				out = append(out, f)
			}
		case "SSL":
			if drv == conn.DriverPostgres {
				out = append(out, f)
			}
		default: // Host, Port, User, Password, Database
			if drv != conn.DriverSQLite {
				out = append(out, f)
			}
		}
	}
	return out
}

// rebuildFields creates the persistent inputs (one per possible field). Values
// already typed are preserved so switching engines does not lose them.
func (c *ConnScreen) rebuildFields() {
	prev := map[string]field{}
	for _, f := range c.fields {
		prev[f.label] = f
	}
	mk := func(label, placeholder string) field {
		in := textinput.New()
		in.Cursor.BlinkSpeed = CursorBlink
		in.Placeholder = placeholder
		in.Prompt = " "
		in.Width = 24
		return field{label: label, input: in}
	}
	c.fields = []field{
		mk("File", "/data/app.db"),
		mk("Host", "localhost"),
		mk("Port", strconv.Itoa(conn.DefaultPort(c.driver()))),
		mk("User", "postgres"),
		mk("Password", ""),
		mk("Database", "mydb"),
		{label: "Read-only", isToggle: true},
		mk("SSL", "prefer"),
	}
	for i := range c.fields {
		if p, ok := prev[c.fields[i].label]; ok {
			c.fields[i].input.SetValue(p.input.Value())
			c.fields[i].checked = p.checked
		}
	}
	// the password is masked only for network engines
	if c.driver() != conn.DriverSQLite {
		c.field("Password").input.EchoMode = textinput.EchoPassword
	}
	c.field("Port").input.Placeholder = strconv.Itoa(conn.DefaultPort(c.driver()))
}

// field returns the form field with the given label, or nil if it does not exist.
func (c *ConnScreen) field(label string) *field {
	for i := range c.fields {
		if c.fields[i].label == label {
			return &c.fields[i]
		}
	}
	return nil
}

// driver returns the selected engine.
func (c *ConnScreen) driver() conn.Driver { return conn.Drivers[c.driverIdx] }

// savedFocus is the focus index of the saved list.
func (c *ConnScreen) savedFocus() int { return len(c.fieldsVisible()) + 1 }

// applyFocus focuses the active input and blurs the rest.
func (c *ConnScreen) applyFocus() tea.Cmd {
	var cmds []tea.Cmd
	for i := range c.fields {
		c.fields[i].input.Blur()
	}
	vis := c.fieldsVisible()
	if idx := c.focus - 1; idx >= 0 && idx < len(vis) && !vis[idx].isToggle {
		cmds = append(cmds, vis[idx].input.Focus())
	}
	return tea.Batch(cmds...)
}

func (c *ConnScreen) nextFocus() {
	total := c.savedFocus() + 1 // +1 to return to the engine
	c.focus = (c.focus + 1) % total
	c.applyFocus()
}

func (c *ConnScreen) cycleDriver(right bool) {
	prevDefault := conn.DefaultPort(c.driver())
	n := len(conn.Drivers)
	if right {
		c.driverIdx = (c.driverIdx + 1) % n
	} else {
		c.driverIdx = (c.driverIdx - 1 + n) % n
	}
	c.rebuildFields()
	// Reset the port when it still holds the previous engine's default (or is
	// empty), so switching engines lands on the new engine's default port
	// instead of a stale one. A custom port the user typed is preserved.
	if p := c.field("Port"); p != nil {
		if v := p.input.Value(); v == "" || v == strconv.Itoa(prevDefault) {
			p.input.SetValue("")
		}
	}
	c.focus = 0
	c.applyFocus()
}

func (c *ConnScreen) moveSaved(up bool) {
	n := len(c.saved)
	if n == 0 {
		return
	}
	if up {
		c.savedIdx = (c.savedIdx - 1 + n) % n
	} else {
		c.savedIdx = (c.savedIdx + 1) % n
	}
}

// cfg builds the current config of the form.
func (c *ConnScreen) cfg() conn.ConnectionConfig {
	val := func(label string) string {
		for i := range c.fields {
			if c.fields[i].label == label {
				return c.fields[i].input.Value()
			}
		}
		return ""
	}
	checked := func(label string) bool {
		for i := range c.fields {
			if c.fields[i].label == label && c.fields[i].isToggle {
				return c.fields[i].checked
			}
		}
		return false
	}
	cfg := conn.New(c.driver())
	cfg.Path = val("File")
	cfg.Host = val("Host")
	cfg.Port = parsePort(val("Port"), conn.DefaultPort(c.driver()))
	cfg.User = val("User")
	cfg.Password = val("Password")
	cfg.Database = val("Database")
	cfg.ReadOnly = checked("Read-only")
	cfg.SSLMode = val("SSL")
	return cfg
}

func parsePort(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 || n > 65535 {
		return def
	}
	return n
}

// validate checks the required fields for the engine.
func (c *ConnScreen) validate() error {
	cfg := c.cfg()
	if cfg.Driver == conn.DriverSQLite {
		if cfg.Path == "" {
			return fmt.Errorf("type the file path")
		}
	} else if cfg.Host == "" {
		return fmt.Errorf("type the host")
	}
	if cfg.Driver != conn.DriverSQLite {
		if p := c.field("Port").input.Value(); p != "" {
			if n, err := strconv.Atoi(p); err != nil || n <= 0 || n > 65535 {
				return fmt.Errorf("port must be a number between 1 and 65535")
			}
		}
	}
	if cfg.SSLMode != "" {
		switch cfg.SSLMode {
		case "prefer", "require", "verify-ca", "verify-full", "disable":
		default:
			return fmt.Errorf("ssl: use prefer, require, verify-ca, verify-full or disable")
		}
	}
	return nil
}

// connectCmd builds the connection message.
func (c *ConnScreen) connectCmd() tea.Cmd {
	cfg := c.cfg()
	return func() tea.Msg { return ConnectMsg{Cfg: cfg} }
}

func (c *ConnScreen) connect() (tea.Cmd, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}
	return c.connectCmd(), nil
}

// Update handles keys and resizing.
func (c *ConnScreen) Update(msg tea.Msg) (*ConnScreen, tea.Cmd) {
	var cmds []tea.Cmd
	handled := false

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.width, c.height = msg.Width, msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			c.nextFocus()
			handled = true
		case "enter":
			if idx := c.focus - 1; idx >= 0 {
				if vis := c.fieldsVisible(); idx < len(vis) && vis[idx].isToggle {
					vis[idx].checked = !vis[idx].checked
				} else if c.focus == c.savedFocus() && len(c.saved) > 0 {
					cfg := c.saved[c.savedIdx].ToConfig()
					cmds = append(cmds, func() tea.Msg { return ConnectMsg{Cfg: cfg} })
				} else {
					cmd, err := c.connect()
					if err != nil {
						c.err = err.Error()
					} else if cmd != nil {
						cmds = append(cmds, cmd)
					}
				}
			}
			handled = true
		case "left", "right":
			if c.focus == 0 {
				c.cycleDriver(msg.String() == "right")
				handled = true
			}
		case "up", "down":
			if c.focus == c.savedFocus() {
				c.moveSaved(msg.String() == "up")
				handled = true
			}
		case "ctrl+s":
			if err := c.validate(); err != nil {
				c.err = err.Error()
			} else {
				cfg := c.cfg()
				cmds = append(cmds, func() tea.Msg { return SaveConnectionMsg{Cfg: cfg} })
			}
			handled = true
		case "d":
			if c.focus == c.savedFocus() && len(c.saved) > 0 {
				cmds = append(cmds, func() tea.Msg {
					return DeleteConnectionMsg{Name: c.saved[c.savedIdx].Name}
				})
				handled = true
			}
		case "r":
			if c.focus == 0 {
				c.reset()
				handled = true
			}
		}

		if !handled {
			vis := c.fieldsVisible()
			if idx := c.focus - 1; idx >= 0 && idx < len(vis) {
				f := vis[idx]
				if f.isToggle {
					if msg.String() == " " {
						f.checked = !f.checked
					}
				} else {
					updated, cmd := f.input.Update(msg)
					f.input = updated
					cmds = append(cmds, cmd)
				}
			}
		}
	}

	return c, tea.Batch(cmds...)
}

// reset clears the form and the error.
func (c *ConnScreen) reset() {
	for i := range c.fields {
		c.fields[i].input.SetValue("")
		c.fields[i].checked = false
	}
	c.field("Port").input.Placeholder = strconv.Itoa(conn.DefaultPort(c.driver()))
	c.err = ""
}

// ResetForm clears the form (for a new session).
func (c *ConnScreen) ResetForm() { c.reset() }

// SetSaved updates the list of saved connections.
func (c *ConnScreen) SetSaved(saved []conn.SavedConnection) {
	c.saved = saved
	if c.savedIdx >= len(c.saved) {
		c.savedIdx = 0
	}
}

// SetError shows a connection error on the screen.
func (c *ConnScreen) SetError(err string) { c.err = err }

// Error returns the current screen error.
func (c *ConnScreen) Error() string { return c.err }

// FieldValue returns the value of a form field by label.
func (c *ConnScreen) FieldValue(label string) string {
	for i := range c.fields {
		if c.fields[i].label == label {
			return c.fields[i].input.Value()
		}
	}
	return ""
}

// FocusOnField reports whether the focus is on a text input (not the engine or
// the saved list).
func (c *ConnScreen) FocusOnField() bool {
	return c.focus >= 1 && c.focus < c.savedFocus()
}

// View renders the lazyvim-style centered menu: logo on top and below the form
// and the saved connections, all centered in the available area.
func (c *ConnScreen) View(width, height int) string {
	if width > 0 {
		c.width = width
	}
	if height > 0 {
		c.height = height
	}

	var b strings.Builder
	b.WriteString(renderLogo(c.width))
	b.WriteString("\n\n")
	b.WriteString(c.renderForm())
	content := b.String()

	// Saved connections are only shown if there is room next to the form;
	// otherwise they are omitted so they don't hide it.
	if len(c.saved) > 0 {
		saved := c.renderSaved()
		if lipgloss.Height(content)+2+lipgloss.Height(saved) <= c.height {
			content += "\n\n" + saved
		}
	}

	if c.err != "" {
		content += "\n\n" + styles.StyleError.Render(c.err)
	}

	// Center every line horizontally so logo and form share the center axis,
	// then center vertically (a no-op horizontally once all lines are the
	// terminal width).
	content = lipgloss.NewStyle().Width(c.width).Align(lipgloss.Center).Render(content)

	if lipgloss.Height(content) > c.height {
		content = lipgloss.Place(c.width, c.height, lipgloss.Center, lipgloss.Top, content)
	} else {
		content = lipgloss.Place(c.width, c.height, lipgloss.Center, lipgloss.Center, content)
	}

	// bubbletea expects exactly c.height lines: trim if needed.
	lines := strings.Split(content, "\n")
	if len(lines) > c.height {
		content = strings.Join(lines[:c.height], "\n")
	}
	return content
}

// renderLogo renders the ASCII logo centered on the given width. The lines are
// first padded to a uniform width so lipgloss.Center does not skew them.
func renderLogo(width int) string {
	lines := strings.Split(logoASCII, "\n")
	maxW := 0
	for _, l := range lines {
		if w := runewidth.StringWidth(l); w > maxW {
			maxW = w
		}
	}
	for i, l := range lines {
		lines[i] = l + strings.Repeat(" ", maxW-runewidth.StringWidth(l))
	}
	return styles.StyleLogo.Width(width).Align(lipgloss.Center).Render(strings.Join(lines, "\n"))
}

// Fixed form widths: label on the left + input box.
const (
	fieldLabelW = 14
	fieldBoxW   = 32
)

func (c *ConnScreen) renderForm() string {
	var b strings.Builder
	b.WriteString(styles.StyleHeader.Render("Connect"))
	b.WriteString("\n\n")

	// engine selector
	b.WriteString(fieldRow("Engine", c.renderMotorSelector(), c.width))
	b.WriteString("\n")

	// fields, stacked without blank lines so they don't overflow on small terminals
	for i := range c.fieldsVisible() {
		f := c.fieldsVisible()[i]
		style := styles.StyleInputBox
		if c.focus == i+1 {
			style = styles.StyleInputBoxFocus
		}
		var box string
		if f.isToggle {
			t := " [ ] "
			if f.checked {
				t = " [x] "
			}
			box = style.Width(fieldBoxW).Render(t + styles.StyleHeaderDim.Render("open read-only"))
		} else {
			box = style.Width(fieldBoxW).Render(f.input.View())
		}
		b.WriteString(fieldRow(f.label, box, c.width))
		b.WriteString("\n")
	}

	// buttons
	b.WriteString("\n")
	b.WriteString(styles.StyleBtnPrimary.Render("Enter · Connect"))
	b.WriteString("\n")
	b.WriteString(strings.Join([]string{
		styles.StyleBtnSecondary.Render("ctrl+s  save"),
		styles.StyleBtnSecondary.Render("r  clear"),
	}, "  "))
	b.WriteString("\n")
	return b.String()
}

// fieldRow centers the input box on the terminal axis and places the label to
// its left. Centering the whole label+box group would visibly shift the box
// right by half the label width.
func fieldRow(label, box string, width int) string {
	lbl := styles.StyleFieldLabel.Width(fieldLabelW).Align(lipgloss.Right).Render(label)
	boxW := lipgloss.Width(box)
	boxLeft := (width - boxW) / 2
	if boxLeft < 0 {
		boxLeft = 0
	}
	prefix := boxLeft - fieldLabelW - 1
	if prefix < 0 {
		prefix = 0
	}
	right := width - boxLeft - boxW
	if right < 0 {
		right = 0
	}

	lines := strings.Split(box, "\n")
	middle := len(lines) / 2
	for i, line := range lines {
		if i == middle {
			lines[i] = strings.Repeat(" ", prefix) + lbl + " " + line + strings.Repeat(" ", right)
		} else {
			lines[i] = strings.Repeat(" ", boxLeft) + line + strings.Repeat(" ", right)
		}
	}
	return strings.Join(lines, "\n")
}

// renderMotorSelector draws the engine selector as a focusable box.
func (c *ConnScreen) renderMotorSelector() string {
	style := styles.StyleInputBox
	if c.focus == 0 {
		style = styles.StyleInputBoxFocus
	}
	content := lipgloss.JoinHorizontal(lipgloss.Bottom,
		styles.StyleHeader.Render(" "+string(c.driver())),
		strings.Repeat(" ", 4),
		styles.StyleHeaderDim.Render("←→ switch"),
	)
	return style.Width(fieldBoxW).Render(content)
}

func (c *ConnScreen) renderSaved() string {
	var b strings.Builder
	b.WriteString(styles.StyleHeader.Render("Saved"))
	b.WriteString("\n\n")
	for i, s := range c.saved {
		line := fmt.Sprintf("%s  %s", s.Name, s.ToConfig().Label())
		if c.focus == c.savedFocus() && i == c.savedIdx {
			b.WriteString(styles.StyleSidebarActive.Render("> " + line))
		} else {
			b.WriteString(styles.StyleSidebarItem.Render(line))
		}
		b.WriteString("\n")
	}
	return b.String()
}
