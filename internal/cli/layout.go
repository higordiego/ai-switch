package cli

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

type layoutMode int

const (
	layoutFull layoutMode = iota
	layoutCompact
	layoutMinimal
)

type layout struct {
	width  int
	height int
	mode   layoutMode
}

func newLayout(width, height int) layout {
	if width < 40 {
		width = 40
	}
	if height < 12 {
		height = 12
	}
	mode := layoutFull
	switch {
	case width < 60:
		mode = layoutMinimal
	case width < 80:
		mode = layoutCompact
	}
	return layout{width: width, height: height, mode: mode}
}

func (l layout) contentWidth() int {
	w := l.width - 2
	if w < 36 {
		return 36
	}
	return w
}

func (l layout) innerWidth() int {
	w := l.contentWidth() - 2
	if w < 32 {
		return 32
	}
	return w
}

func truncate(value string, max int) string {
	if max <= 0 {
		return ""
	}
	if utf8.RuneCountInString(value) <= max {
		return value
	}
	if max < 4 {
		return string([]rune(value)[:max])
	}
	return string([]rune(value)[:max-3]) + "..."
}

func padRight(value string, width int) string {
	n := lipgloss.Width(value)
	if n >= width {
		return truncate(value, width)
	}
	return value + strings.Repeat(" ", width-n)
}

func repeat(char string, n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(char, n)
}

func sectionRule(title string, width int) string {
	line := accentStyle.Render(title)
	rest := width - lipgloss.Width(title)
	if rest < 1 {
		return line
	}
	return line + "\n" + mutedStyle.Render(repeat("─", width))
}

func framedBlock(title, body string, width int) string {
	if width < 20 {
		width = 20
	}
	inner := width - 2
	var top string
	if title == "" {
		top = "╭" + repeat("─", inner) + "╮"
	} else {
		label := "[ " + title + " ]"
		fill := inner - lipgloss.Width(label) - 1
		if fill < 1 {
			label = truncate(label, inner-2)
			fill = inner - lipgloss.Width(label) - 1
			if fill < 1 {
				fill = 1
			}
		}
		top = "╭─" + label + repeat("─", fill) + "╮"
	}
	var b strings.Builder
	b.WriteString(bannerStyle.Render(top))
	b.WriteByte('\n')
	for _, line := range strings.Split(body, "\n") {
		content := padRight(line, inner)
		b.WriteString(bannerStyle.Render("│") + content + bannerStyle.Render("│"))
		b.WriteByte('\n')
	}
	b.WriteString(bannerStyle.Render("╰" + repeat("─", inner) + "╯"))
	return b.String()
}

func appHeader(l layout, breadcrumb, right string) string {
	w := l.contentWidth()
	inner := w - 2
	left := " " + symBrand + " AISWITCH"
	if breadcrumb != "" {
		left += " / " + breadcrumb
	}
	if right == "" && breadcrumb == "" {
		right = "v" + Version
	}
	gap := inner - lipgloss.Width(left) - lipgloss.Width(right) - 1
	if gap < 1 {
		left = truncate(strings.TrimSpace(left), max(4, inner-lipgloss.Width(right)-2))
		gap = inner - lipgloss.Width(left) - lipgloss.Width(right) - 1
		if gap < 1 {
			gap = 1
			right = ""
		}
	}
	line1 := left + repeat(" ", gap) + right
	if breadcrumb == "" {
		sub := " isolated identities for coding agents"
		if l.mode == layoutMinimal {
			sub = " identity control"
		}
		body := padRight(brandStyle.Render(line1), inner) + "\n" + padRight(mutedStyle.Render(sub), inner)
		topLabel := "AISWITCH // IDENTITY CONTROL"
		if l.mode != layoutFull {
			topLabel = "AISWITCH"
		}
		return framedBlock(topLabel, body, w)
	}
	return framedBlock("", padRight(brandStyle.Render(line1), inner), w)
}

func feedbackLine(kind, text string) string {
	switch kind {
	case "success":
		return greenStyle.Render(symOK+" "+text)
	case "warning":
		return yellowStyle.Render(symWarn+" "+text)
	case "error":
		return redStyle.Render(symFail+" "+text)
	case "loading":
		return mutedStyle.Render(symLoad+" "+text)
	default:
		return mutedStyle.Render(text)
	}
}

func footerBar(l layout, parts []string) string {
	w := l.contentWidth()
	rule := mutedStyle.Render(repeat("─", w))
	if len(parts) == 0 {
		return rule
	}
	joined := strings.Join(parts, "   ")
	if lipgloss.Width(joined) > w {
		// Drop secondary hints on narrow terminals.
		for len(parts) > 2 && lipgloss.Width(strings.Join(parts, "   ")) > w {
			parts = parts[:len(parts)-1]
		}
		joined = strings.Join(parts, "  ")
		if lipgloss.Width(joined) > w {
			joined = truncate(joined, w)
		}
	}
	return rule + "\n" + mutedStyle.Render(joined)
}

func badgeActive() string {
	return greenStyle.Render(symActive + " ACTIVE")
}

func badgeLinked() string {
	return greenStyle.Render(symOn + " LINKED")
}

func badgeNotLinked() string {
	return mutedStyle.Render(symOff + " NOT LINKED")
}

func agentMark(linked bool) string {
	if linked {
		return greenStyle.Render(symOn)
	}
	return mutedStyle.Render(symOff)
}
