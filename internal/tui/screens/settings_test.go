package screens

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSettingsScreen_ViewRenders(t *testing.T) {
	s := NewSettingsScreen()
	out := s.View(80, 20)
	if !strings.Contains(out, "Settings") {
		t.Errorf("settings view missing title: %q", out)
	}
	if !strings.Contains(out, "query timeout") {
		t.Errorf("settings view missing field label: %q", out)
	}
}

func TestSettingsScreen_EscapeEmitsBack(t *testing.T) {
	s := NewSettingsScreen()
	updated, cmd := s.Update(tea.KeyMsg{Type: tea.KeyEsc})
	s = updated
	if cmd == nil {
		t.Fatal("expected a SettingsBackMsg cmd on Esc")
	}
	if _, ok := cmd().(SettingsBackMsg); !ok {
		t.Errorf("cmd() = %#v, want SettingsBackMsg", cmd())
	}
}

func TestSettingsScreen_EnterEmitsSettingsMsg(t *testing.T) {
	s := NewSettingsScreen()
	s.SetValue("90")
	updated, cmd := s.Update(tea.KeyMsg{Type: tea.KeyEnter})
	s = updated
	if cmd == nil {
		t.Fatal("expected a SettingsMsg cmd on Enter")
	}
	sm, ok := cmd().(SettingsMsg)
	if !ok {
		t.Fatalf("cmd() = %#v, want SettingsMsg", cmd())
	}
	if sm.QueryTimeoutSeconds != 90 {
		t.Errorf("QueryTimeoutSeconds = %d, want 90", sm.QueryTimeoutSeconds)
	}
}

func TestSettingsScreen_RejectsInvalid(t *testing.T) {
	for _, v := range []string{"", "abc", "0", "-3", "100000"} {
		s := NewSettingsScreen()
		s.SetValue(v)
		updated, cmd := s.Update(tea.KeyMsg{Type: tea.KeyEnter})
		s = updated
		if cmd != nil {
			t.Errorf("value %q: expected no cmd, got %#v", v, cmd)
		}
		if s.err == "" {
			t.Errorf("value %q: expected an error message", v)
		}
	}
}

func TestSettingsScreen_FocusOnField(t *testing.T) {
	s := NewSettingsScreen()
	if !s.FocusOnField() {
		t.Error("input should be focused by default")
	}
	s.Blur()
	if s.FocusOnField() {
		t.Error("input should be blurred after Blur")
	}
}
