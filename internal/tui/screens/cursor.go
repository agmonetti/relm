package screens

import "time"

// CursorBlink is the text input/textarea cursor blink rate. The bubbletea test
// helpers drive commands synchronously, so the 530ms default would block for
// half a second on every focus; the tests set it to a tiny value.
var CursorBlink = 530 * time.Millisecond
