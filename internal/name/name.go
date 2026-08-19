package name

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
)

// ErrNotRepo reports that cwd is not inside a git repository.
var ErrNotRepo = errors.New("not a git repository")

// Name is a composite auto-name repo@branch[@worktree].
type Name struct {
	Repo     string
	Branch   string
	Worktree string
}

func (n Name) String() string {
	s := n.Repo
	if n.Branch != "" {
		s += "@" + n.Branch
	}
	if n.Worktree != "" {
		s += "@" + n.Worktree
	}
	return s
}

// Composite computes the auto-name for cwd per the herdr-canvas rule:
// repo = origin remote basename (else cwd basename); branch = current
// branch with `/` slugged to `-`, a short SHA on detached HEAD, or omitted
// with no commits; worktree = worktree-root basename, only in a linked
// worktree.
func Composite(cwd string) (Name, error) {
	if run(cwd, "rev-parse", "--is-inside-work-tree") != "true" {
		return Name{}, ErrNotRepo
	}
	repo := repoName(cwd)
	if repo == "" {
		repo = filepath.Base(cwd)
	}
	return Name{
		Repo:     repo,
		Branch:   branchName(cwd),
		Worktree: worktreeName(cwd),
	}, nil
}

func repoName(cwd string) string {
	return originBasename(run(cwd, "config", "--get", "remote.origin.url"))
}

func branchName(cwd string) string {
	if run(cwd, "rev-parse", "HEAD") == "" {
		return "" // no commits yet — omit branch
	}
	if b := run(cwd, "branch", "--show-current"); b != "" {
		return strings.ReplaceAll(b, "/", "-")
	}
	return run(cwd, "rev-parse", "--short", "HEAD")
}

func worktreeName(cwd string) string {
	if !strings.Contains(run(cwd, "rev-parse", "--git-dir"), "worktrees") {
		return ""
	}
	return filepath.Base(run(cwd, "rev-parse", "--show-toplevel"))
}

// originBasename strips the scheme, host, and .git suffix from a remote URL.
func originBasename(url string) string {
	url = strings.TrimSuffix(url, ".git")
	if i := strings.LastIndex(url, ":"); i >= 0 && !strings.Contains(url[:i], "/") {
		url = url[i+1:]
	}
	return filepath.Base(url)
}

func run(cwd string, args ...string) string {
	out, err := exec.Command("git", append([]string{"-C", cwd}, args...)...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
