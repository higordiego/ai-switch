package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/higordiego/ai-switch/internal/launch"
	"github.com/higordiego/ai-switch/internal/profile"
)

const Version = "0.1.3"

const help = `ai-switch - contas independentes por terminal (macOS/Linux)

  ai-switch create PERFIL [PERFIL...]
  ai-switch rename PERFIL NOVO-PERFIL
  ai-switch list [--json]
  ai-switch login TOOL [PERFIL] [-- ARGUMENTOS]
  ai-switch logout TOOL [PERFIL]
  ai-switch unlink TOOL [PERFIL]
  ai-switch status TOOL [PERFIL] [-- ARGUMENTOS]
  ai-switch run TOOL [PERFIL] [-- ARGUMENTOS]
  ai-switch current
  ai-switch doctor [PERFIL]
  ai-switch shell-init [zsh|bash]
  ai-switch env PERFIL

TOOL: claude, codex, cursor (Cursor Agent CLI).
Sem PERFIL, usa AISWITCH_PROFILE do terminal; nao existe conta padrao global.

Para habilitar use/switch e os comandos nativos neste terminal:
  eval "$(ai-switch shell-init zsh)"
  ai-switch use trabalho
  claude
  codex
  cursor-agent
  ai-switch deactivate

Em outro terminal, escolha outro perfil. Processos abertos mantem sua conta.
AISWITCH_ROOT altera o armazenamento (padrao: ~/.aiswitch).
`

type App struct {
	Out, Err   io.Writer
	Environ    []string
	Executable string
	Execute    func(launch.Plan) error
	Resolve    func(string) (string, error)
}

func (a App) getenv(key string) string {
	value := ""
	for _, e := range a.Environ {
		if k, v, ok := strings.Cut(e, "="); ok && k == key {
			value = v
		}
	}
	return value
}

