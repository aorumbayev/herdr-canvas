package name

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	git(t, dir, "init", "-q")
	git(t, dir, "config", "user.email", "a@b.c")
	git(t, dir, "config", "user.name", "a")
	git(t, dir, "remote", "add", "origin", "https://github.com/foo/herdr-canvas.git")
	return dir
}

func commit(t *testing.T, dir string) {
	t.Helper()
	git(t, dir, "commit", "--allow-empty", "-q", "-m", "x")
}

func TestCompositeRepoAtBranch(t *testing.T) {
	dir := initRepo(t)
	commit(t, dir)
	got, err := Composite(dir)
	if err != nil {
		t.Fatalf("Composite: %v", err)
	}
	if got.String() != "herdr-canvas@main" {
		t.Errorf("got %q, want herdr-canvas@main", got.String())
	}
}

func TestCompositeNoCommitsOmitsBranch(t *testing.T) {
	dir := initRepo(t)
	got, err := Composite(dir)
	if err != nil {
		t.Fatalf("Composite: %v", err)
	}
	if got.String() != "herdr-canvas" {
		t.Errorf("got %q, want herdr-canvas", got.String())
	}
}

func TestCompositeDetachedHeadUsesShortSHA(t *testing.T) {
	dir := initRepo(t)
	commit(t, dir)
	git(t, dir, "checkout", "-q", "--detach")
	wantSHA := git(t, dir, "rev-parse", "--short", "HEAD")
	got, err := Composite(dir)
	if err != nil {
		t.Fatalf("Composite: %v", err)
	}
	if got.String() != "herdr-canvas@"+wantSHA {
		t.Errorf("got %q, want herdr-canvas@%s", got.String(), wantSHA)
	}
}

func TestCompositeSlugsBranchSlash(t *testing.T) {
	dir := initRepo(t)
	commit(t, dir)
	git(t, dir, "checkout", "-q", "-b", "feat/foo")
	got, err := Composite(dir)
	if err != nil {
		t.Fatalf("Composite: %v", err)
	}
	if got.String() != "herdr-canvas@feat-foo" {
		t.Errorf("got %q, want herdr-canvas@feat-foo", got.String())
	}
}

func TestCompositeLinkedWorktree(t *testing.T) {
	dir := initRepo(t)
	commit(t, dir)
	wt := filepath.Join(t.TempDir(), "wt")
	git(t, dir, "worktree", "add", "-q", "-b", "other", wt)
	got, err := Composite(wt)
	if err != nil {
		t.Fatalf("Composite: %v", err)
	}
	if got.String() != "herdr-canvas@other@wt" {
		t.Errorf("got %q, want herdr-canvas@other@wt", got.String())
	}
}

func TestCompositeNotARepo(t *testing.T) {
	if _, err := Composite(t.TempDir()); err != ErrNotRepo {
		t.Errorf("err = %v, want ErrNotRepo", err)
	}
}
