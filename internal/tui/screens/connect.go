// Package screens contiene las pantallas de la TUI.
package screens

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"relm/internal/conn"
	"relm/internal/tui/styles"
)

// ConnectMsg se emite cuando el usuario pide conectar.
type ConnectMsg struct {
	Cfg conn.ConnectionConfig
}

// SaveConnectionMsg se emite cuando el usuario guarda la conexión actual.
type SaveConnectionMsg struct {
	Cfg conn.ConnectionConfig
}

// field es una etiqueta + input del formulario.
type field struct {
	label string
	input textinput.Model
}

// ConnScreen es el estado del formulario de conexión. El motor se selecciona
// con ←/→ cuando el foco está en el selector; Tab recorre los campos.
type ConnScreen struct {
	driverIdx int
	saved     []conn.SavedConnection
	savedIdx  int
	focus     int // 0 = motor, 1..len(fields) = campos, len(fields)+1 = guardadas
	fields    []field
	err       string

	width  int
	height int
}

// NewConnScreen crea la pantalla con las conexiones guardadas cargadas.
func NewConnScreen(saved []conn.SavedConnection) *ConnScreen {
	c := &ConnScreen{saved: saved}
	c.rebuildFields()
	c.applyFocus()
	return c
}

// fieldsVisible devuelve los campos según el motor seleccionado.
func (c *ConnScreen) fieldsVisible() []*field {
	drv := c.driver()
	var out []*field
	for i := range c.fields {
		f := &c.fields[i]
		switch f.label {
		case "Archivo":
			if drv == conn.DriverSQLite {
				out = append(out, f)
			}
		default: // Host, Puerto, Usuario, Password, Base
			if drv != conn.DriverSQLite {
				out = append(out, f)
			}
		}
	}
	return out
}

// rebuildFields crea los inputs persistentes (uno por campo posible).
func (c *ConnScreen) rebuildFields() {
	mk := func(label, placeholder string) field {
		in := textinput.New()
		in.Placeholder = placeholder
		in.Prompt = " "
		in.Width = 24
		return field{label: label, input: in}
	}
	c.fields = []field{
		mk("Archivo", "/data/app.db"),
		mk("Host", "localhost"),
		mk("Puerto", strconv.Itoa(conn.DefaultPort(c.driver()))),
		mk("Usuario", "postgres"),
		mk("Password", ""),
		mk("Base", "mydb"),
	}
	// la password se enmascara solo en motores de red
	if c.driver() != conn.DriverSQLite {
		c.fields[4].input.EchoMode = textinput.EchoPassword
	}
	c.fields[2].input.Placeholder = strconv.Itoa(conn.DefaultPort(c.driver()))
}

// driver devuelve el motor seleccionado.
func (c *ConnScreen) driver() conn.Driver { return conn.Drivers[c.driverIdx] }

// savedFocus es el índice de foco de la lista de guardadas.
func (c *ConnScreen) savedFocus() int { return len(c.fieldsVisible()) + 1 }

// applyFocus enfoca el input activo y desenfoca el resto.
func (c *ConnScreen) applyFocus() tea.Cmd {
	var cmds []tea.Cmd
	for i := range c.fields {
		c.fields[i].input.Blur()
	}
	vis := c.fieldsVisible()
	if idx := c.focus - 1; idx >= 0 && idx < len(vis) {
		cmds = append(cmds, vis[idx].input.Focus())
	}
	return tea.Batch(cmds...)
}

func (c *ConnScreen) nextFocus() {
	total := c.savedFocus() + 1 // +1 para volver al motor
	c.focus = (c.focus + 1) % total
	c.applyFocus()
}

func (c *ConnScreen) cycleDriver(right bool) {
	n := len(conn.Drivers)
	if right {
		c.driverIdx = (c.driverIdx + 1) % n
	} else {
		c.driverIdx = (c.driverIdx - 1 + n) % n
	}
	c.rebuildFields()
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

// cfg construye la config actual del formulario.
func (c *ConnScreen) cfg() conn.ConnectionConfig {
	val := func(label string) string {
		for i := range c.fields {
			if c.fields[i].label == label {
				return c.fields[i].input.Value()
			}
		}
		return ""
	}
	cfg := conn.New(c.driver())
	cfg.Path = val("Archivo")
	cfg.Host = val("Host")
	cfg.Port = parsePort(val("Puerto"), conn.DefaultPort(c.driver()))
	cfg.User = val("Usuario")
	cfg.Password = val("Password")
	cfg.Database = val("Base")
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

// validate verifica los campos obligatorios según el motor.
func (c *ConnScreen) validate() error {
	cfg := c.cfg()
	if cfg.Driver == conn.DriverSQLite {
		if cfg.Path == "" {
			return fmt.Errorf("escribe el path del archivo")
		}
	} else if cfg.Host == "" {
		return fmt.Errorf("escribe el host")
	}
	return nil
}

// connectCmd arma el mensaje de conexión.
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

// Update maneja teclas y redimensiona.
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
			if c.focus == c.savedFocus() && len(c.saved) > 0 {
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
			handled = true
		case "left", "right":
			if c.focus == 0 {
				c.cycleDriver(msg.String() == "right")
			}
			handled = true
		case "up", "down":
			if c.focus == c.savedFocus() {
				c.moveSaved(msg.String() == "up")
				handled = true
			}
		case "ctrl+s":
			cfg := c.cfg()
			cmds = append(cmds, func() tea.Msg { return SaveConnectionMsg{Cfg: cfg} })
			handled = true
		case "r":
			if c.focus == 0 {
				c.reset()
				handled = true
			}
		}

		if !handled {
			vis := c.fieldsVisible()
			if idx := c.focus - 1; idx >= 0 && idx < len(vis) {
				updated, cmd := vis[idx].input.Update(msg)
				vis[idx].input = updated
				cmds = append(cmds, cmd)
			}
		}
	}

	return c, tea.Batch(cmds...)
}

