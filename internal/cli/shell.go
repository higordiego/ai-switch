package cli

import (
	"fmt"
	"io"
	"strings"
)

func quote(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }

func shellInit(w io.Writer, executable string) {
	// Function-local assignments only run after validation succeeds. No global default.
	fmt.Fprintf(w, `aiswitch() {
  if [ "${1-}" = use ] || [ "${1-}" = switch ]; then
    local _aiswitch_exports
    shift
    _aiswitch_exports=$(%s env "$@") || return $?
    eval "$_aiswitch_exports"
  elif [ "${1-}" = deactivate ]; then
    if [ "$#" -ne 1 ]; then printf 'uso: aiswitch deactivate\n' >&2; return 2; fi
    unset AISWITCH_PROFILE
  else
    %s "$@"
  fi
}
`, quote(executable), quote(executable))
	for _, pair := range [][2]string{{"claude", "claude"}, {"codex", "codex"}, {"cursor-agent", "cursor"}, {"agent", "cursor"}} {
		fmt.Fprintf(w, `%s() {
  if [ -n "${AISWITCH_PROFILE-}" ]; then
    %s run %s -- "$@"
  else
    command %s "$@"
  fi
}
`, pair[0], quote(executable), pair[1], pair[0])
	}
}
