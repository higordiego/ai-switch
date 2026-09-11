# Contributing to aiswitch

Thanks for helping improve an identity control plane for coding agents.

## Development setup

```sh
git clone git@github.com:higordiego/ai-swtich.git aiswitch
cd aiswitch
go test ./...
make verify
```

Requirements:

- Go version from `go.mod`
- macOS or Linux
- Optional: `tmux` / `zsh` for integration tests

## Project map

```
cmd/aiswitch/          CLI entrypoint
internal/cli/          commands + interactive TUI
internal/profile/      isolated profile storage
internal/launch/       environment isolation + process launch
integration/           PTY / tmux / real-CLI checks
```

## Guidelines

1. Prefer incremental changes over rewrites.
2. Never log, print, or expose tokens, API keys, cookies, or private keys.
3. Keep selected cursor (`❯`) and active profile (`● ACTIVE`) as separate states in the TUI.
4. Update or add tests with behavior changes (`internal/*/…_test.go`).
5. Run `make verify` before opening a PR.

## Useful commands

| Command | Purpose |
|---------|---------|
| `make test` | Unit + package tests |
| `make verify` | `vet` + race + coverage + build |
| `make fuzz` | Fuzz profile name validation |
| `make tmux-test` | Real PTY isolation via tmux |
| `make real-test` | Smoke against installed agent CLIs |

## Pull requests

- Describe the problem and the approach.
- Include test evidence (`make verify` output is enough for most changes).
- Keep the hacker/terminal visual language if you touch the TUI.

## Code of conduct (short)

Be respectful. Assume good intent. Security reports belong in private channels
described in `SECURITY.md`, not in public issues with secrets attached.