// reset limpia el formulario y el error.
func (c *ConnScreen) reset() {
	for i := range c.fields {
		c.fields[i].input.SetValue("")
	}
	c.fields[2].input.Placeholder = strconv.Itoa(conn.DefaultPort(c.driver()))
	c.err = ""
}

// ResetForm limpia el formulario (para nueva sesión).
func (c *ConnScreen) ResetForm() { c.reset() }

// SetSaved actualiza la lista de conexiones guardadas.
func (c *ConnScreen) SetSaved(saved []conn.SavedConnection) {
	c.saved = saved
	if c.savedIdx >= len(c.saved) {
		c.savedIdx = 0
	}
}

// SetError muestra un error de conexión en la pantalla.
func (c *ConnScreen) SetError(err string) { c.err = err }

// Error devuelve el error actual de la pantalla.
func (c *ConnScreen) Error() string { return c.err }

// FieldValue devuelve el valor de un campo del formulario por etiqueta.
func (c *ConnScreen) FieldValue(label string) string {
	for i := range c.fields {
		if c.fields[i].label == label {
			return c.fields[i].input.Value()
		}
	}
	return ""
}

// FocusOnField indica si el foco está en un input de texto (no en el motor ni
// en la lista de guardadas).
func (c *ConnScreen) FocusOnField() bool {
	return c.focus >= 1 && c.focus < c.savedFocus()
}

// View renderiza el formulario y la lista de guardadas.
func (c *ConnScreen) View(width, height int) string {
	if width > 0 {
		c.width = width
	}
	if height > 0 {
		c.height = height
	}

	form := c.renderForm()
	saved := c.renderSaved()
	content := styles.StyleBordered.Width(width - 2).Height(c.height - 2).
		Render(lipglossJoinHorizontal(form, saved))

	errLine := ""
	if c.err != "" {
		errLine = "\n" + styles.StyleError.Render(c.err)
	}
	return content + errLine
}

func lipglossJoinHorizontal(a, b string) string {
	return a + "\n\n" + b
}

func (c *ConnScreen) renderForm() string {
	var b strings.Builder
	b.WriteString(styles.StyleHeader.Render("Conectar"))
	b.WriteString("\n\n")

	// selector de motor
	sel := fmt.Sprintf("%s ", styles.StyleHeaderDim.Render("Motor "))
	if c.focus == 0 {
		sel += styles.StyleCursor.Render(string(c.driver()))
	} else {
		sel += string(c.driver())
	}
	sel += styles.StyleHeaderDim.Render("  ←→ cambiar")
	b.WriteString(sel + "\n")

	// campos
	for i := range c.fieldsVisible() {
		f := c.fieldsVisible()[i]
		label := styles.StyleHeaderDim.Render(f.label + " ")
		value := f.input.View()
		if c.focus == i+1 {
			value = styles.StyleAccentInput.Render(value)
		}
		b.WriteString(label + value + "\n")
	}

	b.WriteString("\n")
	btn := "  Conectar (enter)  "
	b.WriteString(styles.StyleBordered.Render(btn) + "\n")
	b.WriteString(styles.StyleHeaderDim.Render("  ctrl+s guardar · r limpiar") + "\n")
	return b.String()
}

func (c *ConnScreen) renderSaved() string {
	var b strings.Builder
	b.WriteString(styles.StyleHeader.Render("Guardadas"))
	b.WriteString("\n\n")
	if len(c.saved) == 0 {
		b.WriteString(styles.StyleHeaderDim.Render("  sin conexiones guardadas"))
		return b.String()
	}
	for i, s := range c.saved {
		line := fmt.Sprintf("  %s  %s", s.Name, s.ToConfig().Label())
		if c.focus == c.savedFocus() && i == c.savedIdx {
			b.WriteString(styles.StyleSidebarActive.Render("> " + line))
		} else {
			b.WriteString(styles.StyleSidebarItem.Render(line))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n" + styles.StyleHeaderDim.Render("  enter conectar") + "\n")
	return b.String()
}
