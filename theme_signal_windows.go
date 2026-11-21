//go:build windows

package main

import tea "charm.land/bubbletea/v2"

func ignoreThemeSignal()                 {}
func reloadThemeOnSignal(_ *tea.Program) {}
