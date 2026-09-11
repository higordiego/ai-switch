package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"
)

var namePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,47}$`)

type Profile struct {
	Version int             `json:"version"`
	Name    string          `json:"name"`
	Created time.Time       `json:"created"`
	Path    string          `json:"path"`
	Tools   map[string]bool `json:"tools,omitempty"`
}

type Store struct{ Root string }

var publishProfileMetadata = os.Rename
var publishToolMetadata = os.Rename

func ValidateName(name string) error {
	if !namePattern.MatchString(name) {
		return errors.New("perfil invalido: use 1-48 caracteres a-z, 0-9, _ ou -; comece com letra ou numero")
	}
	return nil
}

func New(root string) (Store, error) {
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return Store{}, err
		}
		root = filepath.Join(home, ".aiswitch")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return Store{}, err
	}
	// Resolve existing ancestors (including macOS /tmp) without creating anything.
	base := abs
	var tail []string
	for {
		if _, err := os.Lstat(base); err == nil {
			break
		} else if !errors.Is(err, os.ErrNotExist) {
			return Store{}, err
		}
		tail = append(tail, filepath.Base(base))
		base = filepath.Dir(base)
	}
	resolved, err := filepath.EvalSymlinks(base)
	if err != nil {
		return Store{}, err
	}
	for i := len(tail) - 1; i >= 0; i-- {
		resolved = filepath.Join(resolved, tail[i])
	}
	if resolved == string(filepath.Separator) {
		return Store{}, errors.New("a raiz do sistema nao pode ser um armazenamento")
	}
	return Store{Root: resolved}, nil
}

func checkDir(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("diretorio invalido ou symlink: %s", path)
	}
	if info.Mode().Perm()&0077 != 0 {
		return fmt.Errorf("diretorio deve ser privado (chmod 700): %s", path)
	}
	return nil
}

func privateDir(path string) error {
	if err := os.MkdirAll(path, 0700); err != nil {
		return err
	}
	return checkDir(path)
}

func (s Store) Create(name string) (Profile, error) {
	if err := ValidateName(name); err != nil {
		return Profile{}, err
	}
	if err := privateDir(s.Root); err != nil {
		return Profile{}, err
	}
	parent := filepath.Join(s.Root, "profiles")
	if err := privateDir(parent); err != nil {
		return Profile{}, err
	}
	dest := filepath.Join(parent, name)
	if _, err := os.Lstat(dest); err == nil {
		return Profile{}, fmt.Errorf("perfil %q ja existe", name)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Profile{}, err
	}
	tmp, err := os.MkdirTemp(parent, ".create-")
	if err != nil {
		return Profile{}, err
	}
	defer os.RemoveAll(tmp)
	p := Profile{Version: 1, Name: name, Created: time.Now().UTC(), Path: dest, Tools: map[string]bool{}}
	for _, dir := range []string{"claude", "anthropic", "codex", "cursor", "cursor/data"} {
		if err := privateDir(filepath.Join(tmp, dir)); err != nil {
			return Profile{}, err
		}
	}
	if err := os.WriteFile(filepath.Join(tmp, "codex", "config.toml"), []byte("cli_auth_credentials_store = \"file\"\n"), 0600); err != nil {
		return Profile{}, err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return Profile{}, err
	}
	if err := os.WriteFile(filepath.Join(tmp, "profile.json"), append(data, '\n'), 0600); err != nil {
		return Profile{}, err
	}
	// Publish a complete, nonempty directory. Concurrent creation cannot replace it.
	if err := os.Rename(tmp, dest); err != nil {
		return Profile{}, fmt.Errorf("publicar perfil %q (pode ja existir): %w", name, err)
	}
	return p, nil
}

func (s Store) Get(name string) (Profile, error) {
	if err := ValidateName(name); err != nil {
		return Profile{}, err
	}
	path := filepath.Join(s.Root, "profiles", name)
	for _, dir := range []string{s.Root, filepath.Join(s.Root, "profiles"), path} {
		if err := checkDir(dir); err != nil {
			return Profile{}, fmt.Errorf("abrir perfil %q: %w", name, err)
		}
	}
	meta := filepath.Join(path, "profile.json")
	info, err := os.Lstat(meta)
	if err != nil {
		return Profile{}, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > 16384 {
		return Profile{}, errors.New("metadados de perfil invalidos ou nao privados")
	}
	data, err := os.ReadFile(meta)
	if err != nil {
		return Profile{}, err
	}
	var p Profile
	if err := json.Unmarshal(data, &p); err != nil {
		return Profile{}, fmt.Errorf("metadados invalidos: %w", err)
	}
	if p.Version != 1 || p.Name != name {
		return Profile{}, errors.New("versao ou nome nos metadados invalido")
	}
	if p.Tools == nil {
		p.Tools = map[string]bool{}
	}
	p.Path = path
	for _, dir := range []string{"claude", "anthropic", "codex", "cursor", "cursor/data"} {
		if err := checkDir(filepath.Join(path, dir)); err != nil {
			return Profile{}, err
		}
	}
	return p, nil
}

func (s Store) SetToolLinked(name, tool string, linked bool) (Profile, error) {
	if tool != "claude" && tool != "codex" && tool != "cursor" {
		return Profile{}, fmt.Errorf("ferramenta desconhecida: %s", tool)
	}
	p, err := s.Get(name)
	if err != nil {
		return Profile{}, err
	}
	p.Tools[tool] = linked
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return Profile{}, err
	}
	tmp, err := os.CreateTemp(p.Path, ".profile-")
	if err != nil {
		return Profile{}, fmt.Errorf("atualizar vinculo: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		return Profile{}, err
	}
	if err := tmp.Close(); err != nil {
		return Profile{}, err
	}
	if err := publishToolMetadata(tmpName, filepath.Join(p.Path, "profile.json")); err != nil {
		return Profile{}, fmt.Errorf("publicar vinculo: %w", err)
	}
	return s.Get(name)
}

func (s Store) Rename(oldName, newName string) (Profile, error) {
	if err := ValidateName(newName); err != nil {
		return Profile{}, err
	}
	p, err := s.Get(oldName)
	if err != nil {
		return Profile{}, err
	}
	if oldName == newName {
		return p, nil
	}
	dest := filepath.Join(s.Root, "profiles", newName)
	if _, err := os.Lstat(dest); err == nil {
		return Profile{}, fmt.Errorf("perfil %q ja existe", newName)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Profile{}, err
	}
	if err := os.Rename(p.Path, dest); err != nil {
		return Profile{}, fmt.Errorf("renomear perfil: %w", err)
	}
	rollback := func() { _ = os.Rename(dest, p.Path) }
	data, err := json.MarshalIndent(Profile{Version: p.Version, Name: newName, Created: p.Created, Path: dest, Tools: p.Tools}, "", "  ")
	if err != nil {
		rollback()
		return Profile{}, err
	}
	tmp, err := os.CreateTemp(dest, ".profile-")
	if err != nil {
		rollback()
		return Profile{}, fmt.Errorf("atualizar metadados: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		rollback()
		return Profile{}, err
	}
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		rollback()
		return Profile{}, err
	}
	if err := tmp.Close(); err != nil {
		rollback()
		return Profile{}, err
	}
	if err := publishProfileMetadata(tmpName, filepath.Join(dest, "profile.json")); err != nil {
		rollback()
		return Profile{}, fmt.Errorf("publicar metadados: %w", err)
	}
	return s.Get(newName)
}

func (s Store) Delete(name string) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	p, err := s.Get(name)
	if err != nil {
		return err
	}
	parent := filepath.Join(s.Root, "profiles")
	if err := checkDir(parent); err != nil {
		return err
	}
	if err := os.RemoveAll(p.Path); err != nil {
		return fmt.Errorf("remover perfil %q: %w", name, err)
	}
	return nil
}

func (s Store) List() ([]Profile, error) {
	result := []Profile{}
	if _, err := os.Lstat(s.Root); errors.Is(err, os.ErrNotExist) {
		return result, nil
	}
	if err := checkDir(s.Root); err != nil {
		return nil, err
	}
	parent := filepath.Join(s.Root, "profiles")
	if _, err := os.Lstat(parent); errors.Is(err, os.ErrNotExist) {
		return result, nil
	}
	if err := checkDir(parent); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(parent)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if len(e.Name()) > 0 && e.Name()[0] == '.' {
			continue
		}
		p, err := s.Get(e.Name())
		if err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}
