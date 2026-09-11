package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/higordiego/ai-swtich/internal/launch"
	"github.com/higordiego/ai-swtich/internal/profile"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type tuiScreen int

const (
	homeScreen tuiScreen = iota
	profileScreen
	toolScreen
	createScreen
	renameScreen
	confirmScreen
	switchResultScreen
)

type flashKind string

const (
	flashInfo    flashKind = "info"
	flashSuccess flashKind = "success"
	flashWarning flashKind = "warning"
	flashError   flashKind = "error"
	flashLoading flashKind = "loading"
)

type switchLine struct {
	tool   string
	status string // switched | not_linked | failed
	detail string
}

type tuiModel struct {
	store        profile.Store
	root         string
	environ      []string
	profiles     []profile.Profile
	selected     int
	profileIndex int
	tool         int
	tools        []string
	action       string
	screen       tuiScreen
	input        textinput.Model
	message      string
	flash        flashKind
	err          error
	quit         bool
	command      func(launch.Plan) (*exec.Cmd, error)
	linked       map[string]string
	activeName   string
	width        int
	height       int
	createStep   int // 0=name, 1=agents
	createTools  map[string]bool
	createCursor int
	pendingLogin []string
	confirmKind  string
	confirmName  string
	switchFrom   string
	switchTo     string
	switchLines  []switchLine
	switchOK     bool
}

type tuiDoneMsg struct{ err error }

func newTUI(store profile.Store, root string, environ []string) tuiModel {
	in := textinput.New()
	in.Placeholder = "ex.: trabalho"
	in.CharLimit = 48
	in.Width = 32
	ps, _ := store.List()
	active := ""
	for _, item := range environ {
		if key, val, ok := strings.Cut(item, "="); ok && key == "AISWITCH_PROFILE" {
			active = val
			break
		}
	}
	m := tuiModel{
		store:       store,
		root:        root,
		environ:     environ,
		profiles:    ps,
		input:       in,
		command:     launch.Command,
		linked:      map[string]string{},
		activeName:  active,
		width:       80,
		height:      24,
		createTools: map[string]bool{},
		flash:       flashInfo,
	}
	if len(ps) > 0 {
		m.loadLinks(ps[0])
	}
	return m
}

func (m *tuiModel) loadLinks(p profile.Profile) {
	m.linked = map[string]string{}
	for _, tool := range launch.Tools {
		if profileToolLinked(p, tool) {
			m.linked[tool] = "Vinculado"
		}
	}
}

func profileToolLinked(p profile.Profile, tool string) bool {
	if linked, declared := p.Tools[tool]; declared {
		return linked
	}
	return hasStoredCredentials(p, tool)
}

// hasStoredCredentials detects an existing native login without reading or
// exposing secret material — only structural presence of auth objects.
func hasStoredCredentials(p profile.Profile, tool string) bool {
	var data struct {
		OAuthAccount json.RawMessage            `json:"oauthAccount"`
		Tokens       map[string]json.RawMessage `json:"tokens"`
		AuthInfo     map[string]json.RawMessage `json:"authInfo"`
	}
	var path string
	switch tool {
	case "claude":
		path = filepath.Join(p.Path, "claude", ".claude.json")
	case "codex":
		path = filepath.Join(p.Path, "codex", "auth.json")
	case "cursor":
		path = filepath.Join(p.Path, "cursor", "cli-config.json")
	default:
		return false
	}
	content, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(content, &data) != nil {
		return false
	}
	switch tool {
	case "claude":
		return len(data.OAuthAccount) > 0 && string(data.OAuthAccount) != "null" && string(data.OAuthAccount) != "{}"
	case "codex":
		return len(data.Tokens) > 0
	case "cursor":
		return len(data.AuthInfo) > 0
	default:
		return false
	}
}

func (m tuiModel) toolChoices() []string {
	if m.action == "run" || m.action == "logout" {
		result := []string{}
		for _, tool := range launch.Tools {
			if m.linked[tool] == "Vinculado" {
				result = append(result, tool)
			}
		}
		return result
	}
	return append([]string(nil), launch.Tools...)
}

func (m tuiModel) Init() tea.Cmd { return nil }

func (m tuiModel) selectedProfile() profile.Profile {
	if m.profileIndex < 0 || m.profileIndex >= len(m.profiles) {
		return profile.Profile{}
	}
	return m.profiles[m.profileIndex]
}

func (m tuiModel) homeProfile() (profile.Profile, bool) {
	if m.selected < 0 || m.selected >= len(m.profiles) {
		return profile.Profile{}, false
	}
	return m.profiles[m.selected], true
}

func (m tuiModel) isLinked(tool string) bool {
	return m.linked[tool] == "Vinculado"
}

func (m tuiModel) layout() layout {
	return newLayout(m.width, m.height)
}

func (m *tuiModel) setFlash(kind flashKind, text string) {
	m.flash = kind
	m.message = text
}

func (m *tuiModel) refreshProfiles() {
	m.profiles, _ = m.store.List()
}

func (m *tuiModel) focusProfile(name string) {
	for i := range m.profiles {
		if m.profiles[i].Name == name {
			m.profileIndex = i
			m.selected = 0
			m.loadLinks(m.profiles[i])
			return
		}
	}
}

func linkedToolCount(p profile.Profile) int {
	count := 0
	for _, tool := range launch.Tools {
		if profileToolLinked(p, tool) {
			count++
		}
	}
	return count
}

func toolDisplayName(tool string) string {
	return map[string]string{"claude": "Claude Code", "codex": "Codex", "cursor": "Cursor Agent"}[tool]
}

func toolDescription(tool string) string {
	return map[string]string{
		"claude": "Anthropic coding agent",
		"codex":  "OpenAI coding agent",
		"cursor": "Cursor terminal agent",
	}[tool]
}

func toolScope(tool string) string {
	return map[string]string{
		"claude": "CLAUDE_CONFIG_DIR",
		"codex":  "CODEX_HOME",
		"cursor": "CURSOR_CONFIG_DIR",
	}[tool]
}

func toolScopeForAction(m tuiModel) string {
	if m.selected >= 0 && m.selected < len(m.tools) {
		return toolScope(m.tools[m.selected])
	}
	return "isolated profile dirs"
}

func statusChipText(m tuiModel, tool string) string {
	if m.linked[tool] == "Vinculado" {
		return "● LINKED"
	}
	return "○ NOT LINKED"
}

func (m tuiModel) statusCell(tool string) string {
	if m.isLinked(tool) {
		return greenStyle.Render(padRight(statusChipText(m, tool), 13))
	}
	return mutedStyle.Render(padRight(statusChipText(m, tool), 13))
}

func toolTableRow(m tuiModel, tool string) string {
	return fmt.Sprintf("  %-15s %s %s", toolDisplayName(tool), m.statusCell(tool), toolScope(tool))
}

func (m tuiModel) row(selected bool, text string) string {
	if selected {
		return selectedStyle.Render(symSelect + " " + text)
	}
	return "  " + text
}

func RunTUI(store profile.Store, root string, environ []string, in io.Reader, out io.Writer) error {
	program := tea.NewProgram(newTUI(store, root, environ), tea.WithInput(in), tea.WithOutput(out), tea.WithAltScreen())
	_, err := program.Run()
	return err
}

func (a App) Interactive() error {
	store, err := profile.New(a.getenv("AISWITCH_ROOT"))
	if err != nil {
		return err
	}
	if a.Environ == nil {
		a.Environ = os.Environ()
	}
	return RunTUI(store, store.Root, a.Environ, os.Stdin, os.Stdout)
}
