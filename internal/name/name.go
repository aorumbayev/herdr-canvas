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

// Composite computes the auto-name for cwd. The repo part is the basename of
// the origin remote. If cwd has no origin remote, the repo part is the
// basename of cwd. The branch part is the current branch, with each `/`
// replaced by `-`. On a detached HEAD, the branch part is a short SHA. In a
// repository with no commits, Composite omits the branch part. The worktree
// part is the basename of the worktree root. Composite adds the worktree part
// only in a linked worktree.
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

// worktreeName returns the worktree-root basename in a linked worktree, else
// the empty string. --git-common-dir and --git-dir differ only in a linked
// worktree.
func worktreeName(cwd string) string {
	common := run(cwd, "rev-parse", "--absolute-git-dir")
	if common == "" {
		return ""
	}
	if run(cwd, "rev-parse", "--path-format=absolute", "--git-common-dir") == common {
		return ""
	}
	return filepath.Base(run(cwd, "rev-parse", "--show-toplevel"))
}

// originBasename strips the scheme, host, and .git suffix from a remote URL.
// An empty URL yields an empty name, so Composite falls back to cwd.
func originBasename(url string) string {
	if url == "" {
		return ""
	}
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
