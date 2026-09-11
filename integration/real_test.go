package integration

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRealCLIsFreshProfileIsolation(t *testing.T) {
	if os.Getenv("AISWITCH_TEST_REAL") != "1" {
		t.Skip("set AISWITCH_TEST_REAL=1 for installed CLIs (no login, no inference)")
	}
	root := filepath.Join(t.TempDir(), "profiles")
	env := envWith(os.Environ(), map[string]string{"AISWITCH_ROOT": root, "AISWITCH_PROFILE": "", "NO_COLOR": "1"})
	run(t, env, "create", "empty-a", "empty-b")
	for _, tool := range []string{"claude", "codex", "cursor"} {
		for _, name := range []string{"empty-a", "empty-b"} {
			t.Run(tool+"/"+name, func(t *testing.T) {
				cmd := command(t, env, "status", tool, name)
				var stderr bytes.Buffer
				cmd.Stderr = &stderr
				out, err := cmd.Output()
				// Logged-out is a valid status result, but launch/network errors are not.
				combined := strings.ToLower(string(out) + stderr.String())
				if tool == "claude" {
					var status struct {
						LoggedIn        bool   `json:"loggedIn"`
						ConfigDirectory string `json:"configDirectory"`
					}
					if parseErr := json.Unmarshal(out, &status); parseErr != nil {
						t.Fatalf("status: %v %s %s", err, out, stderr.String())
					}
					if status.LoggedIn {
						t.Fatal("fresh Claude profile inherited login")
					}
					want, _ := filepath.EvalSymlinks(filepath.Join(root, "profiles", name, "claude"))
					if status.ConfigDirectory != want {
						t.Fatalf("wrong config directory: %q want %q", status.ConfigDirectory, want)
					}
				} else if !strings.Contains(combined, "not logged in") {
					t.Fatalf("expected logged out, got %v: %s", err, combined)
				}
				t.Log("fresh profile reported logged out through installed CLI")
			})
		}
	}
}
