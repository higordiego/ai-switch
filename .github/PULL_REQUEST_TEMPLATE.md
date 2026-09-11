## Summary

<!-- What changed and why? -->

## Test plan

- [ ] `make verify`
- [ ] `make fuzz` (if touching profile validation)
- [ ] Manual TUI check (if touching `internal/cli`)

## Notes

- Secrets must never be logged or rendered in the TUI/CLI.
- Keep **selected** (`❯`) and **ACTIVE** (`● ACTIVE`) as separate states.
