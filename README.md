# aiswitch

```
╭─[ AISWITCH // IDENTITY CONTROL ]─────────────────────────────╮
│  ◈ AISWITCH                                       v0.1.0     │
│  isolated identities for coding agents                       │
╰──────────────────────────────────────────────────────────────╯
```

**AI Identity & Account Switcher** — CLI + TUI em Go para isolar contas de
coding agents por terminal.

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev/)
[![CI](https://github.com/higordiego/ai-swtich/actions/workflows/ci.yml/badge.svg)](https://github.com/higordiego/ai-swtich/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg?style=flat-square)](LICENSE)
[![Release](https://img.shields.io/github/v/release/higordiego/ai-swtich?style=flat-square&include_prereleases&sort=semver)](https://github.com/higordiego/ai-swtich/releases)

> Feito por **Higor Diego** · [github.com/higordiego/ai-swtich](https://github.com/higordiego/ai-swtich)

---

## Por que existe?

Claude Code, Codex CLI e Cursor Agent CLI autenticam em storage/env **global do
usuário**. No mesmo Mac isso mistura conta pessoal e corporativa, herda tokens
do shell pai e força logout/login toda hora.

**aiswitch** cria um profile isolado (dirs + env) e deixa a troca explícita —
sem ler, copiar ou imprimir secrets.

---

## O que existe hoje (v0.1.0)

| Área | Suporte real |
|------|----------------|
| Agents | `claude` (Claude Code), `codex` (Codex), `cursor` (Cursor Agent CLI) |
| Storage | `~/.aiswitch/profiles/<nome>/` com `claude/`, `anthropic/`, `codex/`, `cursor/` |
| CLI | `create`, `rename`, `list`, `current`, `env`, `doctor`, `login`, `logout`, `unlink`, `status`, `run`, `shell-init`, `version` |
| Shell hooks | `aiswitch use` / `switch` / `deactivate` + wrappers `claude`, `codex`, `cursor-agent`, `agent` |
| TUI | home, detalhes do profile, pick de tool, create wizard, rename, delete confirm, switch result |
| Delete | **somente na TUI** (não há `aiswitch delete` na CLI) |
| Secrets na UI | nunca — só estado `● LINKED` / `○ NOT LINKED` |

**Fora do escopo atual:** Gemini CLI, editor Cursor desktop, conta default global.

---

## Instalação

### Requisitos

- Go (versão em `go.mod`)
- macOS ou Linux
- Agents no `PATH` conforme for usar: `claude`, `codex`, `cursor-agent`

### Build / install

```sh
git clone git@github.com:higordiego/ai-swtich.git aiswitch
cd aiswitch
make install          # → ~/.local/bin/aiswitch
aiswitch version      # 0.1.0
aiswitch doctor
```

Binários pré-compilados: [Releases](https://github.com/higordiego/ai-swtich/releases)
(`darwin`/`linux` × `amd64`/`arm64` + `.sha256`).

```sh
make build            # → bin/aiswitch
```

---

## Início rápido

```sh
# 1) profiles
aiswitch create trabalho pessoal

# 2) login nativo por agent (aiswitch só isola o ambiente)
aiswitch login claude trabalho
aiswitch login codex trabalho
aiswitch login cursor trabalho

# 3) ativar no shell atual
eval "$(aiswitch shell-init zsh)"   # ou: bash
aiswitch use trabalho
claude                              # roda no profile ativo

# 4) ou abrir a TUI
aiswitch
```

Outro terminal → outro profile:

```sh
eval "$(aiswitch shell-init zsh)"
aiswitch use pessoal
codex
```

Launch sem alterar o shell:

```sh
aiswitch run claude trabalho
aiswitch run codex pessoal -- exec --help
aiswitch run cursor trabalho -- --plan
```

---

## TUI — o que a tela realmente faz

`aiswitch` (sem args) abre a interface Bubble Tea. Assinatura no rodapé:
`// feito por Higor Diego`.

### Telas

| Tela | Conteúdo |
|------|----------|
| **Home** | Lista de profiles em árvore (Claude Code / Codex / Cursor Agent com `●`/`○`), badge `● ACTIVE`, item `＋ Create profile` |
| **Profile** | Header `AISWITCH / <nome>`, seção `LINKED AGENTS`, seção `ACTIONS` |
| **Tool pick** | Escolha de agent para Open / Link / Unlink / Status |
| **Create** | Passo 1: nome · Passo 2: checkboxes `[✓]/[ ]` dos 3 agents (`space` alterna) |
| **Rename** | Input do novo nome |
| **Delete confirm** | `DELETE PROFILE` com `[y] Delete` / `[n] Cancel` |
| **Switch result** | `SWITCH IDENTITY` com de→para e status por agent (`✓ switched` / `○ not linked` / `✕ failed`) |

### Home (mock fiel)

```
╭─[ AISWITCH // IDENTITY CONTROL ]─────────────────────────────╮
│  ◈ AISWITCH                                       v0.1.0     │
│  isolated identities for coding agents                       │
╰──────────────────────────────────────────────────────────────╯

  PROFILES                                          2 configured
  ─────────────────────────────────────────────────────────────

  ❯ higor                                              ● ACTIVE
    ├─ Claude Code                                      ●
    ├─ Codex                                            ●
    └─ Cursor Agent                                     ●

    splitwave
    ├─ Claude Code                                      ●
    ├─ Codex                                            ○
    └─ Cursor Agent                                     ○

  ＋ Create profile

────────────────────────────────────────────────────────────────
 ↑↓ Navigate   ↵ Open   [a] Activate   [n] New   [d] Delete   [q] Quit

// feito por Higor Diego
```

### Profile (mock fiel)

```
╭──────────────────────────────────────────────────────────────╮
│  ◈ AISWITCH / higor                              ● ACTIVE    │
╰──────────────────────────────────────────────────────────────╯

  LINKED AGENTS
  ─────────────────────────────────────────────────────────────

  ❯ Claude Code                                      ● LINKED
    Codex                                            ● LINKED
    Cursor Agent                                     ● LINKED

  ACTIONS
  ─────────────────────────────────────────────────────────────

    [s] Switch to this profile
    [l] Link new agent
    [u] Unlink agent
    [r] Rename profile
    [d] Delete profile

────────────────────────────────────────────────────────────────
 ↑↓ Navigate   ↵ Select   [s] Switch   [l] Link   [u] Unlink
 [r] Rename   esc Back   q Quit
```

### Navegação global

| Tecla | Ação |
|-------|------|
| `↑` / `k` | cima |
| `↓` / `j` | baixo |
| `Enter` | abrir / executar seleção |
| `Esc` | voltar (na home: sai) |
| `q` / `Ctrl+C` | sair |

### Atalhos por contexto (como no footer)

**Home**

| Tecla | Ação |
|-------|------|
| `Enter` / `e` | abrir detalhes do profile selecionado |
| `a` / `s` | ativar identidade na **sessão TUI** + tela Switch result |
| `n` | create wizard |
| `d` | confirmar delete do profile selecionado |

**Profile**

| Tecla | Ação |
|-------|------|
| `Enter` em agent **linked** | `run` (abre o agent no profile) |
| `Enter` em agent **not linked** | aviso `⚠ … is not linked` |
| `Enter` em action | executa Switch / Link / Unlink / Rename / Delete |
| `s` / `a` | Switch identity |
| `l` | Link (`login`) → tool pick |
| `u` | Unlink (`logout`) → tool pick (só linked) |
| `r` | Rename |
| `d` | Delete confirm |
| `t` | Status → tool pick |

**Create wizard**

| Tecla | Ação |
|-------|------|
| `Enter` (nome) | vai para seleção de agents |
| `↑↓` / `jk` | move checkbox |
| `space` / `x` | marca/desmarca agent |
| `Enter` (agents) | cria o profile; se houver agents marcados, inicia fila de `login` |

### Selected ≠ Active

- `❯` = cursor (selected)
- `● ACTIVE` = `AISWITCH_PROFILE` do environ **ou** profile ativado com `a`/`s` na TUI

São estados independentes.

### Activate na TUI vs `aiswitch use`

| Mecanismo | Efeito |
|-----------|--------|
| TUI `[a]`/`[s]` | Ativa na sessão da TUI e mostra `SWITCH IDENTITY` por agent |
| `eval "$(aiswitch shell-init …)"` + `aiswitch use PERFIL` | Persiste no **shell pai** (`export AISWITCH_PROFILE=…`) |

A TUI é subprocesso: ela **não** altera o shell pai sozinha.

### Layout responsivo

| Largura | Modo |
|---------|------|
| ≥ 80 cols | completo |
| 60–79 | compacto |
| &lt; 60 | minimal (árvore reduzida / labels curtos) |

### Feedback tipado

| Símbolo | Uso |
|---------|-----|
| `● LINKED` / `○ NOT LINKED` | estado do agent |
| `✓` | sucesso / switched |
| `✕` | erro / failed |
| `⚠` | warning |
| `◌` | loading (ex.: durante exec do agent) |

---

## CLI de referência

Comandos implementados em `internal/cli/app.go`:

```text
aiswitch                         # abre a TUI
aiswitch create PERFIL [PERFIL...]
aiswitch rename ANTIGO NOVO
aiswitch list [--json]
aiswitch current
aiswitch doctor [PERFIL]
aiswitch env PERFIL

aiswitch login  TOOL [PERFIL] [-- ARGS...]
aiswitch logout TOOL [PERFIL]
aiswitch unlink TOOL [PERFIL]      # alias de logout no launcher
aiswitch status TOOL [PERFIL] [-- ARGS...]
aiswitch run    TOOL [PERFIL] [-- ARGS...]

aiswitch shell-init [zsh|bash]
aiswitch version | --version
aiswitch help | -h | --help
```

`TOOL`: `claude` | `codex` | `cursor` (também aceita `cursor-agent` / `agent` → normaliza para `cursor`)

Sem `PERFIL`, usa `AISWITCH_PROFILE`. **Não existe conta padrão global.**

`AISWITCH_ROOT` troca a raiz de storage (default `~/.aiswitch`).

### Shell integration (`shell-init`)

Após `eval "$(aiswitch shell-init zsh)"`:

| Comando | Função |
|---------|--------|
| `aiswitch use PERFIL` / `aiswitch switch PERFIL` | `eval` de `aiswitch env` no shell atual |
| `aiswitch deactivate` | `unset AISWITCH_PROFILE` |
| `claude` / `codex` / `cursor-agent` / `agent` | se houver profile ativo → `aiswitch run <tool> -- …` |

### tmux

Cada pane carrega `shell-init` e escolhe seu profile:

```sh
tmux new -s contas
eval "$(aiswitch shell-init zsh)"
aiswitch use trabalho
codex

tmux split-window -h
eval "$(aiswitch shell-init zsh)"
aiswitch use pessoal
claude
```

---

## Arquitetura real do repo

```
cmd/aiswitch/           entrypoint
internal/cli/
  app.go                CLI commands
  shell.go              shell-init script
  tui.go                model + helpers
  tui_update.go         teclado / atalhos / fluxos
  tui_view.go           render das telas
  theme.go              cores + símbolos
  layout.go             largura / header / footer / frames
internal/profile/       Create Get List Rename Delete SetToolLinked
internal/launch/        Tools, Environment (scrub), Build, Resolve, exec
integration/            PTY / tmux / real-CLI
testdata/provider/      fake provider dos testes
.github/workflows/
  ci.yml                verify + fuzz + tmux (ubuntu/macOS)
  release.yml           cross-compile + GitHub Release em tags v*
```

### Storage

```
~/.aiswitch/
└─ profiles/
   └─ <nome>/
      ├─ profile.json          # metadata (tools linked)
      ├─ claude/
      ├─ anthropic/
      ├─ codex/                # inclui config.toml (credentials store=file)
      └─ cursor/
         └─ data/
```

Permissões esperadas: dirs `0700`, metadata `0600`. Symlinks e dirs públicos são rejeitados. `/` não pode ser root.

### Isolamento no launch

Antes de exec, o launcher:

1. define `AISWITCH_*`, `CLAUDE_CONFIG_DIR`, `ANTHROPIC_CONFIG_DIR`, `CODEX_HOME`, `CURSOR_CONFIG_DIR`, `CURSOR_DATA_DIR`, `AGENT_CLI_CREDENTIAL_STORE=file`
2. **remove** env vars de auth herdadas (Anthropic/OpenAI/Codex/Cursor) — lista em `internal/launch/plan.go` (`blocked`)
3. substitui o processo (`exec`) preservando PTY / sinais / exit code

Detecção de login legado na TUI: só checa presença estrutural em
`.claude.json` / `auth.json` / `cli-config.json` — **sem imprimir tokens**.

---

## Testes

```sh
make test          # go test ./...
make verify        # vet + race + coverage + build
make fuzz          # fuzz de ValidateName
make tmux-test     # AISWITCH_TEST_TMUX=1
make real-test     # AISWITCH_TEST_REAL=1 (CLIs instalados, sem forçar login)
```

CI: [ci.yml](.github/workflows/ci.yml) · Releases: [release.yml](.github/workflows/release.yml)

---

## FAQ

**O aiswitch lê meus tokens?**  
Não. Isola diretórios e detecta só se existe estrutura de auth.

**Posso deletar profile pela CLI?**  
Não nesta versão. Delete é fluxo da TUI (`[d]` + confirmação `[y]`).

**Activate na TUI muda meu zsh?**  
Não. Para o shell pai use `aiswitch use` depois do `shell-init`.

**`cursor` é o editor?**  
Não — é o **Cursor Agent CLI**. Desktop fica fora do v0.1.0.

**Tem Gemini?**  
Não. Roadmap apenas.

---

## Contribuir / segurança / licença

- [CONTRIBUTING.md](CONTRIBUTING.md)
- [SECURITY.md](SECURITY.md)
- MIT © Higor Diego — [LICENSE](LICENSE)

```
// feito por Higor Diego
```
