package screens

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

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
	c.cycleDriver(true) // postgres
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
