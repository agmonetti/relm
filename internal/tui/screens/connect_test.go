package screens

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"relm/internal/conn"
)

func TestConnScreen_SQLiteOnlyShowsPath(t *testing.T) {
	c := NewConnScreen(nil)
	c.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	// sqlite por defecto: solo campo Archivo visible
	if len(c.fieldsVisible()) != 1 {
		t.Fatalf("fieldsVisible = %d, want 1 (solo Archivo)", len(c.fieldsVisible()))
	}
	if c.fieldsVisible()[0].label != "Archivo" {
		t.Errorf("campo = %q, want Archivo", c.fieldsVisible()[0].label)
	}
}

func TestConnScreen_NetworkShowsAllFields(t *testing.T) {
	c := NewConnScreen(nil)
	c.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	c.cycleDriver(true) // sqlite -> postgres

	if len(c.fieldsVisible()) != 5 {
		t.Fatalf("fieldsVisible = %d, want 5 (Host..Base)", len(c.fieldsVisible()))
	}
	// password enmascarada
	if c.fields[4].input.EchoMode != textinput.EchoPassword {
		t.Error("Password debería estar enmascarada en motores de red")
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
		t.Errorf("drivers vistos = %d, want 5", len(seen))
	}
}

func TestConnScreen_Validate(t *testing.T) {
	c := NewConnScreen(nil)
	c.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	// sqlite sin path -> error
	if err := c.validate(); err == nil {
		t.Error("esperaba error: sqlite sin path")
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
