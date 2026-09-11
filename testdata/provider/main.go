// Test-only provider: exercises account lifecycle without network or real tokens.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func main() {
	tool := filepath.Base(os.Args[0])
	args := os.Args[1:]
	dir := os.Getenv("CLAUDE_CONFIG_DIR")
	if tool == "codex" {
		dir = os.Getenv("CODEX_HOME")
		if len(args) >= 2 && args[0] == "-c" {
			args = args[2:]
		}
	}
	if tool == "cursor-agent" || tool == "agent" {
		dir = os.Getenv("CURSOR_CONFIG_DIR")
	}
	if len(args) > 0 && args[0] == "auth" {
		args = args[1:]
	}
	op := "run"
	if len(args) > 0 {
		op = args[0]
	}
	if op == "login" && len(args) > 1 && args[1] == "status" {
		op = "status"
	}
	credential := filepath.Join(dir, "fixture-account")
	switch op {
	case "login":
		if err := os.WriteFile(credential, []byte(os.Getenv("FIXTURE_ACCOUNT")), 0600); err != nil {
			panic(err)
		}
	case "logout":
		if err := os.Remove(credential); err != nil && !os.IsNotExist(err) {
			panic(err)
		}
	}
	data, _ := os.ReadFile(credential)
	wd, _ := os.Getwd()
	size, ttyErr := exec.Command("stty", "size").Output()
	// stty must use the inherited terminal rather than exec.Cmd's default /dev/null.
	stty := exec.Command("stty", "size")
	stty.Stdin = os.Stdin
	size, ttyErr = stty.Output()
	result := map[string]any{"account": string(data), "profile": os.Getenv("AISWITCH_PROFILE"), "dir": dir, "args": args, "cwd": wd, "pid": os.Getpid(), "tty": ttyErr == nil, "size": strings.TrimSpace(string(size))}
	for _, k := range []string{"CLAUDE_CONFIG_DIR", "CODEX_HOME", "CURSOR_CONFIG_DIR", "CURSOR_DATA_DIR", "ANTHROPIC_CONFIG_DIR", "AGENT_CLI_CREDENTIAL_STORE", "OPENAI_API_KEY", "ANTHROPIC_API_KEY", "CURSOR_API_KEY"} {
		result[k] = os.Getenv(k)
	}
	var sig chan os.Signal
	if op == "wait-signal" {
		sig = make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		panic(err)
	}
	if op == "hold" {
		for {
			if _, err := os.Stat(os.Getenv("FIXTURE_RELEASE")); err == nil {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	if op == "wait-signal" {
		fmt.Println("SIGNAL", <-sig)
		os.Exit(42)
	}
	if op == "exit" && len(args) > 1 {
		n, _ := strconv.Atoi(args[1])
		os.Exit(n)
	}
	if op == "echo" {
		buf := make([]byte, 1024)
		n, _ := os.Stdin.Read(buf)
		os.Stdout.Write(buf[:n])
	}
}
