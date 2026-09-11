package launch

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/higordiego/ai-swtich/internal/profile"
)

var Tools = []string{"claude", "codex", "cursor"}

func Normalize(tool string) (string, error) {
	switch tool {
	case "claude", "codex":
		return tool, nil
	case "cursor", "cursor-agent", "agent":
		return "cursor", nil
	default:
		return "", fmt.Errorf("ferramenta %q desconhecida; use claude, codex ou cursor (Agent CLI)", tool)
	}
}

type Plan struct {
	Program string
	Args    []string
	Env     []string
	Removed []string
}

// These inputs can silently select credentials or providers from the parent session.
var blocked = map[string]bool{
	"ANTHROPIC_API_KEY": true, "ANTHROPIC_AUTH_TOKEN": true,
	"ANTHROPIC_BASE_URL": true, "ANTHROPIC_CUSTOM_HEADERS": true,
	"ANTHROPIC_PROFILE": true, "ANTHROPIC_FEDERATION_RULE_ID": true,
	"ANTHROPIC_ORGANIZATION_ID": true, "ANTHROPIC_SERVICE_ACCOUNT_ID": true,
	"ANTHROPIC_WORKSPACE_ID": true, "ANTHROPIC_IDENTITY_TOKEN": true,
	"ANTHROPIC_IDENTITY_TOKEN_FILE": true,
	"CLAUDE_CODE_OAUTH_TOKEN":       true, "CLAUDE_CODE_OAUTH_REFRESH_TOKEN": true,
	"CLAUDE_CODE_OAUTH_SCOPES": true, "CLAUDE_CODE_API_KEY_HELPER_FD": true,
	"CLAUDE_CODE_OAUTH_TOKEN_FILE_DESCRIPTOR": true,
	"CLAUDE_CODE_USE_BEDROCK":                 true, "CLAUDE_CODE_USE_VERTEX": true,
	"CLAUDE_CODE_USE_FOUNDRY": true, "CLAUDE_CODE_USE_ANTHROPIC_AWS": true,
	"OPENAI_API_KEY": true, "OPENAI_BASE_URL": true, "OPENAI_ORG_ID": true,
	"OPENAI_ORGANIZATION": true, "OPENAI_PROJECT_ID": true,
	"CODEX_API_KEY": true, "CODEX_ACCESS_TOKEN": true,
	"CURSOR_API_KEY": true, "CURSOR_API_ENDPOINT": true,
}

func Environment(base []string, root string, p profile.Profile) ([]string, []string) {
	m := map[string]string{}
	removed := map[string]bool{}
	for _, item := range base {
		key, val, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		if blocked[key] {
			removed[key] = true
			continue
		}
		m[key] = val
	}
	m["AISWITCH_ROOT"] = root
	m["AISWITCH_PROFILE"] = p.Name
	m["CLAUDE_CONFIG_DIR"] = filepath.Join(p.Path, "claude")
	m["ANTHROPIC_CONFIG_DIR"] = filepath.Join(p.Path, "anthropic")
	m["CODEX_HOME"] = filepath.Join(p.Path, "codex")
	m["CURSOR_CONFIG_DIR"] = filepath.Join(p.Path, "cursor")
	m["CURSOR_DATA_DIR"] = filepath.Join(p.Path, "cursor", "data")
	m["AGENT_CLI_CREDENTIAL_STORE"] = "file"
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	result := make([]string, 0, len(keys))
	for _, k := range keys {
		result = append(result, k+"="+m[k])
	}
	names := make([]string, 0, len(removed))
	for k := range removed {
		names = append(names, k)
	}
	sort.Strings(names)
	return result, names
}

func Build(root string, p profile.Profile, tool, operation string, args, base []string) (Plan, error) {
	tool, err := Normalize(tool)
	if err != nil {
		return Plan{}, err
	}
	if err := validateArgs(tool, args); err != nil {
		return Plan{}, err
	}
	plan := Plan{Program: tool}
	plan.Env, plan.Removed = Environment(base, root, p)
	if tool == "cursor" {
		plan.Program = "cursor-agent"
	}
	if tool == "codex" {
		plan.Args = append(plan.Args, "-c", `cli_auth_credentials_store="file"`)
	}
	switch operation {
	case "run":
	case "login", "logout", "status":
		switch tool {
		case "claude":
			plan.Args = append(plan.Args, "auth", operation)
			if operation == "login" {
				plan.Args = append(plan.Args, "--claudeai")
			}
		case "codex":
			if operation == "status" {
				plan.Args = append(plan.Args, "login", "status")
			} else {
				plan.Args = append(plan.Args, operation)
			}
		case "cursor":
			plan.Args = append(plan.Args, operation)
		}
	default:
		return Plan{}, errors.New("operacao desconhecida")
	}
	plan.Args = append(plan.Args, args...)
	return plan, nil
}

func validateArgs(tool string, args []string) error {
	for i, arg := range args {
		if arg == "--" {
			break
		}
		if tool == "cursor" && (arg == "--api-key" || strings.HasPrefix(arg, "--api-key=") || arg == "-H" || arg == "--header" || strings.HasPrefix(arg, "--header=")) {
			return errors.New("credencial por argumento substitui a conta; use aiswitch login cursor PERFIL")
		}
		if tool != "codex" {
			continue
		}
		value := ""
		switch {
		case arg == "-c" || arg == "--config":
			if i+1 >= len(args) {
				return errors.New("falta valor para --config")
			}
			value = args[i+1]
		case strings.HasPrefix(arg, "--config="):
			value = strings.TrimPrefix(arg, "--config=")
		case strings.HasPrefix(arg, "-c") && len(arg) > 2:
			value = strings.TrimPrefix(arg, "-c")
		}
		key, _, _ := strings.Cut(value, "=")
		key = strings.Trim(strings.TrimSpace(key), "\"'")
		if key == "cli_auth_credentials_store" {
			return errors.New("cli_auth_credentials_store e gerenciado pelo aiswitch")
		}
	}
	return nil
}