func (a App) Run(args []string) error {
	if len(args) == 0 {
		return a.Interactive()
	}
	if len(args) == 1 && (args[0] == "--help" || args[0] == "help" || args[0] == "-h") {
		_, err := fmt.Fprint(a.Out, help)
		return err
	}
	command, rest := args[0], args[1:]
	if command == "version" || command == "--version" {
		if len(rest) > 0 {
			return errors.New("uso: ai-switch version")
		}
		fmt.Fprintln(a.Out, Version)
		return nil
	}
	if command == "shell-init" {
		if len(rest) > 1 || (len(rest) == 1 && rest[0] != "bash" && rest[0] != "zsh") {
			return errors.New("uso: ai-switch shell-init [bash|zsh]")
		}
		shellInit(a.Out, a.Executable)
		return nil
	}
	if command == "use" || command == "switch" || command == "deactivate" {
		return errors.New("ative a integracao neste terminal: eval \"$(ai-switch shell-init zsh)\"; depois use ai-switch use PERFIL")
	}
	s, err := profile.New(a.getenv("AISWITCH_ROOT"))
	if err != nil {
		return err
	}
	switch command {
	case "create":
		if len(rest) == 0 {
			return errors.New("uso: ai-switch create PERFIL [PERFIL...]")
		}
		for _, name := range rest {
			if err := profile.ValidateName(name); err != nil {
				return err
			}
		}
		for _, name := range rest {
			p, err := s.Create(name)
			if err != nil {
				return err
			}
			fmt.Fprintf(a.Out, "Criado: %s (%s)\n", p.Name, p.Path)
		}
		return nil
	case "rename":
		if len(rest) != 2 {
			return errors.New("uso: ai-switch rename PERFIL NOVO-PERFIL")
		}
		p, err := s.Rename(rest[0], rest[1])
		if err != nil {
			return err
		}
		fmt.Fprintf(a.Out, "Renomeado: %s\n", p.Name)
		return nil
	case "list":
		if len(rest) > 1 || (len(rest) == 1 && rest[0] != "--json") {
			return errors.New("uso: ai-switch list [--json]")
		}
		ps, err := s.List()
		if err != nil {
			return err
		}
		if len(rest) == 1 {
			return json.NewEncoder(a.Out).Encode(ps)
		}
		for _, p := range ps {
			marker := " "
			if p.Name == a.getenv("AISWITCH_PROFILE") {
				marker = "*"
			}
			fmt.Fprintf(a.Out, "%s %s\n", marker, p.Name)
		}
		if len(ps) == 0 {
			fmt.Fprintln(a.Out, "Nenhum perfil. Use ai-switch create trabalho pessoal")
		}
		return nil
	case "current":
		if len(rest) != 0 {
			return errors.New("uso: ai-switch current")
		}
		name := a.getenv("AISWITCH_PROFILE")
		if name == "" {
			return errors.New("nenhum perfil selecionado neste terminal")
		}
		if _, err := s.Get(name); err != nil {
			return err
		}
		fmt.Fprintln(a.Out, name)
		return nil
	case "env":
		if len(rest) != 1 {
			return errors.New("uso: ai-switch env PERFIL")
		}
		p, err := s.Get(rest[0])
		if err != nil {
			return err
		}
		fmt.Fprintf(a.Out, "export AISWITCH_ROOT=%s\nexport AISWITCH_PROFILE=%s\n", quote(s.Root), quote(p.Name))
		return nil
	case "doctor":
		if len(rest) > 1 {
			return errors.New("uso: ai-switch doctor [PERFIL]")
		}
		name := a.getenv("AISWITCH_PROFILE")
		if len(rest) == 1 {
			name = rest[0]
		}
		if name != "" {
			p, err := s.Get(name)
			if err != nil {
				return err
			}
			fmt.Fprintf(a.Out, "Perfil: %s\nDiretorio: %s\n", p.Name, p.Path)
		}
		var missing bool
		resolve := a.Resolve
		if resolve == nil {
			resolve = launch.Resolve
		}
		for _, program := range []string{"claude", "codex", "cursor-agent"} {
			path, err := resolve(program)
			if err != nil {
				fmt.Fprintf(a.Out, "FALTA %s\n", program)
				missing = true
			} else {
				fmt.Fprintf(a.Out, "OK %s: %s\n", program, path)
			}
		}
		_, removed := launch.Environment(a.Environ, s.Root, profile.Profile{})
		if len(removed) > 0 {
			fmt.Fprintf(a.Out, "Ignoradas no processo filho: %s\n", strings.Join(removed, ", "))
		}
		fmt.Fprintln(a.Out, "Login: consulte ai-switch status TOOL PERFIL. Cursor aqui e Agent CLI, nao o editor.")
		if missing {
			return errors.New("instale as ferramentas ausentes e execute doctor novamente")
		}
		return nil
	case "run", "login", "logout", "unlink", "status":
		tool, name, extra, err := parseLaunch(rest, a.getenv("AISWITCH_PROFILE"))
		if err != nil {
			return err
		}
		p, err := s.Get(name)
		if err != nil {
			return err
		}
		operation := command
		if operation == "unlink" {
			operation = "logout"
		}
		plan, err := launch.Build(s.Root, p, tool, operation, extra, a.Environ)
		if err != nil {
			return err
		}
		if len(plan.Removed) > 0 {
			fmt.Fprintf(a.Err, "ai-switch: ignorando autenticacao herdada: %s\n", strings.Join(plan.Removed, ", "))
		}
		fmt.Fprintf(a.Err, "ai-switch: %s / %s\n", p.Name, tool)
		return a.Execute(plan)
	default:
		return fmt.Errorf("comando %q desconhecido; use ai-switch --help", command)
	}
}

func parseLaunch(args []string, selected string) (tool, name string, extra []string, err error) {
	if len(args) == 0 {
		err = errors.New("informe TOOL e PERFIL; use ai-switch --help")
		return
	}
	tool, err = launch.Normalize(args[0])
	if err != nil {
		return
	}
	name = selected
	args = args[1:]
	if len(args) > 0 && args[0] != "--" && !strings.HasPrefix(args[0], "-") {
		name = args[0]
		args = args[1:]
	}
	if name == "" {
		err = errors.New("informe PERFIL ou selecione com ai-switch use PERFIL")
		return
	}
	if len(args) > 0 {
		if args[0] != "--" {
			err = errors.New("separe os argumentos da ferramenta com --")
			return
		}
		extra = append([]string(nil), args[1:]...)
	}
	return
}

func Main() int {
	exe, err := os.Executable()
	if err == nil {
		err = (App{Out: os.Stdout, Err: os.Stderr, Environ: os.Environ(), Executable: exe, Execute: launch.Execute, Resolve: launch.Resolve}).Run(os.Args[1:])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "ai-switch:", err)
		return 1
	}
	return 0
}
