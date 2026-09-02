package tui

import "github.com/atotto/clipboard"

// clipboardWriteAll is a variable so it can be mocked in tests.
var clipboardWriteAll = clipboard.WriteAll
