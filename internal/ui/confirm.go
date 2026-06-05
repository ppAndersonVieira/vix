package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// maskSecret renders a secret for display with its first 6 runes in clear and
// the remainder replaced by bullets, so users can confirm they pasted the right
// key without exposing it. Secrets of 6 runes or fewer are shown verbatim.
func maskSecret(sval string) string {
	r := []rune(sval)
	if len(r) <= 6 {
		return sval
	}
	return string(r[:6]) + strings.Repeat("•", len(r)-6)
}

// renderKeyInputDialog renders the credential-entry popup as a centered overlay.
// provider is the human-readable provider label; masked is the masked value to
// display (see maskSecret).
func renderKeyInputDialog(width, height int, s Styles, provider, masked string) string {
	dialogWidth := 56
	if dialogWidth > width-4 {
		dialogWidth = width - 4
	}
	innerWidth := dialogWidth - 4

	title := lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).
		Width(innerWidth).Align(lipgloss.Center).
		Render("Set " + provider + " API key")

	sep := s.CommandPaletteSepStyle.Width(innerWidth).Render(strings.Repeat("─", innerWidth))

	field := masked
	if field == "" {
		field = lipgloss.NewStyle().Foreground(s.ColorDimGray).Render("Paste your key…")
	}
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorSecondary).
		Width(innerWidth - 2).
		Render(field)

	hint := lipgloss.NewStyle().Foreground(s.ColorDimGray).
		Width(innerWidth).Align(lipgloss.Center).
		Render("Enter save · Esc cancel")

	content := title + "\n" + sep + "\n" + box + "\n" + hint
	return s.CommandPaletteStyle.Width(dialogWidth).Render(content)
}

// renderKeyDeleteDialog renders the credential-deletion confirmation as a
// centered overlay. kind is "api_key" or "oauth"; selected: 0 = Yes, 1 = No.
func renderKeyDeleteDialog(width, height int, s Styles, provider, kind string, selected int) string {
	dialogWidth := 54
	if dialogWidth > width-4 {
		dialogWidth = width - 4
	}
	innerWidth := dialogWidth - 4

	what := "API key"
	if kind == "oauth" {
		what = "OAuth token"
	}

	title := lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).
		Width(innerWidth).Align(lipgloss.Center).
		Render("Delete " + what + "?")

	sep := s.CommandPaletteSepStyle.Width(innerWidth).Render(strings.Repeat("─", innerWidth))

	msg := lipgloss.NewStyle().Foreground(s.ColorDimGray).
		Width(innerWidth).Align(lipgloss.Center).
		Render("The stored " + provider + " " + what + " will be removed. This cannot be undone.")

	yesStyle := lipgloss.NewStyle().Bold(true).Foreground(s.ColorDimGray)
	noStyle := lipgloss.NewStyle().Bold(true).Foreground(s.ColorDimGray)
	if selected == 0 {
		yesStyle = yesStyle.Foreground(colorSecondary)
	} else {
		noStyle = noStyle.Foreground(colorSecondary)
	}

	buttons := lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center).
		Render(yesStyle.Render("Yes") + "    " + noStyle.Render("No"))

	content := title + "\n" + sep + "\n" + msg + "\n\n" + buttons
	return s.CommandPaletteStyle.Width(dialogWidth).Render(content)
}

