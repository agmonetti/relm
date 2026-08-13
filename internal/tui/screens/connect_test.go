package screens

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"

	"github.com/agmonetti/relm/internal/conn"
)

func TestConnScreen_SQLiteOnlyShowsPath(t *testing.T) {
	c := NewConnScreen(nil)
	c.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	// sqlite by default: File + Read-only toggle
	if len(c.fieldsVisible()) != 2 {
		t.Fatalf("fieldsVisible = %d, want 2 (File + Read-only)", len(c.fieldsVisible()))
	}
	if c.fieldsVisible()[0].label != "File" {
		t.Errorf("field = %q, want File", c.fieldsVisible()[0].label)
	}
	if c.fieldsVisible()[1].label != "Read-only" || !c.fieldsVisible()[1].isToggle {
		t.Errorf("second field = %+v, want Read-only toggle", c.fieldsVisible()[1])
	}
}

func TestConnScreen_NetworkShowsAllFields(t *testing.T) {
	c := NewConnScreen(nil)
	c.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	c.cycleDriver(true) // sqlite -> postgres

	if len(c.fieldsVisible()) != 6 {
		t.Fatalf("fieldsVisible = %d, want 6 (Host..Database + SSL)", len(c.fieldsVisible()))
	}
	// password masked
	if c.fields[4].input.EchoMode != textinput.EchoPassword {
		t.Error("Password should be masked for network engines")
	}
}

func TestConnScreen_ToggleReadOnly(t *testing.T) {
	c := NewConnScreen(nil)
	c.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	c.fields[0].input.SetValue("/data/app.db")

	c.focus = 2 // Read-only toggle (field 2 of sqlite)
	c.Update(tea.KeyMsg{Type: tea.KeySpace})
	if !c.cfg().ReadOnly {
		t.Error("ReadOnly should be true after toggling")
	}
	c.Update(tea.KeyMsg{Type: tea.KeySpace})
	if c.cfg().ReadOnly {
		t.Error("ReadOnly should be false after toggling twice")
	}
}

func TestConnScreen_SSLModeInConfig(t *testing.T) {
	c := NewConnScreen(nil)
	c.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	c.cycleDriver(true) // postgres
	// SSL field is the last of postgres (index 5 in fieldsVisible)
	vis := c.fieldsVisible()
	if len(vis) != 6 || vis[5].label != "SSL" {
		t.Fatalf("expected SSL as sixth field, got %+v", vis)
	}
	vis[5].input.SetValue("require")
	if cfg := c.cfg(); cfg.SSLMode != "require" {
		t.Errorf("SSLMode = %q, want require", cfg.SSLMode)
	}
}

func TestConnScreen_ValidateSSLMode(t *testing.T) {
	c := NewConnScreen(nil)
	c.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	c.cycleDriver(true)                     // postgres
	c.fields[1].input.SetValue("localhost") // required host
	vis := c.fieldsVisible()
	vis[5].input.SetValue("bogus")
	if err := c.validate(); err == nil {
		t.Error("expected error for invalid sslmode")
	}
	vis[5].input.SetValue("verify-full")
	if err := c.validate(); err != nil {
		t.Errorf("validate = %v, want nil", err)
	}
}

func TestConnScreen_DriverCyclesAll(t *testing.T) {
	c := NewConnScreen(nil)
	seen := map[conn.Driver]bool{}
	for range conn.Drivers {
		seen[c.driver()] = true
		c.cycleDriver(true)
	}
	if len(seen) != 5 {
		t.Errorf("drivers seen = %d, want 5", len(seen))
	}
}

func TestConnScreen_FieldHelper(t *testing.T) {
	c := NewConnScreen(nil)
	c.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	if c.field("File") == nil || c.field("Read-only") == nil {
		t.Fatal("expected File and Read-only fields to exist")
	}
	if c.field("nonexistent") != nil {
		t.Fatal("expected nil for an unknown label")
	}

	// password is masked only for network engines
	if c.driver() != conn.DriverSQLite {
		if c.field("Password").input.EchoMode != textinput.EchoPassword {
			t.Error("Password should be masked for network engines")
		}
	}
}

