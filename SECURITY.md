# Security Policy

## What aiswitch does with secrets

aiswitch isolates credential directories per profile and launches coding agents
with scrubbed environments. It does **not** implement its own OAuth store and
must never print:

- access / refresh tokens
- API keys
- cookies
- private keys
- raw credential files

The TUI and CLI only expose linkage state (`linked` / `not linked`).

## Supported versions

| Version | Supported |
|---------|-----------|
| `0.1.x` | Yes |

## Reporting a vulnerability

Please report security issues privately to the maintainer:

- Author: Higor Diego
- Prefer email or a private channel — do not open a public issue with secret
  material, reproduction dumps, or credential paths containing live tokens.

Include:

1. Affected version / commit
2. Impact (credential leak, profile escape, env inheritance, etc.)
3. Minimal reproduction without pasting real secrets

## Hardening expectations

- Profile roots default to `~/.aiswitch` with private directory permissions (`0700` / `0600`)
- Symlinks and world-readable profile metadata are rejected
- Inherited auth-related environment variables are blocked from child processes
- System root (`/`) cannot be used as storage