// renderTrimDialog renders the trim confirmation as a centered overlay box.
// selected: 0 = Yes, 1 = No.
func renderTrimDialog(width, height int, s Styles, selected int) string {
	dialogWidth := 50
	if dialogWidth > width-4 {
		dialogWidth = width - 4
	}
	innerWidth := dialogWidth - 4 // account for border + padding

	title := lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).
		Width(innerWidth).Align(lipgloss.Center).
		Render("✂  Trim conversation?")

	sep := s.CommandPaletteSepStyle.Width(innerWidth).Render(strings.Repeat("─", innerWidth))

	msg := lipgloss.NewStyle().Foreground(s.ColorDimGray).
		Width(innerWidth).Align(lipgloss.Center).
		Render("All messages below this point will be permanently deleted.")

	yesStyle := lipgloss.NewStyle().Bold(true).Foreground(s.ColorDimGray)
	noStyle := lipgloss.NewStyle().Bold(true).Foreground(s.ColorDimGray)
	if selected == 0 {
		yesStyle = yesStyle.Foreground(colorSecondary)
	} else {
		noStyle = noStyle.Foreground(colorSecondary)
	}

	yesBtn := yesStyle.Render("Yes")
	noBtn := noStyle.Render("No")
	buttons := lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center).
		Render(yesBtn + "    " + noBtn)

	content := title + "\n" + sep + "\n" + msg + "\n\n" + buttons

	return s.CommandPaletteStyle.Width(dialogWidth).Render(content)
}

// renderSessionCloseDialog renders the session-close confirmation as a centered overlay box.
// selected: 0 = Yes, 1 = No.
func renderSessionCloseDialog(width, height int, s Styles, selected int, sessionID string) string {
	dialogWidth := 52
	if dialogWidth > width-4 {
		dialogWidth = width - 4
	}
	innerWidth := dialogWidth - 4 // account for border + padding

	title := lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).
		Width(innerWidth).Align(lipgloss.Center).
		Render("Close session?")

	sep := s.CommandPaletteSepStyle.Width(innerWidth).Render(strings.Repeat("─", innerWidth))

	body := "The session will be terminated."
	if sessionID != "" {
		body = body + "\n" + lipgloss.NewStyle().Foreground(s.ColorDimGray).Render(sessionID)
	}
	msg := lipgloss.NewStyle().Foreground(s.ColorDimGray).
		Width(innerWidth).Align(lipgloss.Center).
		Render(body)

	yesStyle := lipgloss.NewStyle().Bold(true).Foreground(s.ColorDimGray)
	noStyle := lipgloss.NewStyle().Bold(true).Foreground(s.ColorDimGray)
	if selected == 0 {
		yesStyle = yesStyle.Foreground(colorSecondary)
	} else {
		noStyle = noStyle.Foreground(colorSecondary)
	}

	yesBtn := yesStyle.Render("Yes")
	noBtn := noStyle.Render("No")
	buttons := lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center).
		Render(yesBtn + "    " + noBtn)

	content := title + "\n" + sep + "\n" + msg + "\n\n" + buttons

	return s.CommandPaletteStyle.Width(dialogWidth).Render(content)
}

// renderQuitDialog renders the quit confirmation as a centered overlay box,
// styled like the command palette. width/height are the terminal dimensions.
// selected: 0 = Yes, 1 = No.
func renderQuitDialog(width, height int, s Styles, selected int) string {
	dialogWidth := 44
	if dialogWidth > width-4 {
		dialogWidth = width - 4
	}
	innerWidth := dialogWidth - 4 // account for border + padding

	title := lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).
		Width(innerWidth).Align(lipgloss.Center).
		Render("Quit vix?")

	sep := s.CommandPaletteSepStyle.Width(innerWidth).Render(strings.Repeat("─", innerWidth))

	msg := lipgloss.NewStyle().Foreground(s.ColorDimGray).
		Width(innerWidth).Align(lipgloss.Center).
		Render("Any running agent will be cancelled.")

	yesStyle := lipgloss.NewStyle().Bold(true).Foreground(s.ColorDimGray)
	noStyle := lipgloss.NewStyle().Bold(true).Foreground(s.ColorDimGray)
	if selected == 0 {
		yesStyle = yesStyle.Foreground(colorSecondary)
	} else {
		noStyle = noStyle.Foreground(colorSecondary)
	}

	yesBtn := yesStyle.Render("Yes")
	noBtn := noStyle.Render("No")
	buttons := lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center).
		Render(yesBtn + "    " + noBtn)

	content := title + "\n" + sep + "\n" + msg + "\n\n" + buttons

	return s.CommandPaletteStyle.Width(dialogWidth).Render(content)
}
