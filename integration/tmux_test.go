package integration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTmuxIndependentPanesAndCtrlC(t *testing.T) {
	if os.Getenv("AISWITCH_TEST_TMUX") != "1" {
		t.Skip("set AISWITCH_TEST_TMUX=1 for real PTY/tmux test")
	}
	tmux, err := exec.LookPath("tmux")
	if err != nil {
		t.Fatal(err)
	}
	env := newEnv(t)
	login(t, env, "codex", "a", "account-A")
	login(t, env, "codex", "b", "account-B")
	// A dedicated server cannot modify the user's existing sessions.
	socket := fmt.Sprintf("aiswitch-test-%d-%d", os.Getpid(), time.Now().UnixNano())
	call := func(args ...string) string {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, tmux, append([]string{"-L", socket, "-f", "/dev/null"}, args...)...)
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("tmux %v: %v %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		exec.CommandContext(ctx, tmux, "-L", socket, "kill-server").Run()
	})
	logs := t.TempDir()
	script := func(p string) string {
		return "eval \"$(" + shq(binary) + " shell-init bash)\"; ai-switch use " + p + "; codex wait-signal; printf 'EXIT=%s\\n' \"$?\"; exec sleep 20"
	}
	paneA := call("new-session", "-d", "-P", "-F", "#{pane_id}", "-s", "isolation", "-x", "180", "-y", "40", "bash --noprofile --norc -c "+shq(script("a")))
	paneB := call("split-window", "-h", "-P", "-F", "#{pane_id}", "-t", paneA, "bash --noprofile --norc -c "+shq(script("b")))
	waitFor := func(pane string, markers ...string) string {
		t.Helper()
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			out := call("capture-pane", "-p", "-J", "-S", "-100", "-t", pane)
			ok := true
			for _, marker := range markers {
				if !strings.Contains(out, marker) {
					ok = false
				}
			}
			if ok {
				return out
			}
			time.Sleep(50 * time.Millisecond)
		}
		out := call("capture-pane", "-p", "-J", "-S", "-100", "-t", pane)
		t.Fatalf("missing %v in %s", markers, out)
		return ""
	}
	a := waitFor(paneA, `"account":"account-A"`, `"tty":true`)
	b := waitFor(paneB, `"account":"account-B"`, `"tty":true`)
	if strings.Contains(a, `"account":"account-B"`) || strings.Contains(b, `"account":"account-A"`) {
		t.Fatal("account crossed panes")
	}
	call("send-keys", "-t", paneA, "C-c")
	waitFor(paneA, "SIGNAL interrupt", "EXIT=42")
	b = call("capture-pane", "-p", "-J", "-S", "-100", "-t", paneB)
	if strings.Contains(b, "SIGNAL") || strings.Contains(b, "EXIT=") {
		t.Fatal("Ctrl-C affected other pane")
	}
	call("send-keys", "-t", paneB, "C-c")
	waitFor(paneB, "SIGNAL interrupt", "EXIT=42")
	if err := os.WriteFile(filepath.Join(logs, "panes.txt"), []byte(a+"\n"+b), 0600); err != nil {
		t.Fatal(err)
	}
}
