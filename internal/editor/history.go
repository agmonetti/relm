package editor

import "strings"

// History is a ring buffer of executed queries (last 100).
type History struct {
	items []string
	pos   int // navigation position; -1 = end (typing a new query)
	max   int
}

// NewHistory creates an empty history with a default capacity of 100.
func NewHistory() *History {
	return &History{pos: -1, max: 100}
}

// Push adds a query to the history. Ignores empty and consecutive duplicates.
// Resets navigation (the user goes back to typing from the end).
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

// Prev returns the previous query in the history, or the first one if already there.
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

// Next returns the next query in the history, or "" when reaching the end.
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

// Reset stops navigation (the user started editing the buffer).
func (h *History) Reset() { h.pos = -1 }

// InNavigation reports whether the user is navigating the history.
func (h *History) InNavigation() bool { return h.pos >= 0 }

// Len returns the number of stored queries.
func (h *History) Len() int { return len(h.items) }
