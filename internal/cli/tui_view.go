package cli

import (
	"fmt"
	"strings"

	"aiswitch/internal/launch"

	"github.com/charmbracelet/lipgloss"
)

func (m tuiModel) View() string {
	if m.quit {
		return ""
	}
	l := m.layout()
	var b strings.Builder
	switch m.screen {
	case homeScreen:
		b.WriteString(m.viewHome(l))
	case profileScreen:
		b.WriteString(m.viewProfile(l))
	case toolScreen:
		b.WriteString(m.viewTools(l))
	case createScreen:
		b.WriteString(m.viewCreate(l))
	case renameScreen:
		b.WriteString(m.viewRename(l))
	case confirmScreen:
		b.WriteString(m.viewConfirm(l))
	case switchResultScreen:
		b.WriteString(m.viewSwitchResult(l))
	}
	if m.message != "" && m.screen != switchResultScreen {
		b.WriteString("\n" + feedbackLine(string(m.flash), m.message) + "\n")
	}
	b.WriteString("\n" + m.viewFooter(l))
	b.WriteString("\n" + mutedStyle.Render("// feito por ") + brandStyle.Render("Higor Diego"))
	return b.String()
}

func (m tuiModel) viewFooter(l layout) string {
	switch m.screen {
	case homeScreen:
		return footerBar(l, []string{"↑↓ Navigate", "↵ Open", "[a] Activate", "[n] New", "[d] Delete", "[q] Quit"})
	case profileScreen:
		return footerBar(l, []string{"↑↓ Navigate", "↵ Select", "[s] Switch", "[l] Link", "[u] Unlink", "[r] Rename", "esc Back", "q Quit"})
	case toolScreen:
		return footerBar(l, []string{"↑↓ Navigate", "↵ Select", "esc Back", "q Quit"})
	case createScreen:
		if m.createStep == 0 {
			return footerBar(l, []string{"type name", "↵ Next", "esc Cancel"})
		}
		return footerBar(l, []string{"↑↓ Move", "space Toggle", "↵ Create", "esc Cancel"})
	case renameScreen:
		return footerBar(l, []string{"type name", "↵ Save", "esc Cancel"})
	case confirmScreen:
		return footerBar(l, []string{"[y] Delete", "[n] Cancel"})
	case switchResultScreen:
		return footerBar(l, []string{"↵ Continue", "esc Back", "q Quit"})
	default:
		return footerBar(l, []string{"↑↓ Navigate", "↵ Select", "esc Back", "q Quit"})
	}
}

func (m tuiModel) viewHome(l layout) string {
	var b strings.Builder
	b.WriteString(appHeader(l, "", "") + "\n\n")
	count := fmt.Sprintf("%d configured", len(m.profiles))
	if l.mode == layoutMinimal {
		count = fmt.Sprintf("%d", len(m.profiles))
	}
	title := "PROFILES"
	b.WriteString(sectionRule(fmt.Sprintf("%s%s%s", title, repeat(" ", max(1, l.innerWidth()-len(title)-len(count))), count), l.innerWidth()) + "\n\n")

	for i, p := range m.profiles {
		selected := i == m.selected
		name := p.Name
		prefix := "  "
		if selected {
			prefix = selectedStyle.Render(symSelect+" ") + ""
			name = selectedStyle.Render(p.Name)
		} else {
			name = whiteStyle.Render(p.Name)
		}
		right := ""
		if p.Name == m.activeName {
			right = badgeActive()
		}
		lineWidth := l.innerWidth()
		left := prefix + name
		gap := lineWidth - lipgloss.Width(left) - lipgloss.Width(right)
		if gap < 1 {
			gap = 1
		}
		b.WriteString(left + repeat(" ", gap) + right + "\n")
		if l.mode != layoutMinimal || selected {
			for ti, tool := range launch.Tools {
				branch := symTree
				if ti == len(launch.Tools)-1 {
					branch = symTreeEnd
				}
				mark := agentMark(profileToolLinked(p, tool))
				label := toolDisplayName(tool)
				if l.mode == layoutMinimal {
					label = truncate(label, 10)
				}
				agentLine := fmt.Sprintf("    %s %s", branch, label)
				gap = lineWidth - lipgloss.Width(agentLine) - lipgloss.Width(mark) - 1
				if gap < 1 {
					gap = 1
				}
				style := mutedStyle
				if selected {
					style = whiteStyle
				}
				b.WriteString(style.Render(agentLine) + repeat(" ", gap) + mark + "\n")
			}
		}
		b.WriteByte('\n')
	}

	createSelected := m.selected == len(m.profiles)
	createLabel := symPlus + " Create profile"
	if createSelected {
		b.WriteString(selectedStyle.Render(symSelect+" "+createLabel) + "\n")
	} else {
		b.WriteString("  " + mutedStyle.Render(createLabel) + "\n")
	}
	return b.String()
}

