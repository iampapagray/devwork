package branch

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/iampapagray/devwork/internal/issue"
)

func TestHasVersionToken(t *testing.T) {
	if HasVersionToken("{key}_{slug}") {
		t.Error("default template should not activate the gate")
	}
	if !HasVersionToken("v{version}/{key}_{slug}") {
		t.Error("expected version token detected")
	}
}

func TestValidateTemplate(t *testing.T) {
	if err := ValidateTemplate("v{version}/{key}_{slug}"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := ValidateTemplate("{key}_{bogus}_{type}"); err == nil {
		t.Error("expected error for unknown tokens")
	}
}

func TestRender(t *testing.T) {
	got, err := Render("v{version}/{key}_{slug}", map[string]string{
		"version": "7.3", "key": "QA-1", "slug": "fix-login",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != "v7.3/QA-1_fix-login" {
		t.Errorf("got %q", got)
	}

	if _, err := Render("v{version}/{key}", map[string]string{"key": "QA-1"}); err == nil {
		t.Error("expected error for missing version value")
	}
}

func TestGate(t *testing.T) {
	v := func(s string) issue.Version { return issue.ParseVersion(s) }

	eq := Gate(v("7.3"), v("7.3"), false)
	if eq.Status != StatusEqual || eq.NeedsConfirm || eq.Abort {
		t.Errorf("equal: %+v", eq)
	}

	ahead := Gate(v("7.10"), v("7.9"), false)
	if ahead.Status != StatusAhead || !ahead.NeedsConfirm || ahead.Version != "7.10" {
		t.Errorf("ahead: %+v", ahead)
	}

	behind := Gate(v("7.2"), v("7.3"), false)
	if behind.Status != StatusBehind || !behind.NeedsConfirm || behind.Abort {
		t.Errorf("behind non-strict: %+v", behind)
	}

	strict := Gate(v("7.2"), v("7.3"), true)
	if !strict.Abort {
		t.Errorf("behind strict should abort: %+v", strict)
	}
}

func TestResolveRepoVersion(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"version":"7.3.1"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	v, err := ResolveRepoVersion(root, "package.json", nil)
	if err != nil || v.MajorMinor() != "7.3" {
		t.Fatalf("package.json: v=%v err=%v", v, err)
	}

	root2 := t.TempDir()
	if err := os.WriteFile(filepath.Join(root2, "VERSION"), []byte("v8.0.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	v, err = ResolveRepoVersion(root2, "VERSION", nil)
	if err != nil || v.MajorMinor() != "8.0" {
		t.Fatalf("VERSION: v=%v err=%v", v, err)
	}

	// git-tag source
	v, err = ResolveRepoVersion(root2, "git-tag", func() (string, error) { return "v9.4.2", nil })
	if err != nil || v.MajorMinor() != "9.4" {
		t.Fatalf("git-tag: v=%v err=%v", v, err)
	}

	// auto falls through to VERSION when no package.json
	v, err = ResolveRepoVersion(root2, "auto", func() (string, error) { return "", os.ErrNotExist })
	if err != nil || v.MajorMinor() != "8.0" {
		t.Fatalf("auto: v=%v err=%v", v, err)
	}
}
