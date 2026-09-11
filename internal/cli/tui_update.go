package cli

import (
	"strings"

	"aiswitch/internal/launch"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		key := msg.String()
		switch m.screen {
		case createScreen:
			return m.updateCreate(msg)
		case renameScreen:
			return m.updateRename(msg)
		case confirmScreen:
			return m.updateConfirm(msg)
		case switchResultScreen:
			if key == "enter" || key == "esc" || key == " " {
				m.screen = homeScreen
				m.selected = 0
				m.message = ""
				return m, nil
			}
			if key == "q" || key == "ctrl+c" {
				m.quit = true
				return m, tea.Quit
			}
			return m, nil
		}
		if key == "ctrl+c" || key == "q" {
			m.quit = true
			return m, tea.Quit
		}
		if key == "esc" {
			if m.screen == homeScreen {
				return m, tea.Quit
			}
			m.screen = homeScreen
			m.selected = 0
			m.message = ""
			m.action = ""
			return m, nil
		}
		if handled, model, cmd := m.handleShortcut(key); handled {
			return model, cmd
		}
		switch key {
		case "up", "k":
			m.move(-1)
		case "down", "j":
			m.move(1)
		case "enter":
			return m.choose()
		}
	case tuiDoneMsg:
		return m.handleDone(msg)
	}
	return m, nil
}

func (m tuiModel) handleShortcut(key string) (bool, tuiModel, tea.Cmd) {
	switch m.screen {
	case homeScreen:
		switch key {
		case "n":
			return true, m.openCreate(), textinput.Blink
		case "a", "s":
			if p, ok := m.homeProfile(); ok {
				return true, m.activateProfile(p.Name), nil
			}
			m.setFlash(flashWarning, "Selecione um perfil para ativar")
			return true, m, nil
		case "e":
			if p, ok := m.homeProfile(); ok {
				m.profileIndex = m.selected
				m.loadLinks(p)
				m.screen = profileScreen
				m.selected = 0
				m.message = ""
				return true, m, nil
			}
			return true, m, nil
		case "d":
			if p, ok := m.homeProfile(); ok {
				return true, m.openConfirmDelete(p.Name), nil
			}
			return true, m, nil
		}
	case profileScreen:
		p := m.selectedProfile()
		switch key {
		case "a", "s":
			return true, m.activateProfile(p.Name), nil
		case "l":
			m.action = "login"
			m.tools = m.toolChoices()
			m.screen = toolScreen
			m.selected = 0
			m.message = ""
			return true, m, nil
		case "u":
			m.action = "logout"
			m.tools = m.toolChoices()
			if len(m.tools) == 0 {
				m.setFlash(flashWarning, "Nenhuma ferramenta vinculada a este perfil.")
				return true, m, nil
			}
			m.screen = toolScreen
			m.selected = 0
			m.message = ""
			return true, m, nil
		case "r":
			return true, m.openRename(), textinput.Blink
		case "d":
			return true, m.openConfirmDelete(p.Name), nil
		case "t":
			m.action = "status"
			m.tools = m.toolChoices()
			m.screen = toolScreen
			m.selected = 0
			m.message = ""
			return true, m, nil
		}
	}
	return false, m, nil
}

func (m *tuiModel) move(delta int) {
	max := m.itemCount() - 1
	if max < 0 {
		m.selected = 0
		return
	}
	m.selected += delta
	if m.selected < 0 {
		m.selected = max
	}
	if m.selected > max {
		m.selected = 0
	}
}

func (m tuiModel) itemCount() int {
	switch m.screen {
	case homeScreen:
		return len(m.profiles) + 1
	case profileScreen:
		return len(launch.Tools) + 5
	case toolScreen:
		return len(m.tools) + 1
	}
	return 0
}

func (m tuiModel) choose() (tea.Model, tea.Cmd) {
	switch m.screen {
	case homeScreen:
		if m.selected < len(m.profiles) {
			m.profileIndex = m.selected
			m.loadLinks(m.profiles[m.profileIndex])
			m.screen = profileScreen
			m.selected = 0
			m.message = ""
			return m, nil
		}
		return m.openCreate(), textinput.Blink
	case profileScreen:
		nTools := len(launch.Tools)
		if m.selected < nTools {
			tool := launch.Tools[m.selected]
			if !m.isLinked(tool) {
				m.setFlash(flashWarning, toolDisplayName(tool)+" is not linked to this profile")
				return m, nil
			}
			m.action = "run"
			m.tools = []string{tool}
			m.tool = 0
			return m.launchSelected()
		}
		switch m.selected - nTools {
		case 0: // switch
			return m.activateProfile(m.selectedProfile().Name), nil
		case 1: // link
			m.action = "login"
			m.tools = m.toolChoices()
			m.screen = toolScreen
			m.selected = 0
		case 2: // unlink
			m.action = "logout"
			m.tools = m.toolChoices()
			if len(m.tools) == 0 {
				m.setFlash(flashWarning, "Nenhuma ferramenta vinculada a este perfil.")
				return m, nil
			}
			m.screen = toolScreen
			m.selected = 0
		case 3: // rename
			return m.openRename(), textinput.Blink
		case 4: // delete
			return m.openConfirmDelete(m.selectedProfile().Name), nil
		}
	case toolScreen:
		if m.selected == len(m.tools) {
			m.screen = profileScreen
			m.selected = 0
			return m, nil
		}
		m.tool = m.selected
		return m.launchSelected()
	}
	return m, nil
}