func (m tuiModel) viewProfile(l layout) string {
	p := m.selectedProfile()
	right := ""
	if p.Name == m.activeName {
		right = badgeActive()
	}
	var b strings.Builder
	b.WriteString(appHeader(l, p.Name, right) + "\n\n")
	b.WriteString(sectionRule("LINKED AGENTS", l.innerWidth()) + "\n\n")

	nTools := len(launch.Tools)
	for i, tool := range launch.Tools {
		label := toolDisplayName(tool)
		status := badgeNotLinked()
		if m.isLinked(tool) {
			status = badgeLinked()
		}
		row := padRight(label, 28) + status
		if l.mode == layoutMinimal {
			row = padRight(truncate(label, 12), 14) + agentMark(m.isLinked(tool))
		}
		b.WriteString(m.row(m.selected == i, row) + "\n")
	}

	b.WriteString("\n" + sectionRule("ACTIONS", l.innerWidth()) + "\n\n")
	actions := []string{
		"[s] Switch to this profile",
		"[l] Link new agent",
		"[u] Unlink agent",
		"[r] Rename profile",
		"[d] Delete profile",
	}
	for i, action := range actions {
		b.WriteString(m.row(m.selected == nTools+i, action) + "\n")
	}
	return b.String()
}

func (m tuiModel) viewTools(l layout) string {
	p := m.selectedProfile()
	verb := map[string]string{"run": "Open", "login": "Link", "status": "Status", "logout": "Unlink"}[m.action]
	var b strings.Builder
	b.WriteString(appHeader(l, p.Name, "") + "\n\n")
	b.WriteString(sectionRule(strings.ToUpper(verb)+" // SELECT TOOL", l.innerWidth()) + "\n")
	b.WriteString(titleStyle.Render(verb+" qual ferramenta?") + "\n\n")
	for i, tool := range m.tools {
		label := fmt.Sprintf("%-17s %s", toolDisplayName(tool), toolDescription(tool))
		if m.action != "run" {
			label = fmt.Sprintf("%-17s %s %s", toolDisplayName(tool), m.statusCell(tool), toolDescription(tool))
		}
		if l.mode != layoutFull {
			label = toolDisplayName(tool) + "  " + statusChipText(m, tool)
		}
		b.WriteString(m.row(i == m.selected, label) + "\n")
	}
	b.WriteString(mutedStyle.Render("\n  scope: "+toolScopeForAction(m)) + "\n")
	b.WriteString(m.row(m.selected == len(m.tools), "Voltar") + "\n")
	return b.String()
}

func (m tuiModel) viewCreate(l layout) string {
	var body strings.Builder
	body.WriteString("\n")
	body.WriteString(" Profile name\n\n")
	body.WriteString(" > " + m.input.View() + "\n\n")
	if m.createStep >= 1 {
		body.WriteString(" Linked agents\n\n")
		for i, tool := range launch.Tools {
			mark := "[ ]"
			if m.createTools[tool] {
				mark = "[✓]"
			}
			line := fmt.Sprintf(" %s %s", mark, toolDisplayName(tool))
			if i == m.createCursor {
				line = selectedStyle.Render(symSelect+" "+line)
			} else {
				line = "  " + line
			}
			body.WriteString(line + "\n")
		}
		body.WriteString("\n")
	} else {
		body.WriteString("\n")
	}
	return framedBlock("CREATE PROFILE", body.String(), l.contentWidth()) + "\n\n" +
		titleStyle.Render("Cadastrar novo perfil") + "\n"
}

func (m tuiModel) viewRename(l layout) string {
	body := "\n Rename profile\n\n > " + m.input.View() + "\n\n"
	return framedBlock("RENAME PROFILE", body, l.contentWidth())
}

func (m tuiModel) viewConfirm(l layout) string {
	body := fmt.Sprintf("\n Delete profile %q?\n\n Stored credentials associated with this profile\n may be removed.\n\n          [y] Delete                 [n] Cancel\n\n", m.confirmName)
	return framedBlock("DELETE PROFILE", body, l.contentWidth())
}

func (m tuiModel) viewSwitchResult(l layout) string {
	from := m.switchFrom
	to := m.switchTo
	arrow := " ───────────────────────────────▶ "
	if l.mode != layoutFull {
		arrow = " ──▶ "
	}
	var body strings.Builder
	body.WriteString("\n")
	body.WriteString("   " + from + arrow + to + "\n\n")
	ok := m.switchOK
	for _, line := range m.switchLines {
		label := toolDisplayName(line.tool)
		var status string
		switch line.status {
		case "switched":
			status = greenStyle.Render(symOK + " switched")
		case "failed":
			status = redStyle.Render(symFail + " failed")
			ok = false
		default:
			status = mutedStyle.Render(symOff + " not linked")
		}
		row := padRight(label, 40) + status
		if l.mode != layoutFull {
			row = padRight(truncate(label, 14), 16) + status
		}
		body.WriteString("   " + row + "\n")
	}
	body.WriteString("\n")
	if ok {
		body.WriteString("   " + greenStyle.Render("Identity switched successfully.") + "\n")
	} else {
		body.WriteString("   " + yellowStyle.Render("Identity switch completed with errors.") + "\n")
	}
	body.WriteString("\n")
	return framedBlock("SWITCH IDENTITY", body.String(), l.contentWidth())
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
