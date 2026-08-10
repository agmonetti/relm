package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap agrupa todos los keybindings de la aplicación.
type KeyMap struct {
	Quit          key.Binding
	NewSession    key.Binding
	SaveConn      key.Binding
	ToggleSidebar key.Binding
	Help          key.Binding
	Switch        key.Binding
	Inspect       key.Binding
	Refresh       key.Binding
	Execute       key.Binding
	ClearInput    key.Binding
	LineStart     key.Binding
	LineEnd       key.Binding
	Back          key.Binding

	Up       key.Binding
	Down     key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	First    key.Binding
	Last     key.Binding
}

// DefaultKeyMap devuelve el keymap completo.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit:          key.NewBinding(key.WithKeys("ctrl+c", "q"), key.WithHelp("ctrl+c / q", "salir")),
		NewSession:    key.NewBinding(key.WithKeys("ctrl+n"), key.WithHelp("ctrl+n", "nueva conexión")),
		SaveConn:      key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "guardar conexión")),
		ToggleSidebar: key.NewBinding(key.WithKeys("alt+b"), key.WithHelp("alt+b", "sidebar")),
		Help:          key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "ayuda")),
		Switch:        key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "browser/editor")),
		Inspect:       key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "estructura")),
		Refresh:       key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refrescar")),
		Execute:       key.NewBinding(key.WithKeys("ctrl+r"), key.WithHelp("ctrl+r", "ejecutar")),
		ClearInput:    key.NewBinding(key.WithKeys("ctrl+l"), key.WithHelp("ctrl+l", "limpiar")),
		LineStart:     key.NewBinding(key.WithKeys("ctrl+a"), key.WithHelp("ctrl+a", "inicio de línea")),
		LineEnd:       key.NewBinding(key.WithKeys("ctrl+e"), key.WithHelp("ctrl+e", "fin de línea")),
		Back:          key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "volver")),

		Up:       key.NewBinding(key.WithKeys("up", "k")),
		Down:     key.NewBinding(key.WithKeys("down", "j")),
		PageUp:   key.NewBinding(key.WithKeys("pgup", "ctrl+u")),
		PageDown: key.NewBinding(key.WithKeys("pgdown", "ctrl+d")),
		First:    key.NewBinding(key.WithKeys("g", "home")),
		Last:     key.NewBinding(key.WithKeys("G", "end")),
	}
}

// ShortHelp implementa help.KeyMap.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Quit, k.NewSession, k.Switch, k.Inspect, k.Help,
	}
}

// FullHelp implementa help.KeyMap.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Quit, k.NewSession, k.Switch, k.Inspect, k.Refresh},
		{k.ToggleSidebar, k.Help, k.Back, k.Execute, k.ClearInput},
	}
}
