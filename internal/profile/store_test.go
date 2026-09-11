package profile

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func newStore(t *testing.T) Store {
	t.Helper()
	s, err := New(filepath.Join(t.TempDir(), "store"))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestProfileLifecycle(t *testing.T) {
	s := newStore(t)
	ps, err := s.List()
	if err != nil || len(ps) != 0 {
		t.Fatalf("empty list: %v %v", ps, err)
	}
	if _, err := os.Stat(s.Root); !os.IsNotExist(err) {
		t.Fatal("list created state")
	}
	for _, name := range []string{"work", "personal"} {
		p, err := s.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		got, err := s.Get(name)
		if err != nil {
			t.Fatal(err)
		}
		if p.Name != got.Name || p.Path != got.Path || p.Created != got.Created {
			t.Fatalf("roundtrip: %+v %+v", p, got)
		}
		if _, err := s.Create(name); err == nil {
			t.Fatal("duplicate accepted")
		}
	}
	ps, err = s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(ps) != 2 || ps[0].Name != "personal" || ps[1].Name != "work" {
		t.Fatalf("list: %+v", ps)
	}
	if _, err = s.Get("missing"); err == nil {
		t.Fatal("missing profile accepted")
	}
	err = filepath.Walk(s.Root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		want := os.FileMode(0600)
		if info.IsDir() {
			want = 0700
		}
		if info.Mode().Perm() != want {
			t.Errorf("%s permissions = %o", path, info.Mode().Perm())
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestInvalidNames(t *testing.T) {
	s := newStore(t)
	for _, name := range []string{"", ".", "..", "../escape", "a/b", "a\\b", "Work", "-bad", "a b", "x\nexport BAD=1", strings.Repeat("a", 49), "$(id)", "é"} {
		if _, err := s.Create(name); err == nil {
			t.Errorf("accepted %q", name)
		}
		if _, err := s.Get(name); err == nil {
			t.Errorf("get accepted %q", name)
		}
	}
	if _, err := os.Stat(s.Root); !os.IsNotExist(err) {
		t.Fatal("invalid create wrote state")
	}
	for _, name := range []string{"a", "0", "work-1", "personal_2", strings.Repeat("a", 48)} {
		if err := ValidateName(name); err != nil {
			t.Error(err)
		}
	}
}

func TestConcurrentCreate(t *testing.T) {
	s := newStore(t)
	var successes atomic.Int32
	var wg sync.WaitGroup
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.Create("same"); err == nil {
				successes.Add(1)
			}
		}()
	}
	wg.Wait()
	if successes.Load() != 1 {
		t.Fatalf("successful creators: %d", successes.Load())
	}
	if _, err := s.Get("same"); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Join(s.Root, "profiles"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("orphan staging files: %v", entries)
	}
}

func TestRejectSymlinksAndCorruption(t *testing.T) {
	for _, component := range []string{"profiles/work", "profiles/work/claude", "profiles/work/cursor/data", "profiles/work/profile.json", "profiles"} {
		t.Run(component, func(t *testing.T) {
			s := newStore(t)
			if _, err := s.Create("work"); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(s.Root, component)
			if err := os.Rename(path, path+"-original"); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(path+"-original", path); err != nil {
				t.Fatal(err)
			}
			if _, err := s.Get("work"); err == nil {
				t.Fatal("accepted symlink")
			}
		})
	}
	for _, content := range []string{"garbage", `{"version":2,"name":"work"}`, `{"version":1,"name":"someone-else"}`} {
		s := newStore(t)
		p, err := s.Create("work")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(p.Path, "profile.json"), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Get("work"); err == nil {
			t.Fatal("accepted corrupt metadata")
		}
	}
}

func TestRejectPublicDirectory(t *testing.T) {
	s := newStore(t)
	p, err := s.Create("work")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p.Path, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get("work"); err == nil {
		t.Fatal("accepted public profile")
	}
}

func TestRootCanonicalization(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(dir, "alias")
	if err := os.Symlink(dir, link); err != nil {
		t.Fatal(err)
	}
	a, err := New(filepath.Join(dir, "new"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := New(filepath.Join(link, "new"))
	if err != nil {
		t.Fatal(err)
	}
	if a.Root != b.Root {
		t.Fatalf("same store, different roots: %s %s", a.Root, b.Root)
	}
	if _, err := New("/"); err == nil {
		t.Fatal("accepted system root")
	}
}

func TestStoreErrorPaths(t *testing.T) {
	if _, err := New(""); err != nil {
		t.Fatal(err)
	}
	if _, err := New(filepath.Join(t.TempDir(), "missing", "store")); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	s := Store{Root: file}
	if _, err := s.Create("work"); err == nil {
		t.Fatal("created below a file")
	}
	if _, err := s.List(); err == nil {
		t.Fatal("listed a file as store")
	}
	if _, err := s.Get("work"); err == nil {
		t.Fatal("opened a file as store")
	}
	broken := filepath.Join(t.TempDir(), "broken")
	if err := os.Symlink(filepath.Join(t.TempDir(), "does-not-exist"), broken); err != nil {
		t.Fatal(err)
	}
	if _, err := New(filepath.Join(broken, "store")); err == nil {
		t.Fatal("accepted broken symlink root")
	}
	fileRoot := filepath.Join(t.TempDir(), "file-root")
	if err := os.WriteFile(fileRoot, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := New(filepath.Join(fileRoot, "store")); err == nil {
		t.Fatal("accepted file as ancestor")
	}

	s = Store{Root: t.TempDir()}
	if err := os.Chmod(s.Root, 0700); err != nil {
		t.Fatal(err)
	}
	if got, err := s.List(); err != nil || len(got) != 0 {
		t.Fatalf("missing profiles directory: %v %v", got, err)
	}
	if err := os.MkdirAll(filepath.Join(s.Root, "profiles", ".partial"), 0700); err != nil {
		t.Fatal(err)
	}
	if got, err := s.List(); err != nil || len(got) != 0 {
		t.Fatalf("hidden staging directory: %v %v", got, err)
	}
	s2 := Store{Root: t.TempDir()}
	if err := os.Chmod(s2.Root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s2.Root, "profiles"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s2.Create("blocked"); err == nil {
		t.Fatal("created through profiles file")
	}

	s = newStore(t)
	if err := os.MkdirAll(filepath.Join(s.Root, "profiles"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.Root, "profiles", "bad"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.List(); err == nil {
		t.Fatal("accepted non-directory profile")
	}

	s = newStore(t)
	if err := os.MkdirAll(filepath.Join(s.Root, "profiles"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.Root, "profiles", "work"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create("work"); err == nil {
		t.Fatal("accepted profile file")
	}

	s = newStore(t)
	if _, err := s.Create("work"); err != nil {
		t.Fatal(err)
	}
	meta := filepath.Join(s.Root, "profiles", "work", "profile.json")
	if err := os.Rename(meta, meta+".missing"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get("work"); err == nil {
		t.Fatal("accepted profile without metadata")
	}
	if err := os.Rename(meta+".missing", meta); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{"claude", "anthropic", "codex", "cursor", "cursor/data"} {
		path := filepath.Join(s.Root, "profiles", "work", dir)
		if err := os.Rename(path, path+".missing"); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Get("work"); err == nil {
			t.Fatalf("accepted missing %s", dir)
		}
		if err := os.Rename(path+".missing", path); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chmod(meta, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get("work"); err == nil {
		t.Fatal("accepted public metadata")
	}
}

func TestRenameProfile(t *testing.T) {
	s := newStore(t)
	if _, err := s.Rename("missing", "new"); err == nil {
		t.Fatal("renamed missing profile")
	}
	p, err := s.Create("work")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetToolLinked("work", "codex", true); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p.Path, "claude", "marker"), []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := s.Rename("work", "personal")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "personal" || got.Path != filepath.Join(s.Root, "profiles", "personal") {
		t.Fatalf("renamed profile: %+v", got)
	}
	if !got.Tools["codex"] {
		t.Fatal("rename lost linked tool")
	}
	if _, err := s.Get("work"); err == nil {
		t.Fatal("old name still exists")
	}
	if data, err := os.ReadFile(filepath.Join(got.Path, "claude", "marker")); err != nil || string(data) != "keep" {
		t.Fatal("profile data was not preserved")
	}
	if _, err := s.Rename("personal", "personal"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Rename("personal", "bad/name"); err == nil {
		t.Fatal("invalid new name accepted")
	}
	if _, err := s.Create("other"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Rename("personal", "other"); err == nil {
		t.Fatal("duplicate new name accepted")
	}

	s = newStore(t)
	if _, err := s.Create("source"); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(s.Root, "profiles"), 0500); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Rename("source", "renamed"); err == nil {
		t.Fatal("rename succeeded in read-only parent")
	}
	if err := os.Chmod(filepath.Join(s.Root, "profiles"), 0700); err != nil {
		t.Fatal(err)
	}

	s = newStore(t)
	p, err = s.Create("locked")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p.Path, 0500); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Rename("locked", "locked-new"); err == nil {
		t.Fatal("metadata write succeeded in read-only profile")
	}
	if err := os.Chmod(filepath.Join(s.Root, "profiles", "locked"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get("locked"); err != nil {
		t.Fatal("rename failure did not roll back")
	}

	s = newStore(t)
	if _, err := s.Create("publish"); err != nil {
		t.Fatal(err)
	}
	oldPublish := publishProfileMetadata
	publishProfileMetadata = func(string, string) error { return errors.New("publish failed") }
	_, err = s.Rename("publish", "published")
	publishProfileMetadata = oldPublish
	if err == nil {
		t.Fatal("metadata publish failure ignored")
	}
	if _, err := s.Get("publish"); err != nil {
		t.Fatal("publish failure did not roll back")
	}
}

func TestToolLinkState(t *testing.T) {
	s := newStore(t)
	if _, err := s.SetToolLinked("missing", "claude", true); err == nil {
		t.Fatal("linked missing profile")
	}
	if _, err := s.Create("work"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetToolLinked("work", "claude", true); err != nil {
		t.Fatal(err)
	}
	p, err := s.Get("work")
	if err != nil {
		t.Fatal(err)
	}
	if !p.Tools["claude"] {
		t.Fatal("link was not persisted")
	}
	if _, err := s.SetToolLinked("work", "claude", false); err != nil {
		t.Fatal(err)
	}
	p, err = s.Get("work")
	if err != nil {
		t.Fatal(err)
	}
	if p.Tools["claude"] {
		t.Fatal("unlink was not persisted")
	}
	if _, err := s.SetToolLinked("work", "unknown", true); err == nil {
		t.Fatal("unknown tool accepted")
	}
	p, err = s.Get("work")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p.Path, 0500); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetToolLinked("work", "codex", true); err == nil {
		t.Fatal("linked read-only profile")
	}
	if err := os.Chmod(p.Path, 0700); err != nil {
		t.Fatal(err)
	}
	oldPublish := publishToolMetadata
	publishToolMetadata = func(string, string) error { return errors.New("publish link failed") }
	if _, err := s.SetToolLinked("work", "codex", true); err == nil {
		t.Fatal("link publish failure ignored")
	}
	publishToolMetadata = oldPublish
}

func FuzzValidateName(f *testing.F) {
	for _, v := range []string{"work", "../escape", "a b", "a", "A", "x\x00y"} {
		f.Add(v)
	}
	f.Fuzz(func(t *testing.T, name string) {
		if ValidateName(name) != nil {
			return
		}
		if len(name) < 1 || len(name) > 48 || strings.ContainsAny(name, "/.\\\x00\n") {
			t.Fatalf("unsafe accepted name %q", name)
		}
		if filepath.Base(name) != name {
			t.Fatal("not a basename")
		}
	})
}


func TestDeleteProfile(t *testing.T) {
	s := newStore(t)
	if err := s.Delete("missing"); err == nil {
		t.Fatal("deleted missing profile")
	}
	p, err := s.Create("work")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p.Path, "claude", "marker"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete("work"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get("work"); err == nil {
		t.Fatal("deleted profile still readable")
	}
	if _, err := os.Lstat(p.Path); !os.IsNotExist(err) {
		t.Fatal("profile directory still exists")
	}
	if err := s.Delete("bad/name"); err == nil {
		t.Fatal("invalid name accepted")
	}
}