func TestConnScreen_DeleteSavedConnection(t *testing.T) {
	c := NewConnScreen([]conn.SavedConnection{
		{Name: "local", Driver: conn.DriverSQLite, Path: "/data/app.db"},
	})
	c.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	// sqlite: engine(0) + File(1) + Read-only(2) -> saved list(3)
	for i := 0; i < 3; i++ {
		c.nextFocus()
	}
	if c.focus != c.savedFocus() {
		t.Fatalf("focus = %d, want saved %d", c.focus, c.savedFocus())
	}

	updated, cmd := c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	c = updated
	if cmd == nil {
		t.Fatal("expected a DeleteConnectionMsg cmd")
	}
	msg := cmd()
	if msg == nil {
		t.Fatal("expected a DeleteConnectionMsg")
	}
	dm, ok := msg.(DeleteConnectionMsg)
	if !ok || dm.Name != "local" {
		t.Errorf("msg = %#v, want DeleteConnectionMsg{Name: local}", msg)
	}
}

func TestConnScreen_Validate(t *testing.T) {
	c := NewConnScreen(nil)
	c.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	// sqlite without path -> error
	if err := c.validate(); err == nil {
		t.Error("expected error: sqlite without path")
	}
	c.fields[0].input.SetValue("/data/app.db")
	if err := c.validate(); err != nil {
		t.Errorf("validate = %v, want nil", err)
	}
}

func TestConnScreen_CycleDriverUpdatesConfig(t *testing.T) {
	c := NewConnScreen(nil)
	c.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	c.cycleDriver(true) // postgres
	cfg := c.cfg()
	if cfg.Driver != conn.DriverPostgres || cfg.Port != 5432 {
		t.Errorf("cfg = %+v, want postgres:5432", cfg)
	}
}

func TestConnScreen_ValuesSurviveDriverSwitch(t *testing.T) {
	c := NewConnScreen(nil)
	c.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	c.field("File").input.SetValue("/data/app.db")
	c.field("Host").input.SetValue("db.example.com")
	c.field("Port").input.SetValue("5433")

	c.cycleDriver(true) // sqlite -> postgres
	if got := c.field("Host").input.Value(); got != "db.example.com" {
		t.Errorf("Host after switch = %q, want preserved", got)
	}
	if got := c.field("Port").input.Value(); got != "5433" {
		t.Errorf("Port after switch = %q, want preserved", got)
	}

	c.cycleDriver(false) // postgres -> sqlite
	if got := c.field("File").input.Value(); got != "/data/app.db" {
		t.Errorf("File after switch back = %q, want preserved", got)
	}
}

func TestConnScreen_ValidatePort(t *testing.T) {
	c := NewConnScreen(nil)
	c.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	c.cycleDriver(true) // postgres
	c.field("Host").input.SetValue("localhost")

	c.field("Port").input.SetValue("not-a-number")
	if err := c.validate(); err == nil {
		t.Error("expected error for a non-numeric port")
	}
	c.field("Port").input.SetValue("99999")
	if err := c.validate(); err == nil {
		t.Error("expected error for a port out of range")
	}
	c.field("Port").input.SetValue("")
	if err := c.validate(); err != nil {
		t.Errorf("empty port (use default) = %v, want nil", err)
	}
	c.field("Port").input.SetValue("5432")
	if err := c.validate(); err != nil {
		t.Errorf("valid port = %v, want nil", err)
	}
}

func TestConnScreen_SaveValidatesBeforeSaving(t *testing.T) {
	c := NewConnScreen(nil)
	c.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	// empty sqlite path: ctrl+s must not emit a SaveConnectionMsg
	updated, cmd := c.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	c = updated
	if cmd != nil {
		t.Fatal("ctrl+s with invalid form must not emit a save command")
	}
	if c.err == "" {
		t.Error("expected a visible error on invalid save")
	}
}