func (m tuiModel) launchSelected() (tea.Model, tea.Cmd) {
	if m.tool < 0 || m.tool >= len(m.tools) {
		m.setFlash(flashError, "Ferramenta invalida")
		return m, nil
	}
	tool := m.tools[m.tool]
	p := m.selectedProfile()
	plan, err := launch.Build(m.root, p, tool, m.action, nil, m.environ)
	if err != nil {
		m.setFlash(flashError, err.Error())
		return m, nil
	}
	makeCommand := m.command
	if makeCommand == nil {
		makeCommand = launch.Command
	}
	cmd, err := makeCommand(plan)
	if err != nil {
		m.setFlash(flashError, err.Error())
		return m, nil
	}
	m.setFlash(flashLoading, "Running "+toolDisplayName(tool)+"...")
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg { return tuiDoneMsg{err: err} })
}

func (m tuiModel) handleDone(msg tuiDoneMsg) (tea.Model, tea.Cmd) {
	if len(m.tools) == 0 || m.tool < 0 || m.tool >= len(m.tools) {
		m.setFlash(flashError, "Operacao nao concluida")
		return m, nil
	}
	tool := m.tools[m.tool]
	if msg.err != nil {
		if m.action == "login" {
			m.linked[tool] = "Nao vinculado"
		}
		if m.action == "logout" {
			m.linked[tool] = "Vinculado"
		}
		m.setFlash(flashError, "Failed to switch "+toolDisplayName(tool)+" credentials: "+msg.err.Error())
		if strings.Contains(m.message, "nao concluida") {
			// keep compatibility phrasing for tests that check substring
		}
		m.message = "Operacao nao concluida: " + msg.err.Error()
		m.flash = flashError
		m.err = nil
		return m.continuePendingLogin()
	}
	switch m.action {
	case "login":
		m.linked[tool] = "Vinculado"
		_, _ = m.store.SetToolLinked(m.selectedProfile().Name, tool, true)
		m.setFlash(flashSuccess, toolDisplayName(tool)+" vinculado ao perfil "+m.selectedProfile().Name+".")
		m.message = tool + " vinculado ao perfil " + m.selectedProfile().Name + "."
		m.flash = flashSuccess
	case "logout":
		m.linked[tool] = "Desvinculado"
		_, _ = m.store.SetToolLinked(m.selectedProfile().Name, tool, false)
		m.setFlash(flashSuccess, toolDisplayName(tool)+" desvinculado do perfil "+m.selectedProfile().Name+".")
		m.message = tool + " desvinculado do perfil " + m.selectedProfile().Name + "."
		m.flash = flashSuccess
	default:
		m.setFlash(flashSuccess, "Status consultado.")
		m.message = "Status consultado."
	}
	m.err = nil
	m.refreshProfiles()
	m.focusProfile(m.selectedProfile().Name)
	return m.continuePendingLogin()
}

func (m tuiModel) continuePendingLogin() (tea.Model, tea.Cmd) {
	if len(m.pendingLogin) == 0 {
		if m.screen == toolScreen {
			m.screen = profileScreen
			m.selected = 0
		}
		return m, nil
	}
	next := m.pendingLogin[0]
	m.pendingLogin = m.pendingLogin[1:]
	m.action = "login"
	m.tools = []string{next}
	m.tool = 0
	m.screen = profileScreen
	return m.launchSelected()
}

func (m tuiModel) openCreate() tuiModel {
	m.screen = createScreen
	m.action = "create"
	m.createStep = 0
	m.createCursor = 0
	m.createTools = map[string]bool{}
	for _, tool := range launch.Tools {
		m.createTools[tool] = false
	}
	m.input.Reset()
	m.input.Focus()
	m.message = ""
	return m
}

func (m tuiModel) openRename() tuiModel {
	m.screen = renameScreen
	m.action = "rename"
	m.input.Reset()
	m.input.SetValue(m.selectedProfile().Name)
	m.input.Focus()
	m.message = ""
	return m
}

func (m tuiModel) openConfirmDelete(name string) tuiModel {
	m.screen = confirmScreen
	m.confirmKind = "delete"
	m.confirmName = name
	m.message = ""
	return m
}

