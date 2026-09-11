package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"aiswitch/internal/launch"
)

func TestCLIFlow(t *testing.T) {
	var out, stderr bytes.Buffer
	var plans []launch.Plan
	a := App{Out: &out, Err: &stderr, Executable: "/bin/aiswitch", Environ: []string{"AISWITCH_ROOT=" + filepath.Join(t.TempDir(), "store")}, Execute: func(p launch.Plan) error { plans = append(plans, p); return nil }}
	run := func(args ...string) string {
		t.Helper()
		out.Reset()
		if err := a.Run(args); err != nil {
			t.Fatal(err)
		}
		return out.String()
	}
	run("create", "work", "personal")
	if got := run("list"); !strings.Contains(got, "work") || !strings.Contains(got, "personal") {
		t.Fatal(got)
	}
	if got := run("env", "work"); !strings.Contains(got, "export AISWITCH_PROFILE='work'") {
		t.Fatal(got)
	}
	if err := a.Run([]string{"run", "codex"}); err == nil {
		t.Fatal("implicit global profile")
	}
	a.Environ = append(a.Environ, "AISWITCH_PROFILE=work")
	if got := run("current"); got != "work\n" {
		t.Fatal(got)
	}
	run("run", "codex", "personal", "--", "exec", "hello world")
	run("status", "claude")
	if !reflect.DeepEqual(plans[0].Args, []string{"-c", `cli_auth_credentials_store="file"`, "exec", "hello world"}) {
		t.Fatal(plans[0])
	}
	if got := run("current"); got != "work\n" {
		t.Fatal("explicit run changed selected profile")
	}
	if len(plans) != 2 {
		t.Fatal(plans)
	}
}

func TestCLIRejectsInvalidInput(t *testing.T) {
	for _, args := range [][]string{{"create"}, {"unknown"}, {"run"}, {"run", "codex"}, {"run", "codex", "work", "--version"}, {"env"}, {"env", "../escape"}, {"list", "junk"}, {"current", "junk"}, {"doctor", "a", "b"}, {"shell-init", "fish"}, {"use", "work"}, {"deactivate"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var out bytes.Buffer
			a := App{Out: &out, Err: &out, Environ: []string{"AISWITCH_ROOT=" + filepath.Join(t.TempDir(), "store")}}
			if err := a.Run(args); err == nil {
				t.Fatal("accepted invalid command")
			}
		})
	}
}

func TestShellQuote(t *testing.T) {
	if quote("abc' xyz") != "'abc'\"'\"' xyz'" {
		t.Fatal(quote("abc' xyz"))
	}
}

func TestCLICommandSurface(t *testing.T) {
	root := filepath.Join(t.TempDir(), "store")
	var out, stderr bytes.Buffer
	executed := 0
	a := App{Out: &out, Err: &stderr, Executable: "/bin/aiswitch", Environ: []string{"AISWITCH_ROOT=" + root}, Execute: func(launch.Plan) error { executed++; return nil }}
	for _, args := range [][]string{{"help"}, {"-h"}, {"--help"}} {
		out.Reset()
		if err := a.Run(args); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), "aiswitch") {
			t.Fatal("help missing")
		}
	}
	for _, args := range [][]string{{"version"}, {"--version"}} {
		out.Reset()
		if err := a.Run(args); err != nil {
			t.Fatal(err)
		}
		if strings.TrimSpace(out.String()) != Version {
			t.Fatal(out.String())
		}
	}
	if err := a.Run([]string{"version", "extra"}); err == nil {
		t.Fatal("accepted version arguments")
	}
	for _, args := range [][]string{{"shell-init"}, {"shell-init", "bash"}, {"shell-init", "zsh"}} {
		out.Reset()
		if err := a.Run(args); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), "aiswitch()") || !strings.Contains(out.String(), "cursor-agent()") {
			t.Fatal("shell integration incomplete")
		}
	}
	if err := a.Run([]string{"create", "work"}); err != nil {
		t.Fatal(err)
	}
	a.Environ = append(a.Environ, "AISWITCH_PROFILE=work", "OPENAI_API_KEY=hidden")
	for _, args := range [][]string{{"list", "--json"}, {"doctor"}, {"doctor", "work"}} {
		out.Reset()
		err := a.Run(args)
		if err != nil {
			t.Fatal(err)
		}
	}
	a.Resolve = func(string) (string, error) { return "", errors.New("missing") }
	if err := a.Run([]string{"doctor", "work"}); err == nil {
		t.Fatal("doctor accepted missing tools")
	}
	if err := a.Run([]string{"current"}); err != nil {
		t.Fatal(err)
	}
	if err := a.Run([]string{"env", "work"}); err != nil {
		t.Fatal(err)
	}
	if err := a.Run([]string{"run", "codex", "work", "--", "exec"}); err != nil {
		t.Fatal(err)
	}
	if executed != 1 {
		t.Fatalf("execute calls: %d", executed)
	}
	if err := a.Run([]string{"rename", "work", "renamed"}); err != nil {
		t.Fatal(err)
	}
	if err := a.Run([]string{"unlink", "codex", "renamed"}); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"current", "extra"}, {"env"}, {"doctor", "missing"}, {"run", "claude", "missing"}, {"unknown"}} {
		if err := a.Run(args); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	for _, args := range [][]string{{"rename"}, {"rename", "renamed"}, {"rename", "renamed", "bad/name"}} {
		if err := a.Run(args); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}

func TestMainReturnsSuccessAndFailure(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	os.Args = []string{"aiswitch", "version"}
	if got := Main(); got != 0 {
		t.Fatalf("success code: %d", got)
	}
	os.Args = []string{"aiswitch", "unknown"}
	if got := Main(); got != 1 {
		t.Fatalf("failure code: %d", got)
	}
}