// centerDelta returns twice the horizontal center of a line minus the ideal
// center of a `width`-wide terminal (59.5 for 120), so centered content is 0.
func centerDelta(line string, width int) (int, bool) {
	plain := ansi.Strip(line)
	trimmed := strings.TrimSpace(plain)
	if trimmed == "" {
		return 0, false
	}
	start := strings.Index(plain, trimmed)
	w := runewidth.StringWidth(trimmed)
	center := start + (w-1)/2
	return 2*center - (width - 1), true
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func TestConnScreen_ArrowsMoveCaretInField(t *testing.T) {
	c := NewConnScreen(nil)
	c.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	// focus on the File field (sqlite: engine(0) + File(1))
	c.focus = 1
	c.applyFocus()
	f := c.field("File")
	f.input.SetValue("abcd")
	updated, _ := f.input.Update(tea.KeyMsg{Type: tea.KeyLeft})
	f.input = updated
	if p := f.input.Position(); p != 3 {
		t.Fatalf("left arrow on a field should move the caret, position = %d, want 3", p)
	}
	updated, _ = f.input.Update(tea.KeyMsg{Type: tea.KeyRight})
	f.input = updated
	if p := f.input.Position(); p != 4 {
		t.Fatalf("right arrow on a field should move the caret, position = %d, want 4", p)
	}
	// the screen must not swallow the arrow when a field has the focus
	c.fields[0].input = f.input
	c.focus = 1
	c.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if p := c.field("File").input.Position(); p != 3 {
		t.Errorf("ConnScreen must forward left arrow to the focused field, position = %d, want 3", p)
	}
}

func TestConnScreen_PortResetsToNewEngineDefault(t *testing.T) {
	c := NewConnScreen(nil)
	c.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	// sqlite -> postgres: the port was empty (sqlite has no port), so it lands
	// on the postgres default placeholder
	c.cycleDriver(true)
	if got := c.field("Port").input.Value(); got != "" {
		t.Errorf("postgres default port value = %q, want empty (placeholder %q)",
			got, c.field("Port").input.Placeholder)
	}

	// postgres -> mysql: the port is still empty, so the mysql default applies
	c.field("Port").input.SetValue("5432") // postgres default typed
	c.cycleDriver(true)                    // -> mysql (default 3306)
	if got := c.field("Port").input.Value(); got != "" {
		t.Errorf("after cycling with the old default, port = %q, want empty", got)
	}
	c.field("Port").input.SetValue("1234") // custom port
	c.cycleDriver(true)                    // -> mariadb
	if got := c.field("Port").input.Value(); got != "1234" {
		t.Errorf("custom port should survive the switch, got %q", got)
	}
}

func TestConnScreen_ContentHorizontallyCentered(t *testing.T) {
	for _, w := range []int{80, 120, 160} {
		w := w
		c := NewConnScreen(nil)
		out := c.View(w, 30)

		// cell column of a byte index (chars before `sub` may include multibyte)
		colOf := func(plain, sub string) int {
			return runewidth.StringWidth(plain[:strings.Index(plain, sub)])
		}

		var logoOK bool
		var boxLeft, boxRight, boxTopCol = -1, -1, -1
		for _, l := range strings.Split(out, "\n") {
			plain := ansi.Strip(l)
			trimmed := strings.TrimSpace(plain)
			switch {
			case strings.HasPrefix(trimmed, "_____"): // logo line 1
				d, ok := centerDelta(l, w)
				if !ok {
					continue
				}
				logoOK = true
				if d < -3 || d > 3 {
					t.Errorf("width %d: logo center off by %d cols: %q", w, d, l)
				}
			case strings.Contains(trimmed, "Engine │"): // a field row (label + box)
				boxLeft = runewidth.StringWidth(plain[:strings.Index(plain, "│")])
				boxRight = runewidth.StringWidth(plain[:strings.LastIndex(plain, "│")])
			case strings.HasPrefix(trimmed, "╭"): // the box top border line
				boxTopCol = colOf(plain, "╭")
			}
		}
		if !logoOK {
			t.Fatalf("width %d: logo line not found in the connect screen", w)
		}
		if boxLeft < 0 || boxRight < 0 {
			t.Fatalf("width %d: form field row not found in the connect screen", w)
		}

		// The input box itself is the visual anchor. Labels intentionally sit
		// to its left and must not move the box away from the terminal center.
		boxCenter2 := boxLeft + boxRight
		if d := absInt(boxCenter2 - (w - 1)); d > 1 {
			t.Errorf("width %d: form box center off by %d cols (box %d..%d)", w, d, boxLeft, boxRight)
		}

		// the box top border must align with the box content
		if boxTopCol < 0 {
			t.Error("box top border not found")
		} else if boxTopCol != boxLeft {
			t.Errorf("width %d: box top border at col %d but content at %d", w, boxTopCol, boxLeft)
		}
	}
}

func TestLogoLinesAreUniform(t *testing.T) {
	lines := strings.Split(logoASCII, "\n")
	w := runewidth.StringWidth(lines[0])
	for i, l := range lines {
		if got := runewidth.StringWidth(l); got != w {
			t.Errorf("logo line %d is %d cells wide, want %d (uniform, so centering does not skew)", i, got, w)
		}
	}
}
