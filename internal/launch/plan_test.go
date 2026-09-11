package launch

import (
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/higordiego/ai-swtich/internal/profile"
)

func TestResolve(t *testing.T) {
	if path, err := Resolve("sh"); err != nil || path == "" {
		t.Fatalf("resolve sh: %v %q", err, path)
	}
	if _, err := Resolve("aiswitch-program-that-does-not-exist"); err == nil {
		t.Fatal("resolved missing program")
	}
	if _, err := exec.LookPath("cursor-agent"); err == nil {
		if path, err := Resolve("cursor-agent"); err != nil || path == "" {
			t.Fatalf("resolve cursor-agent: %v %q", err, path)
		}
		return
	}
	if _, err := exec.LookPath("agent"); err == nil {
		if path, err := Resolve("cursor-agent"); err != nil || path == "" {
			t.Fatalf("resolve cursor-agent via agent fallback: %v %q", err, path)
		}
	}
}

func TestExecuteMissingProgram(t *testing.T) {
	if err := Execute(Plan{Program: "aiswitch-program-that-does-not-exist"}); err == nil {
		t.Fatal("executed missing program")
	}
}

func envMap(env []string) map[string]string {
	m := map[string]string{}
	for _, e := range env {
		k, v, _ := strings.Cut(e, "=")
		m[k] = v
	}
	return m
}

func TestEnvironmentIsolation(t *testing.T) {
	base := []string{"HOME=/real-home", "PATH=/usr/bin", "TERM=xterm-256color", "TMUX=/socket,1,0", "CODEX_HOME=/native", "CURSOR_DATA_DIR=/native", "DUP=old", "DUP=new"}
	for k := range blocked {
		base = append(base, k+"=SECRET_VALUE")
	}
	before := slices.Clone(base)
	for _, name := range []string{"work", "personal"} {
		p := profile.Profile{Name: name, Path: filepath.Join("/store/profiles", name)}
		env, removed := Environment(base, "/store", p)
		m := envMap(env)
		if len(removed) != len(blocked) {
			t.Fatalf("removed = %v", removed)
		}
		if strings.Contains(strings.Join(env, "\n"), "SECRET_VALUE") {
			t.Fatal("secret survived")
		}
		if m["HOME"] != "/real-home" || m["TERM"] != "xterm-256color" || m["DUP"] != "new" {
			t.Fatal("parent environment lost")
		}
		for _, key := range []string{"CLAUDE_CONFIG_DIR", "CODEX_HOME", "CURSOR_CONFIG_DIR", "CURSOR_DATA_DIR", "ANTHROPIC_CONFIG_DIR"} {
			if !strings.HasPrefix(m[key], p.Path+"/") {
				t.Errorf("%s escaped: %s", key, m[key])
			}
		}
		if m["AGENT_CLI_CREDENTIAL_STORE"] != "file" || m["AISWITCH_PROFILE"] != name {
			t.Fatal("wrong profile")
		}
	}
	if !reflect.DeepEqual(base, before) {
		t.Fatal("mutated parent environment")
	}
}

func TestRouting(t *testing.T) {
	for _, tc := range []struct {
		tool, op, program string
		args              []string
	}{
		{"claude", "login", "claude", []string{"auth", "login", "--claudeai"}},
		{"claude", "status", "claude", []string{"auth", "status"}},
		{"claude", "logout", "claude", []string{"auth", "logout"}},
		{"codex", "login", "codex", []string{"-c", `cli_auth_credentials_store="file"`, "login"}},
		{"codex", "status", "codex", []string{"-c", `cli_auth_credentials_store="file"`, "login", "status"}},
		{"codex", "logout", "codex", []string{"-c", `cli_auth_credentials_store="file"`, "logout"}},
		{"cursor", "login", "cursor-agent", []string{"login"}},
		{"agent", "status", "cursor-agent", []string{"status"}},
		{"cursor-agent", "logout", "cursor-agent", []string{"logout"}},
		{"claude", "run", "claude", nil},
	} {
		t.Run(tc.tool+"/"+tc.op, func(t *testing.T) {
			p, err := Build("/store", profile.Profile{Name: "work", Path: "/store/profiles/work"}, tc.tool, tc.op, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			if p.Program != tc.program || !reflect.DeepEqual(p.Args, tc.args) {
				t.Fatalf("plan = %+v", p)
			}
		})
	}
}

func TestArguments(t *testing.T) {
	args := []string{"exec", "a prompt with spaces", `$(touch /do-not-execute)`}
	p, err := Build("/store", profile.Profile{}, "codex", "run", args, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(p.Args[2:], args) {
		t.Fatal("arguments rewritten")
	}
	for _, args := range [][]string{{"-c", `cli_auth_credentials_store="keyring"`}, {"--config=cli_auth_credentials_store=auto"}, {"-ccli_auth_credentials_store=auto"}, {"--config", `"cli_auth_credentials_store" = "auto"`}, {"-c"}} {
		if _, err := Build("", profile.Profile{}, "codex", "run", args, nil); err == nil {
			t.Errorf("accepted %v", args)
		}
	}
	for _, args := range [][]string{{"--api-key", "secret"}, {"--header=Authorization: secret"}} {
		if _, err := Build("", profile.Profile{}, "cursor", "run", args, nil); err == nil {
			t.Errorf("accepted %v", args)
		}
	}
	if _, err := Build("", profile.Profile{}, "unknown", "run", nil, nil); err == nil {
		t.Fatal("accepted unknown tool")
	}
	if _, err := Build("", profile.Profile{}, "claude", "bad", nil, nil); err == nil {
		t.Fatal("accepted unknown op")
	}
}

func TestConcurrentPlans(t *testing.T) {
	base := []string{"CODEX_HOME=/native", "OPENAI_API_KEY=secret"}
	var wg sync.WaitGroup
	for i := range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			name := "a"
			if i%2 != 0 {
				name = "b"
			}
			p, err := Build("/store", profile.Profile{Name: name, Path: "/store/" + name}, "codex", "run", nil, base)
			if err != nil {
				t.Error(err)
				return
			}
			if envMap(p.Env)["CODEX_HOME"] != "/store/"+name+"/codex" {
				t.Error("crossed profile")
			}
		}()
	}
	wg.Wait()
}
