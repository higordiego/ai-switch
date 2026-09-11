package integration

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

var binary, providerDir string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "aiswitch-integration-")
	if err != nil {
		panic(err)
	}
	binary = filepath.Join(dir, "ai-switch")
	providerDir = filepath.Join(dir, "providers")
	if err = os.Mkdir(providerDir, 0700); err != nil {
		panic(err)
	}
	for _, build := range [][2]string{{binary, "../cmd/ai-switch"}, {filepath.Join(providerDir, "provider"), "../testdata/provider"}} {
		cmd := exec.Command("go", "build", "-o", build[0], build[1])
		if output, err := cmd.CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "build: %v\n%s", err, output)
			os.RemoveAll(dir)
			os.Exit(1)
		}
	}
	for _, name := range []string{"claude", "codex", "cursor-agent", "agent"} {
		if err := os.Symlink(filepath.Join(providerDir, "provider"), filepath.Join(providerDir, name)); err != nil {
			panic(err)
		}
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

func envWith(base []string, values map[string]string) []string {
	result := []string{}
	for _, s := range base {
		k, _, _ := strings.Cut(s, "=")
		if _, ok := values[k]; !ok {
			result = append(result, s)
		}
	}
	for k, v := range values {
		result = append(result, k+"="+v)
	}
	return result
}

func newEnv(t *testing.T) []string {
	t.Helper()
	env := envWith(os.Environ(), map[string]string{
		"AISWITCH_ROOT": filepath.Join(t.TempDir(), "state with ' quote"), "AISWITCH_PROFILE": "",
		"PATH":           providerDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		"OPENAI_API_KEY": "fixture-inherited-secret", "ANTHROPIC_API_KEY": "fixture-inherited-secret", "CURSOR_API_KEY": "fixture-inherited-secret",
	})
	run(t, env, "create", "a", "b")
	return env
}

func command(t *testing.T, env []string, args ...string) *exec.Cmd {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Env = env
	return cmd
}

func run(t *testing.T, env []string, args ...string) []byte {
	t.Helper()
	cmd := command(t, env, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("%v: %v\n%s\n%s", args, err, out, stderr.String())
	}
	return out
}

type record struct {
	Account string   `json:"account"`
	Profile string   `json:"profile"`
	Dir     string   `json:"dir"`
	Args    []string `json:"args"`
	CWD     string   `json:"cwd"`
	PID     int      `json:"pid"`
	TTY     bool     `json:"tty"`
}

func decode(t *testing.T, data []byte) record {
	t.Helper()
	var r record
	if err := json.Unmarshal(bytes.TrimSpace(data), &r); err != nil {
		t.Fatalf("decode %s: %v", data, err)
	}
	return r
}

func login(t *testing.T, env []string, tool, profile, account string) {
	t.Helper()
	run(t, envWith(env, map[string]string{"FIXTURE_ACCOUNT": account}), "login", tool, profile)
}

func TestLoginRunStatusLogoutIsolation(t *testing.T) {
	t.Parallel()
	for _, tool := range []string{"claude", "codex", "cursor"} {
		t.Run(tool, func(t *testing.T) {
			env := newEnv(t)
			login(t, env, tool, "a", "account-A")
			login(t, env, tool, "b", "account-B")
			for _, p := range []string{"a", "b"} {
				r := decode(t, run(t, env, "status", tool, p))
				want := "account-" + strings.ToUpper(p)
				if r.Account != want || r.Profile != p {
					t.Fatalf("crossed account: %+v", r)
				}
			}
			run(t, env, "logout", tool, "a")
			if r := decode(t, run(t, env, "status", tool, "a")); r.Account != "" {
				t.Fatalf("logout failed: %+v", r)
			}
			if r := decode(t, run(t, env, "status", tool, "b")); r.Account != "account-B" {
				t.Fatalf("logout affected B: %+v", r)
			}
		})
	}
}

func TestConcurrentProcessesRetainAccounts(t *testing.T) {
	t.Parallel()
	env := newEnv(t)
	for _, tool := range []string{"claude", "codex", "cursor"} {
		login(t, env, tool, "a", "account-A")
		login(t, env, tool, "b", "account-B")
	}
	release := filepath.Join(t.TempDir(), "release")
	var running []*exec.Cmd
	for _, tool := range []string{"claude", "codex", "cursor"} {
		for _, p := range []string{"a", "b"} {
			cmd := command(t, envWith(env, map[string]string{"FIXTURE_RELEASE": release}), "run", tool, p, "--", "hold")
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err = cmd.Start(); err != nil {
				t.Fatal(err)
			}
			running = append(running, cmd)
			line, err := bufio.NewReader(stdout).ReadBytes('\n')
			if err != nil {
				t.Fatal(err)
			}
			r := decode(t, line)
			if r.Profile != p || r.Account != "account-"+strings.ToUpper(p) {
				t.Fatalf("wrong process: %+v", r)
			}
			if r.PID != cmd.Process.Pid {
				t.Fatalf("PID changed: launcher=%d provider=%d", cmd.Process.Pid, r.PID)
			}
			var full map[string]any
			if err = json.Unmarshal(line, &full); err != nil {
				t.Fatal(err)
			}
			for _, key := range []string{"OPENAI_API_KEY", "ANTHROPIC_API_KEY", "CURSOR_API_KEY"} {
				if full[key] != "" {
					t.Fatalf("inherited %s", key)
				}
			}
		}
	}
	// All six providers are still alive while another terminal changes selection.
	run(t, env, "env", "b")
	for _, cmd := range running {
		if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
			t.Fatalf("process exited early: %v", err)
		}
	}
	if err := os.WriteFile(release, []byte("go"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, cmd := range running {
		if err := cmd.Wait(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestExitSignalsStdinAndArguments(t *testing.T) {
	t.Parallel()
	env := newEnv(t)
	cmd := command(t, env, "run", "codex", "a", "--", "exit", "37")
	if err := cmd.Run(); err == nil {
		t.Fatal("lost exit status")
	} else if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 37 {
		t.Fatal(err)
	}
	cmd = command(t, env, "run", "claude", "a", "--", "echo")
	cmd.Stdin = strings.NewReader("input with spaces\n")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasSuffix(out, []byte("input with spaces\n")) {
		t.Fatal("stdin not forwarded")
	}
	r := decode(t, run(t, env, "run", "claude", "a", "--", "prompt", `a 'quoted' prompt`, "$(exit 99)"))
	if strings.Join(r.Args, "|") != "prompt|a 'quoted' prompt|$(exit 99)" {
		t.Fatal(r.Args)
	}
	wd, _ := os.Getwd()
	if r.CWD != wd {
		t.Fatalf("cwd changed: %s", r.CWD)
	}
	cmd = command(t, env, "run", "cursor", "b", "--", "wait-signal")
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	reader := bufio.NewReader(pipe)
	if _, err = reader.ReadBytes('\n'); err != nil {
		t.Fatal(err)
	}
	if err = cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	line, _ := reader.ReadString('\n')
	if !strings.Contains(line, "SIGNAL") {
		t.Fatal(line)
	}
	if err = cmd.Wait(); err == nil {
		t.Fatal("expected provider exit 42")
	} else if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 42 {
		t.Fatal(err)
	}
}

func shq(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }

func TestShellIntegration(t *testing.T) {
	t.Parallel()
	for _, shell := range []string{"bash", "zsh"} {
		t.Run(shell, func(t *testing.T) {
			path, err := exec.LookPath(shell)
			if err != nil {
				t.Skip(shell + " unavailable")
			}
			env := newEnv(t)
			login(t, env, "codex", "a", "account-A")
			login(t, env, "codex", "b", "account-B")
			script := `set -eu
eval "$(` + shq(binary) + ` shell-init ` + shell + `)"
ai-switch use a
codex --version
if ai-switch use missing; then exit 90; fi
ai-switch current
ai-switch switch b
codex --version
ai-switch deactivate
test -z "${AISWITCH_PROFILE-}"
`
			args := []string{"--noprofile", "--norc", "-c", script}
			if shell == "zsh" {
				args = []string{"-f", "-c", script}
			}
			cmd := exec.Command(path, args...)
			cmd.Env = env
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			out, err := cmd.Output()
			if err != nil {
				t.Fatalf("shell: %v %s %s", err, out, stderr.String())
			}
			lines := bytes.Split(bytes.TrimSpace(out), []byte("\n"))
			if len(lines) != 3 {
				t.Fatalf("output: %s", out)
			}
			if decode(t, lines[0]).Account != "account-A" || string(lines[1]) != "a" || decode(t, lines[2]).Account != "account-B" {
				t.Fatalf("wrong shell selection: %s", out)
			}
		})
	}
}

func TestConcurrentCreationAcrossProcesses(t *testing.T) {
	t.Parallel()
	env := newEnv(t)
	var wg sync.WaitGroup
	codes := make(chan bool, 12)
	for range 12 {
		wg.Add(1)
		cmd := command(t, env, "create", "same")
		go func() { defer wg.Done(); codes <- cmd.Run() == nil }()
	}
	wg.Wait()
	close(codes)
	success := 0
	for ok := range codes {
		if ok {
			success++
		}
	}
	if success != 1 {
		t.Fatalf("creators succeeded: %d", success)
	}
	run(t, env, "status", "claude", "same")
}
