package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/agmonetti/relm/internal/browser"
	"github.com/agmonetti/relm/internal/store"
)

// browserDoneMsg carries the result of an asynchronous browser operation (a
// page change, a table selection, a reload or the initial connection load).
type browserDoneMsg struct {
	token int
	navID int // matches m.navID; guards against superseded/cancelled ops
	err   error
	b     *browser.Browser // the updated browser; nil on error
	load  bool             // true when it was the initial connection load
}

// runBrowserOp runs a browser mutation (select table, change page, reload) on
// a background goroutine. The mutation happens on a clone, so the UI goroutine
// never observes a half-updated Browser; on success the clone is swapped in.
// Only one navigation may run at a time and it is bounded by the configured
// query timeout, so a slow or hung engine shows a spinner instead of freezing.
func (m *Model) runBrowserOp(op func(b *browser.Browser, st store.Store, ctx context.Context) error) tea.Cmd {
	if m.browser == nil || m.store == nil || m.navigating {
		return nil
	}
	clone := m.browser.Clone()
	st := m.store
	token := m.queryID
	m.navID++
	navID := m.navID
	timeout := time.Duration(m.prefs.QueryTimeout()) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	m.cancelNav()
	m.navCancel = cancel
	m.navigating = true
	return tea.Batch(
		func() tea.Msg {
			if err := op(clone, st, ctx); err != nil {
				return browserDoneMsg{token: token, navID: navID, err: err}
			}
			return browserDoneMsg{token: token, navID: navID, b: clone}
		},
		m.spinner.Tick,
	)
}

// loadBrowserCmd loads the tables and the first page of a freshly connected
// store in the background. Unlike runBrowserOp there is no clone: the Browser
// does not exist yet, so it is built entirely off-screen and swapped in.
func (m *Model) loadBrowserCmd(st store.Store) tea.Cmd {
	token := m.queryID
	timeout := time.Duration(m.prefs.QueryTimeout()) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	m.cancelConnect()
	m.connectCancel = cancel
	m.connecting = true
	return func() tea.Msg {
		b, err := browser.New(ctx, st)
		if err != nil {
			return browserDoneMsg{token: token, err: err, load: true}
		}
		return browserDoneMsg{token: token, b: b, load: true}
	}
}

// cancelNav cancels a running browser navigation, if any.
func (m *Model) cancelNav() {
	if m.navCancel != nil {
		m.navCancel()
		m.navCancel = nil
	}
}

// cancelConnect cancels a running initial connection load, if any.
func (m *Model) cancelConnect() {
	if m.connectCancel != nil {
		m.connectCancel()
		m.connectCancel = nil
	}
}

// busy reports whether any background operation is running (a query, a
// navigation or a connection load); the footer shows the spinner meanwhile.
func (m *Model) busy() bool {
	return m.loading || m.navigating || m.connecting
}
