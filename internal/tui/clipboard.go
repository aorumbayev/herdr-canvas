package tui

import (
	"github.com/atotto/clipboard"

	tea "charm.land/bubbletea/v2"
)

// clipboardRead reads the OS clipboard. Tests replace it. OSC52
// (tea.ReadClipboard) is skipped: many terminals refuse clipboard reads, so
// Ctrl+V would silently do nothing.
var clipboardRead = clipboard.ReadAll

func readClipboardCmd() tea.Cmd {
	return func() tea.Msg {
		s, err := clipboardRead()
		if err != nil {
			s = ""
		}
		return tea.ClipboardMsg{Content: s}
	}
}
