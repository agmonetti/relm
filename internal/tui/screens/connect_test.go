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

	// sqlite por defecto: Archivo + toggle Solo lectura
	if len(c.fieldsVisible()) != 2 {
		t.Fatalf("fieldsVisible = %d, want 2 (Archivo + Solo lectura)", len(c.fieldsVisible()))
	}
	if c.fieldsVisible()[0].label != "Archivo" {
		t.Errorf("campo = %q, want Archivo", c.fieldsVisible()[0].label)
	}
	if c.fieldsVisible()[1].label != "Solo lectura" || !c.fieldsVisible()[1].isToggle {
		t.Errorf("segundo campo = %+v, want toggle Solo lectura", c.fieldsVisible()[1])
	}
}

func TestConnScreen_NetworkShowsAllFields(t *testing.T) {
	c := NewConnScreen(nil)
	c.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	c.cycleDriver(true) // sqlite -> postgres

	if len(c.fieldsVisible()) != 6 {
		t.Fatalf("fieldsVisible = %d, want 6 (Host..Base + SSL)", len(c.fieldsVisible()))
	}
	// password enmascarada
	if c.fields[4].input.EchoMode != textinput.EchoPassword {
		t.Error("Password debería estar enmascarada en motores de red")
	}
}

func TestConnScreen_ToggleSoloLectura(t *testing.T) {
	c := NewConnScreen(nil)
	c.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	c.fields[0].input.SetValue("/data/app.db")

	c.focus = 2 // toggle Solo lectura (campo 2 de sqlite)
	c.Update(tea.KeyMsg{Type: tea.KeySpace})
	if !c.cfg().ReadOnly {
		t.Error("ReadOnly debería ser true tras alternar el toggle")
	}
	c.Update(tea.KeyMsg{Type: tea.KeySpace})
	if c.cfg().ReadOnly {
		t.Error("ReadOnly debería volver a false tras alternar dos veces")
	}
}

func TestConnScreen_SSLModeEnConfig(t *testing.T) {
	c := NewConnScreen(nil)
	c.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	c.cycleDriver(true) // postgres
	// campo SSL es el último de postgres (índice 5 en fieldsVisible)
	vis := c.fieldsVisible()
	if len(vis) != 6 || vis[5].label != "SSL" {
		t.Fatalf("esperaba SSL como sexto campo, got %+v", vis)
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
	c.fields[1].input.SetValue("localhost") // host obligatorio
	vis := c.fieldsVisible()
	vis[5].input.SetValue("bogus")
	if err := c.validate(); err == nil {
		t.Error("esperaba error por sslmode inválido")
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
