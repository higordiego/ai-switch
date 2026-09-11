package cli

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"aiswitch/internal/launch"
	"aiswitch/internal/profile"

	tea "github.com/charmbracelet/bubbletea"
)

func testTUI(t *testing.T) tuiModel {
	t.Helper()
	store, err := profile.New(filepath.Join(t.TempDir(), "store"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.Create("work"); err != nil {
		t.Fatal(err)
	}
	for _, tool := range []string{"claude", "codex", "cursor"} {
		if _, err = store.SetToolLinked("work", tool, true); err != nil {
			t.Fatal(err)
		}
	}
	m := newTUI(store, store.Root, []string{"PATH=/bin", "AISWITCH_PROFILE=work"})
	if m.Init() != nil {
		t.Fatal("unexpected init command")
	}
	return m
}

func pressTUI(m tuiModel, key string) tuiModel {
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
	return updated.(tuiModel)
}

func TestTUIHomeProfileAndToolNavigation(t *testing.T) {
	m := testTUI(t)
	if m.selectedProfile().Name != "work" || m.itemCount() != 2 {
		t.Fatalf("home model not initialized: profile=%q count=%d", m.selectedProfile().Name, m.itemCount())
	}
	view := m.View()
	if !strings.Contains(view, "PROFILES") || !strings.Contains(view, "Create profile") {
		t.Fatal("home view incomplete")
	}
	if !strings.Contains(view, "Claude Code") || !strings.Contains(view, "ACTIVE") {
		t.Fatal("home missing agents or active badge")
	}
	m = pressTUI(m, "j")
	if m.selected != 1 {
		t.Fatal("down did not move")
	}
	m = pressTUI(m, "k")
	if m.selected != 0 {
		t.Fatal("up did not move")
	}
	var cmd tea.Cmd
	var model tea.Model
	model, cmd = m.choose()
	_ = cmd
	m = model.(tuiModel)
	if m.screen != profileScreen || m.selected != 0 {
		t.Fatal("profile screen not opened")
	}
	if !strings.Contains(m.View(), "LINKED AGENTS") || !strings.Contains(m.View(), "Link new agent") {
		t.Fatal("profile view incomplete")
	}
	// Jump to status via shortcut.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	m = updated.(tuiModel)
	if m.screen != toolScreen || m.action != "status" {
		t.Fatal("tool status screen not opened")
	}
	if !strings.Contains(m.View(), "Status qual ferramenta") {
		t.Fatal("tool view incomplete")
	}
	m = pressTUI(m, "j")
	m = pressTUI(m, "j")
	m = pressTUI(m, "j")
	if m.selected != 3 {
		t.Fatal("tool navigation failed")
	}
	model, cmd = m.choose()
	_ = cmd
	m = model.(tuiModel)
	if m.screen != profileScreen {
		t.Fatal("tool back failed")
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(tuiModel)
	if m.screen != homeScreen {
		t.Fatal("profile back failed")
	}
	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	_ = updated
	if cmd == nil {
		t.Fatal("home quit failed")
	}
}

func TestTUIOpenOnlyLinkedTools(t *testing.T) {
	store, err := profile.New(filepath.Join(t.TempDir(), "store"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Create("empty"); err != nil {
		t.Fatal(err)
	}
	m := newTUI(store, store.Root, []string{"PATH=/bin"})
	updated, _ := m.choose()
	m = updated.(tuiModel)
	updated, _ = m.choose()
	m = updated.(tuiModel)
	if m.screen != profileScreen || !strings.Contains(m.message, "not linked") {
		t.Fatalf("unlinked tool appeared in open menu: screen=%d message=%s tools=%v linked=%v", m.screen, m.message, m.tools, m.linked)
	}
	m.action = "run"
	m.tools = m.toolChoices()
	if len(m.tools) != 0 {
		t.Fatal("open menu exposed unlinked tool")
	}
}

func TestTUILoadLinksDetectsStoredCredentials(t *testing.T) {
	store, err := profile.New(filepath.Join(t.TempDir(), "store"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := store.Create("legacy")
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		filepath.Join(p.Path, "claude", ".claude.json"):    `{"oauthAccount":{"accountUuid":"account"}}`,
		filepath.Join(p.Path, "codex", "auth.json"):        `{"tokens":{"access_token":"present"}}`,
		filepath.Join(p.Path, "cursor", "cli-config.json"): `{"authInfo":{"userId":"account"}}`,
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	m := newTUI(store, store.Root, []string{"PATH=/bin"})
	for _, tool := range launch.Tools {
		if m.linked[tool] != "Vinculado" {
			t.Fatalf("stored %s credentials were not detected: %#v", tool, m.linked)
		}
	}
	if _, err := store.SetToolLinked("legacy", "codex", false); err != nil {
		t.Fatal(err)
	}
	p, err = store.Get("legacy")
	if err != nil {
		t.Fatal(err)
	}
	m.loadLinks(p)
	if _, ok := m.linked["codex"]; ok {
		t.Fatal("explicit unlink was overridden by stored credentials")
	}
}

func TestTUIUnlinkAndRenameActions(t *testing.T) {
	m := testTUI(t)
	var model tea.Model
	var cmd tea.Cmd
	m.selected = 0
	model, cmd = m.choose()
	_ = cmd
	m = model.(tuiModel)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	m = updated.(tuiModel)
	if m.screen != toolScreen || m.action != "logout" {
		t.Fatal("unlink action not opened")
	}
	m.selected = len(m.tools)
	model, cmd = m.choose()
	_ = cmd
	m = model.(tuiModel)
	if m.screen != profileScreen {
		t.Fatal("unlink back failed")
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = updated.(tuiModel)
	if m.screen != renameScreen || m.action != "rename" {
		t.Fatal("rename action not opened")
	}
	m.input.SetValue("personal")
	model, cmd = m.updateRename(tea.KeyMsg{Type: tea.KeyEnter})
	_ = cmd
	m = model.(tuiModel)
	if m.screen != profileScreen || m.selectedProfile().Name != "personal" {
		t.Fatalf("rename action failed: screen=%d selected=%d action=%s input=%q profiles=%+v message=%s", m.screen, m.selected, m.action, m.input.Value(), m.profiles, m.message)
	}
	if !strings.Contains(m.View(), "Perfil renomeado") {
		t.Fatal("rename confirmation missing")
	}
}

func TestTUIKeyBranchesAndWrapping(t *testing.T) {
	m := testTUI(t)
	m.selected = 0
	m = pressTUI(m, "k")
	if m.selected != 1 {
		t.Fatalf("up did not wrap: %d", m.selected)
	}
	m = pressTUI(m, "j")
	if m.selected != 0 {
		t.Fatalf("down did not wrap: %d", m.selected)
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(tuiModel)
	if m.screen != profileScreen {
		t.Fatal("enter was not handled by update")
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(tuiModel)
	if m.screen != homeScreen {
		t.Fatal("escape failed")
	}
	m.screen = createScreen
	m.createStep = 0
	m.input.Focus()
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = updated.(tuiModel)
	if m.input.Value() != "x" {
		t.Fatal("create input did not receive key")
	}
	m.screen = homeScreen
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	_ = updated
	if cmd == nil {
		t.Fatal("home escape did not quit")
	}
	m.screen = createScreen
	m.move(1)
	if m.itemCount() != 0 {
		t.Fatal("create screen should not have a list")
	}
}

func TestTUICreateAndError(t *testing.T) {
	m := testTUI(t)
	m.selected = 1
	var cmd tea.Cmd
	var model tea.Model
	model, cmd = m.choose()
	_ = cmd
	m = model.(tuiModel)
	if m.screen != createScreen {
		t.Fatal("create screen not opened")
	}
	if !strings.Contains(m.View(), "Cadastrar novo perfil") && !strings.Contains(m.View(), "CREATE PROFILE") {
		t.Fatal("create view incomplete")
	}
	model, cmd = m.updateCreate(tea.KeyMsg{Type: tea.KeyEnter})
	_ = cmd
	m = model.(tuiModel)
	if m.message == "" || m.screen != createScreen {
		t.Fatal("empty name accepted")
	}
	for _, r := range "personal" {
		model, cmd = m.updateCreate(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		_ = cmd
		m = model.(tuiModel)
	}
	model, cmd = m.updateCreate(tea.KeyMsg{Type: tea.KeyEnter})
	_ = cmd
	m = model.(tuiModel)
	if m.createStep != 1 {
		t.Fatal("create did not advance to agent selection")
	}
	model, cmd = m.updateCreate(tea.KeyMsg{Type: tea.KeyEnter})
	_ = cmd
	m = model.(tuiModel)
	if m.screen != profileScreen || len(m.profiles) != 2 {
		t.Fatalf("profile not created: screen=%d profiles=%d message=%s", m.screen, len(m.profiles), m.message)
	}
	m.screen = createScreen
	m.createStep = 0
	m.action = "create"
	m.input.SetValue("personal")
	model, cmd = m.updateCreate(tea.KeyMsg{Type: tea.KeyEnter})
	_ = cmd
	m = model.(tuiModel)
	model, cmd = m.updateCreate(tea.KeyMsg{Type: tea.KeyEnter})
	_ = cmd
	m = model.(tuiModel)
	if m.message == "" {
		t.Fatal("duplicate not reported")
	}
	model, cmd = m.updateCreate(tea.KeyMsg{Type: tea.KeyEsc})
	_ = cmd
	m = model.(tuiModel)
	if m.screen != homeScreen {
		t.Fatal("create cancel failed")
	}
}

func TestTUILaunchActionsAndMessages(t *testing.T) {
	m := testTUI(t)
	m.screen, m.action, m.tools, m.tool, m.selected = toolScreen, "run", []string{"claude"}, 0, 0
	called := 0
	m.command = func(plan launch.Plan) (*exec.Cmd, error) { called++; return exec.Command("true"), nil }
	_, cmd := m.choose()
	if cmd == nil || called != 1 {
		t.Fatal("tool command not prepared")
	}
	if msg := cmd(); msg == nil {
		t.Fatal("tool command callback missing")
	}
	m.action = "login"
	m.selected = 0
	m.tools = []string{"claude", "codex", "cursor"}
	_, cmd = m.choose()
	if cmd == nil {
		t.Fatal("login command not prepared")
	}
	m.action = "status"
	m.selected = 1
	_, cmd = m.choose()
	if cmd == nil {
		t.Fatal("status command not prepared")
	}
	m.tools, m.tool = []string{"claude"}, 0
	m.err = errors.New("old")
	model, _ := m.Update(tuiDoneMsg{err: errors.New("failed")})
	m = model.(tuiModel)
	if !strings.Contains(m.message, "nao concluida") {
		t.Fatal("error message missing")
	}
	m.action, m.tool = "login", 0
	model, _ = m.Update(tuiDoneMsg{err: errors.New("login failed")})
	m = model.(tuiModel)
	if m.linked["claude"] != "Nao vinculado" {
		t.Fatal("failed login status missing")
	}
	m.action = "logout"
	model, _ = m.Update(tuiDoneMsg{err: errors.New("logout failed")})
	m = model.(tuiModel)
	if m.linked["claude"] != "Vinculado" {
		t.Fatal("failed logout status missing")
	}
	model, _ = m.Update(tuiDoneMsg{})
	m = model.(tuiModel)
	if m.message == "" {
		t.Fatal("done message missing")
	}
	m.action, m.tool, m.tools = "login", 0, []string{"claude"}
	model, _ = m.Update(tuiDoneMsg{})
	m = model.(tuiModel)
	if m.linked["claude"] != "Vinculado" || !strings.Contains(m.message, "vinculado") {
		t.Fatal("link confirmation missing")
	}
	m.action, m.tool, m.tools = "logout", 0, []string{"claude"}
	model, _ = m.Update(tuiDoneMsg{})
	m = model.(tuiModel)
	if m.linked["claude"] == "Vinculado" || !strings.Contains(m.message, "desvinculado") {
		t.Fatalf("unlink confirmation missing: linked=%q message=%q", m.linked["claude"], m.message)
	}
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = model.(tuiModel)
	if m.screen != homeScreen {
		t.Fatal("escape did not return home")
	}
}

func TestTUIActivateAndDelete(t *testing.T) {
	m := testTUI(t)
	if _, err := m.store.Create("other"); err != nil {
		t.Fatal(err)
	}
	m.refreshProfiles()
	m.selected = 0 // other (sorted)
	if m.profiles[0].Name != "other" {
		t.Fatalf("unexpected order: %+v", m.profiles)
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(tuiModel)
	if m.screen != switchResultScreen || m.activeName != "other" {
		t.Fatalf("activate failed: screen=%d active=%s", m.screen, m.activeName)
	}
	if !strings.Contains(m.View(), "SWITCH IDENTITY") || !strings.Contains(m.View(), "not linked") {
		t.Fatal("switch result incomplete")
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(tuiModel)
	if m.screen != homeScreen {
		t.Fatal("switch result did not return home")
	}
	m.selected = 0
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = updated.(tuiModel)
	if m.screen != confirmScreen {
		t.Fatal("delete confirm not opened")
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = updated.(tuiModel)
	if m.screen != homeScreen || len(m.profiles) != 1 || m.profiles[0].Name != "work" {
		t.Fatalf("delete failed: profiles=%+v", m.profiles)
	}
	if m.activeName != "" {
		t.Fatal("active profile not cleared after delete")
	}
}

func TestTUIProgramQuit(t *testing.T) {
	m := testTUI(t)
	var output bytes.Buffer
	if err := RunTUI(m.store, m.root, m.environ, strings.NewReader("q"), &output); err != nil {
		t.Fatal(err)
	}
	if output.Len() == 0 {
		t.Fatal("TUI produced no output")
	}
}

func TestInteractiveRejectsSystemRoot(t *testing.T) {
	a := App{Environ: []string{"AISWITCH_ROOT=/"}}
	if err := a.Interactive(); err == nil {
		t.Fatal("interactive accepted system root")
	}
}

func TestTUIResponsiveHeader(t *testing.T) {
	m := testTUI(t)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 50, Height: 20})
	m = updated.(tuiModel)
	if m.layout().mode != layoutMinimal {
		t.Fatal("expected minimal layout")
	}
	if !strings.Contains(m.View(), "AISWITCH") {
		t.Fatal("header missing on narrow terminal")
	}
	updated, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = updated.(tuiModel)
	if m.layout().mode != layoutFull {
		t.Fatal("expected full layout")
	}
}
