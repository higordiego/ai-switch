# aiswitch

```
╭─[ AISWITCH // IDENTITY CONTROL ]─────────────────────────────╮
│  ◈ isolated identities for coding agents                     │
│                                                              │
│  one terminal → one identity → many agents                   │
╰──────────────────────────────────────────────────────────────╯
```

**AI Identity & Account Switcher** — CLI/TUI em Go para gerenciar identidades
isoladas usadas por coding agents.

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev/)
[![CI](https://github.com/higordiego/ai-swtich/actions/workflows/ci.yml/badge.svg)](https://github.com/higordiego/ai-swtich/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg?style=flat-square)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux-black?style=flat-square)](#instalação)
[![Release](https://img.shields.io/github/v/release/higordiego/ai-swtich?style=flat-square&include_prereleases&sort=semver)](https://github.com/higordiego/ai-swtich/releases)

> Feito por **Higor Diego**

---

## Por que aiswitch existe?

Coding agents modernos — Claude Code, Codex CLI, Cursor Agent — autenticam em
arquivos e variáveis de ambiente **globais do usuário**.

Isso funciona até o dia em que você precisa:

- usar a conta **pessoal** e a conta **da empresa** no mesmo Mac
- abrir dois terminais com identidades diferentes ao mesmo tempo
- evitar que um agent herde o token errado do shell pai
- trocar de contexto sem logout/login manual a cada sessão

Sem isolamento, a identidade vaza entre projetos, clientes e empregadores.

**aiswitch** trata isso como um plano de controle de identidade:

```
onde estou?
    ↓
qual profile está ativo?
    ↓
quais agents estão vinculados?
    ↓
o que acontece se eu trocar?
    ↓
quais tools mudaram — e o que falhou?
```

Ele não substitui o login oficial de cada ferramenta.
Ele **isola** o storage, **roteia** o ambiente e **torna a troca explícita**.

---

## O que ele faz

| Capacidade | Detalhe |
|------------|---------|
| Perfis isolados | Cada profile tem diretórios próprios sob `~/.aiswitch/profiles/<nome>` |
| Multi-agent | Claude Code · Codex · Cursor Agent CLI |
| TUI | Plano de identidades com árvore de agents, badges e feedback tipado |
| Shell integration | `aiswitch use` ativa a identidade no terminal atual |
| Launch seguro | Remove env vars de auth herdadas antes de iniciar o agent |
| Zero secret dump | Nunca imprime tokens, keys, cookies ou private keys |

### Modelo mental

```
┌──────────── terminal A ────────────┐   ┌──────────── terminal B ────────────┐
│  aiswitch use trabalho             │   │  aiswitch use pessoal              │
│  ● ACTIVE → trabalho               │   │  ● ACTIVE → pessoal                │
│                                    │   │                                    │
│  claude  ───▶ credenciais trabalho │   │  claude  ───▶ credenciais pessoal  │
│  codex   ───▶ credenciais trabalho │   │  codex   ───▶ credenciais pessoal  │
└────────────────────────────────────┘   └────────────────────────────────────┘
```

Processos já abertos **mantêm** a identidade com que foram iniciados.
Trocar o profile no shell não reescreve sessões em andamento.

---

## Instalação

### Requisitos

- Go (versão em `go.mod`)
- macOS ou Linux
- Agents que você for usar instalados no `PATH` (`claude`, `codex`, `cursor-agent`)

### Build local

```sh
git clone git@github.com:higordiego/ai-swtich.git aiswitch
cd aiswitch
make install          # → ~/.local/bin/aiswitch
aiswitch version
aiswitch doctor
```

Ou via HTTPS:

```sh
git clone https://github.com/higordiego/ai-swtich.git aiswitch
```

Garanta que `~/.local/bin` esteja no `PATH`.

### Build apenas o binário

```sh
make build            # → bin/aiswitch
./bin/aiswitch --help
```

---

## Início rápido

### 1. Criar identidades

```sh
aiswitch create trabalho pessoal
```

### 2. Vincular agents (login oficial de cada ferramenta)

```sh
aiswitch login claude trabalho
aiswitch login codex trabalho
aiswitch login cursor trabalho

aiswitch login claude pessoal
aiswitch login codex pessoal
aiswitch login cursor pessoal
```

O aiswitch **chama o fluxo nativo** de login. Ele não copia nem exibe tokens.

### 3. Ativar no terminal

```sh
eval "$(aiswitch shell-init zsh)"   # ou bash
aiswitch use trabalho
claude
```

Em outro terminal:

```sh
eval "$(aiswitch shell-init zsh)"
aiswitch use pessoal
codex
```

### 4. Ou abrir a TUI

```sh
aiswitch
```

---

## Tela interativa (TUI)

A interface é terminal-first: minimalista, cyber, sem ASCII art gigante.

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
 ↑↓ Navigate   ↵ Open   [a] Activate   [n] New   [q] Quit

// feito por Higor Diego
```

### Navegação

| Tecla | Ação |
|-------|------|
| `↑` / `k` | cima |
| `↓` / `j` | baixo |
| `Enter` | abrir / executar |
| `Esc` | voltar |
| `q` | sair |

### Atalhos contextuais

O footer mostra **somente** o que vale na tela atual:

| Tecla | Ação |
|-------|------|
| `a` / `s` | ativar identidade (sessão TUI + relatório) |
| `n` | novo profile |
| `l` / `u` | link / unlink agent |
| `r` | renomear |
| `d` | deletar (com confirmação) |
| `t` | status da ferramenta |

### Selected ≠ Active

- `❯` — item sob o cursor
- `● ACTIVE` — identidade em uso na sessão

Você pode navegar em `splitwave` enquanto `higor` continua ACTIVE.

### Feedback

| Símbolo | Significado |
|---------|-------------|
| `●` / `○` | linked / not linked |
| `✓` | sucesso / switched |
| `✕` | falha |
| `⚠` | aviso |
| `◌` | loading |

Ao ativar um profile, a TUI mostra o resultado **por agent** — inclusive falhas.
Nada é escondido.

> Persistência no shell pai continua sendo `aiswitch use PERFIL` após `shell-init`.
> A ativação dentro da TUI controla a sessão interativa e reporta o estado dos agents.

---

## CLI de referência

```text
aiswitch                      # TUI
aiswitch create PERFIL...
aiswitch rename ANTIGO NOVO
aiswitch list [--json]
aiswitch current
aiswitch doctor [PERFIL]

aiswitch login  TOOL [PERFIL]
aiswitch logout TOOL [PERFIL]
aiswitch unlink TOOL [PERFIL]
aiswitch status TOOL [PERFIL]
aiswitch run    TOOL [PERFIL] [-- ARGS...]

aiswitch shell-init [zsh|bash]
aiswitch env PERFIL
aiswitch version
```

`TOOL`: `claude` · `codex` · `cursor` (Cursor Agent CLI)

Sem `PERFIL`, usa `AISWITCH_PROFILE` do terminal. **Não existe conta padrão global.**

### Launch direto (sem alterar o shell)

```sh
aiswitch run claude trabalho
aiswitch run codex pessoal -- exec --help
aiswitch run cursor trabalho -- --plan
```

### tmux — um profile por pane

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

## Como funciona (arquitetura)

```
cmd/aiswitch
    └─ internal/cli          comandos + TUI (Bubble Tea / Lipgloss)
         ├─ internal/profile storage isolado (~/.aiswitch)
         └─ internal/launch  env scrub + exec do agent
```

### Storage

```
~/.aiswitch/
└─ profiles/
   ├─ trabalho/
   │  ├─ profile.json
   │  ├─ claude/
   │  ├─ anthropic/
   │  ├─ codex/
   │  └─ cursor/
   └─ pessoal/
      └─ ...
```

Diretórios privados (`0700` / arquivos `0600`). Symlinks e metadata pública são rejeitados.

### Isolamento de ambiente

Ao lançar um agent, aiswitch:

1. monta o env do profile (`CLAUDE_CONFIG_DIR`, `CODEX_HOME`, `CURSOR_*`, …)
2. **remove** variáveis de autenticação herdadas do processo pai
3. executa o binário oficial no lugar do launcher (PTY, sinais e exit code preservados)

### Segurança por design

- sem dump de secrets na UI ou logs
- sem “conta default” silenciosa
- root do sistema (`/`) não pode ser storage
- ver [SECURITY.md](SECURITY.md)

---

## Testes

Status local da suíte (race + cobertura + build):

```sh
make verify
```

| Comando | O que cobre |
|---------|-------------|
| `make test` | unitários e pacotes |
| `make verify` | `go vet` + race detector + coverage + build |
| `make fuzz` | fuzz de nomes de profile |
| `make tmux-test` | isolamento real com PTY/tmux |
| `make real-test` | smoke com CLIs instalados (sem login forçado) |

A integração usa um provider local falso para validar isolamento, concorrência,
stdin, sinais, argumentos e códigos de saída — sem precisar de contas reais.

CI: [`.github/workflows/ci.yml`](.github/workflows/ci.yml) roda verify, fuzz e
tmux-test em Ubuntu e macOS a cada push/PR.

Releases: tags `v*` disparam [`.github/workflows/release.yml`](.github/workflows/release.yml)
com binários para `darwin`/`linux` (`amd64`/`arm64`) e checksums SHA-256.

---

## Roadmap (ideias)

- Mais agents (ex.: Gemini CLI) quando houver storage/CLI estável
- Isolamento do editor Cursor desktop (hoje só Agent CLI)
- Hooks de shell mais ricos / prompt com profile ativo
- Empacotamento (`brew`, release binários)

Contribuições são bem-vindas — veja [CONTRIBUTING.md](CONTRIBUTING.md).

---

## FAQ

**aiswitch lê meus tokens?**  
Não. Ele aponta cada tool para diretórios isolados e detecta apenas se existe
estrutura de login — nunca imprime o conteúdo.

**Posso ter o mesmo agent logado em dois profiles?**  
Sim. Cada combinação `profile + tool` tem seu próprio login.

**Trocar de profile derruba o Claude que já está aberto?**  
Não. Processos abertos mantêm a identidade com que foram iniciados.

**`cursor` é o editor?**  
Não. No aiswitch, `cursor` significa **Cursor Agent CLI**. O editor desktop
usa outro armazenamento e fica para uma etapa futura.

---

## Licença

MIT © Higor Diego — ver [LICENSE](LICENSE).

---

```
// feito por Higor Diego
// isolated identities for coding agents
```
