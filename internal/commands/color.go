package commands

import (
	"os"
	"runtime"
	"strings"

	"golang.org/x/term"
)

// Terminal color support. Populated in init().
var (
	cBold  string
	cGold  string // golden orange (214)
	cOrange string // blood orange (202)
	cGreen string // vibrant green (40)
	cUline string
	cReset string
)

// noColor is true when colors should be suppressed.
var noColor bool

func init() {
	if os.Getenv("NO_COLOR") != "" {
		noColor = true
		return
	}

	// On Windows, disable colors by default unless running in Windows Terminal.
	// Legacy PowerShell and cmd.exe don't support ANSI codes properly.
	if runtime.GOOS == "windows" {
		// WT_SESSION is set by Windows Terminal
		// TERM=xterm* indicates a proper terminal emulator
		if os.Getenv("WT_SESSION") == "" && !strings.HasPrefix(os.Getenv("TERM"), "xterm") {
			noColor = true
			return
		}
	}

	fd := int(os.Stdout.Fd())
	if !term.IsTerminal(fd) {
		noColor = true
		return
	}
	cBold = "\033[1m"
	cGold = "\033[38;5;214m"
	cOrange = "\033[38;5;202m"
	cGreen = "\033[38;5;40m"
	cUline = "\033[4m"
	cReset = "\033[0m"
}

// F formats help text with simple markup tags:
//
//	{b}...{/b}  → bold         (section headings, labels)
//	{c}...{/c}  → blood orange (command names, syntax patterns)
//	{g}...{/g}  → green        (descriptions, notes)
//	{y}...{/y}  → golden       (labels, highlights)
//	{u}...{/u}  → underline    (URLs)
//
// When color is disabled (NO_COLOR, non-terminal), tags are stripped.
func F(s string) string {
	if noColor {
		s = strings.ReplaceAll(s, "{b}", "")
		s = strings.ReplaceAll(s, "{/b}", "")
		s = strings.ReplaceAll(s, "{c}", "")
		s = strings.ReplaceAll(s, "{/c}", "")
		s = strings.ReplaceAll(s, "{g}", "")
		s = strings.ReplaceAll(s, "{/g}", "")
		s = strings.ReplaceAll(s, "{y}", "")
		s = strings.ReplaceAll(s, "{/y}", "")
		s = strings.ReplaceAll(s, "{u}", "")
		s = strings.ReplaceAll(s, "{/u}", "")
		return s
	}
	s = strings.ReplaceAll(s, "{b}", cBold)
	s = strings.ReplaceAll(s, "{/b}", cReset)
	s = strings.ReplaceAll(s, "{c}", cOrange)
	s = strings.ReplaceAll(s, "{/c}", cReset)
	s = strings.ReplaceAll(s, "{g}", cGreen)
	s = strings.ReplaceAll(s, "{/g}", cReset)
	s = strings.ReplaceAll(s, "{y}", cGold)
	s = strings.ReplaceAll(s, "{/y}", cReset)
	s = strings.ReplaceAll(s, "{u}", cUline)
	s = strings.ReplaceAll(s, "{/u}", cReset)
	return s
}
