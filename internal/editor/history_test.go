package editor

import "testing"

func TestHistory_PushIgnoresConsecutiveDuplicates(t *testing.T) {
	h := NewHistory()
	h.Push("SELECT 1")
	h.Push("SELECT 1")
	h.Push("SELECT 2")
	if h.Len() != 2 {
		t.Errorf("Len = %d, want 2", h.Len())
	}
}

func TestHistory_PushIgnoresEmpty(t *testing.T) {
	h := NewHistory()
	h.Push("")
	h.Push("   ")
	if h.Len() != 0 {
		t.Errorf("Len = %d, want 0", h.Len())
	}
}

func TestHistory_Navigation(t *testing.T) {
	h := NewHistory()
	h.Push("a")
	h.Push("b")
	h.Push("c")

	if got := h.Prev(); got != "c" {
		t.Errorf("Prev() = %q, want c", got)
	}
	if got := h.Prev(); got != "b" {
		t.Errorf("Prev() = %q, want b", got)
	}
	if got := h.Prev(); got != "a" {
		t.Errorf("Prev() = %q, want a", got)
	}
	if got := h.Prev(); got != "a" {
		t.Errorf("Prev() at start = %q, want a", got)
	}
	if got := h.Next(); got != "b" {
		t.Errorf("Next() = %q, want b", got)
	}
	if got := h.Next(); got != "c" {
		t.Errorf("Next() = %q, want c", got)
	}
	if got := h.Next(); got != "" {
		t.Errorf("Next() at end = %q, want empty", got)
	}
}

func TestHistory_ResetStopsNavigation(t *testing.T) {
	h := NewHistory()
	h.Push("a")
	h.Push("b")
	h.Prev() // navigating to "b"
	h.Reset()
	if got := h.Next(); got != "" {
		t.Errorf("Next() after Reset = %q, want empty", got)
	}
	if got := h.Prev(); got != "b" {
		t.Errorf("Prev() after Reset = %q, want b", got)
	}
}

func TestHistory_CapsAt100(t *testing.T) {
	h := NewHistory()
	for i := 0; i < 120; i++ {
		h.Push(string(rune('a' + i%26)))
	}
	if h.Len() != 100 {
		t.Errorf("Len = %d, want 100", h.Len())
	}
}
