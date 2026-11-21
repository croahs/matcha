//go:build !windows

package main

import (
	"os"
	"os/signal"
	"syscall"

	tea "charm.land/bubbletea/v2"

	"github.com/floatpane/matcha/tui"
)

// ignoreThemeSignal keeps SIGUSR2 from killing non-interactive runs (such as
// the background daemon) when a theme switch signals every matcha process.
func ignoreThemeSignal() {
	signal.Ignore(syscall.SIGUSR2)
}

// reloadThemeOnSignal re-reads the theme when SIGUSR2 arrives, the way
// Omarchy tells btop and opencode to pick up a new theme.
func reloadThemeOnSignal(p *tea.Program) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGUSR2)
	go func() {
		for range ch {
			p.Send(tui.ReloadThemeMsg{})
		}
	}()
}