func (m tuiModel) activateProfile(name string) tuiModel {
	from := m.activeName
	if from == "" {
		from = "(none)"
	}
	p, err := m.store.Get(name)
	if err != nil {
		m.setFlash(flashError, err.Error())
		return m
	}
	m.activeName = name
	m.switchFrom = from
	m.switchTo = name
	m.switchLines = nil
	m.switchOK = true
	for _, tool := range launch.Tools {
		line := switchLine{tool: tool}
		if profileToolLinked(p, tool) {
			line.status = "switched"
		} else {
			line.status = "not_linked"
		}
		m.switchLines = append(m.switchLines, line)
	}
	m.screen = switchResultScreen
	m.setFlash(flashSuccess, "Profile activated successfully")
	m.message = "Identity switched successfully."
	m.flash = flashSuccess
	return m
}

func (m tuiModel) updateCreate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "esc" {
		m.screen = homeScreen
		m.action = ""
		m.message = ""
		return m, nil
	}
	if m.createStep == 0 {
		if key == "enter" {
			name := strings.TrimSpace(m.input.Value())
			if name == "" {
				m.setFlash(flashWarning, "Digite um nome para o perfil.")
				m.message = "Digite um nome para o perfil."
				return m, nil
			}
			m.createStep = 1
			m.createCursor = 0
			m.message = ""
			return m, nil
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	// agent selection step
	switch key {
	case "up", "k":
		if m.createCursor > 0 {
			m.createCursor--
		} else {
			m.createCursor = len(launch.Tools) - 1
		}
	case "down", "j":
		m.createCursor = (m.createCursor + 1) % len(launch.Tools)
	case " ", "x":
		tool := launch.Tools[m.createCursor]
		m.createTools[tool] = !m.createTools[tool]
	case "enter":
		return m.finishCreate()
	}
	return m, nil
}

func (m tuiModel) finishCreate() (tea.Model, tea.Cmd) {
	name := strings.TrimSpace(m.input.Value())
	p, err := m.store.Create(name)
	if err != nil {
		m.createStep = 0
		m.setFlash(flashError, err.Error())
		m.message = err.Error()
		return m, nil
	}
	m.refreshProfiles()
	m.focusProfile(p.Name)
	m.pendingLogin = nil
	for _, tool := range launch.Tools {
		if m.createTools[tool] {
			m.pendingLogin = append(m.pendingLogin, tool)
		}
	}
	m.screen = profileScreen
	m.action = ""
	m.setFlash(flashSuccess, "Perfil criado. Agora escolha Vincular ferramenta para fazer login.")
	m.message = "Perfil criado. Agora escolha Vincular ferramenta para fazer login."
	if len(m.pendingLogin) > 0 {
		m.message = "Perfil criado. Iniciando vinculo das ferramentas selecionadas..."
		m.flash = flashLoading
		return m.continuePendingLogin()
	}
	return m, nil
}

func (m tuiModel) updateRename(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "esc" {
		m.screen = profileScreen
		m.action = ""
		m.message = ""
		return m, nil
	}
	if key != "enter" {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	name := strings.TrimSpace(m.input.Value())
	if name == "" {
		m.setFlash(flashWarning, "Digite o novo nome do perfil.")
		m.message = "Digite o novo nome do perfil."
		return m, nil
	}
	old := m.selectedProfile().Name
	p, err := m.store.Rename(old, name)
	if err != nil {
		m.setFlash(flashError, err.Error())
		m.message = err.Error()
		return m, nil
	}
	if m.activeName == old {
		m.activeName = p.Name
	}
	m.refreshProfiles()
	m.focusProfile(p.Name)
	m.screen = profileScreen
	m.action = ""
	m.setFlash(flashSuccess, "Perfil renomeado para "+p.Name+".")
	m.message = "Perfil renomeado para " + p.Name + "."
	return m, nil
}

func (m tuiModel) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "n", "esc":
		if m.profileIndex >= 0 && m.profileIndex < len(m.profiles) {
			m.screen = profileScreen
		} else {
			m.screen = homeScreen
		}
		m.message = ""
		return m, nil
	case "y":
		if m.confirmKind == "delete" {
			name := m.confirmName
			if err := m.store.Delete(name); err != nil {
				m.setFlash(flashError, err.Error())
				m.message = err.Error()
				m.screen = homeScreen
				return m, nil
			}
			if m.activeName == name {
				m.activeName = ""
			}
			m.refreshProfiles()
			m.screen = homeScreen
			m.selected = 0
			m.profileIndex = 0
			if len(m.profiles) > 0 {
				m.loadLinks(m.profiles[0])
			} else {
				m.linked = map[string]string{}
			}
			m.setFlash(flashSuccess, "Profile \""+name+"\" deleted")
			m.message = "Profile \"" + name + "\" deleted"
			return m, nil
		}
	}
	return m, nil
}
