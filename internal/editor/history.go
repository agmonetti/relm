package editor

import "strings"

// History es un ring buffer de queries ejecutados (últimos 100).
type History struct {
	items []string
	pos   int // posición de navegación; -1 = fin (escribiendo un query nuevo)
	max   int
}

// NewHistory crea un historial vacío con capacidad default de 100.
func NewHistory() *History {
	return &History{pos: -1, max: 100}
}

// Push agrega un query al historial. Ignora vacíos y duplicados consecutivos.
// Resetea la navegación (el usuario vuelve a escribir desde el fin).
func (h *History) Push(q string) {
	q = strings.TrimSpace(q)
	if q == "" {
		return
	}
	if n := len(h.items); n > 0 && h.items[n-1] == q {
		h.pos = -1
		return
	}
	h.items = append(h.items, q)
	if len(h.items) > h.max {
		h.items = h.items[1:]
	}
	h.pos = -1
}

// Prev devuelve el query anterior del historial, o el primero si ya estamos en él.
func (h *History) Prev() string {
	if len(h.items) == 0 {
		return ""
	}
	if h.pos == -1 {
		h.pos = len(h.items) - 1
	} else if h.pos > 0 {
		h.pos--
	}
	return h.items[h.pos]
}

// Next devuelve el query siguiente del historial, o "" si llegamos al fin.
func (h *History) Next() string {
	if h.pos == -1 || len(h.items) == 0 {
		return ""
	}
	if h.pos < len(h.items)-1 {
		h.pos++
		return h.items[h.pos]
	}
	h.pos = -1
	return ""
}

// Reset detiene la navegación (el usuario empezó a editar el buffer).
func (h *History) Reset() { h.pos = -1 }

// InNavigation indica si el usuario está navegando el historial.
func (h *History) InNavigation() bool { return h.pos >= 0 }

// Len devuelve la cantidad de queries guardados.
func (h *History) Len() int { return len(h.items) }
