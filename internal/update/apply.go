package update

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/creativeprojects/go-selfupdate"

	"herdr-canvas/internal/version"
)

const releaseRepo = "aorumbayev/herdr-canvas"

// Apply installs latest (a 0.x.y string) using herdr or a standalone replace.
func Apply(ctx context.Context, latest string) error {
	return Default().Apply(ctx, latest)
}

// Apply installs latest using c's seams. Tests must set Updater, Executable,
// ListPlugins, and RunHerdr so this never hits GitHub or replaces the test binary.
func (c *Client) Apply(ctx context.Context, latest string) error {
	if !version.IsRelease() {
		return errDevBuild(version.Version)
	}
	switch c.classify(ctx) {
	case KindLocalLinked:
		return errLocalLinked()
	case KindGitHubManaged:
		return c.applyManaged(ctx, latest)
	default:
		return c.applyStandalone(ctx, latest)
	}
}

func (c *Client) applyManaged(ctx context.Context, latest string) error {
	root := c.lookup("HERDR_PLUGIN_ROOT")
	dir := c.dirOutsidePluginRoot(root)
	env := os.Environ()
	err := c.runHerdr(ctx, HerdrRun{
		Bin:  c.herdrBin(),
		Args: []string{"plugin", "install", releaseRepo, "--ref", "v" + latest, "--yes"},
		Env:  env,
		Dir:  dir,
	})
	return wrapRunError(err)
}

func (c *Client) dirOutsidePluginRoot(root string) string {
	candidates := []string{}
	if home, err := c.home(); err == nil && home != "" {
		candidates = append(candidates, home)
	}
	candidates = append(candidates, c.tempDir())
	if root != "" {
		candidates = append(candidates, filepath.Dir(root))
	}
	for _, dir := range candidates {
		if root == "" || !underDir(root, dir) {
			return dir
		}
	}
	return c.tempDir()
}

func (c *Client) applyStandalone(ctx context.Context, latest string) error {
	goos, goarch := c.goos(), c.goarch()
	if !supported(goos, goarch) {
		return errUnsupportedPlatform(goos, goarch)
	}
	exe, err := c.executable()
	if err != nil {
		return err
	}
	if err := refuseManagedCheckout(exe, c.lookup("HERDR_PLUGIN_ROOT")); err != nil {
		return err
	}
	u := c.Updater
	if u == nil {
		u = goreleaserUpdater{os: goos, arch: goarch}
	}
	return u.UpdateTo(ctx, latest, exe)
}

func supported(goos, goarch string) bool {
	switch goos + "/" + goarch {
	case "darwin/amd64", "darwin/arm64", "linux/amd64", "linux/arm64":
		return true
	default:
		return false
	}
}

func refuseManagedCheckout(exe, pluginRoot string) error {
	if _, err := os.Lstat(exe); err != nil {
		return err
	}
	for _, p := range pathAndResolved(exe) {
		for _, root := range pathAndResolved(pluginRoot) {
			if root != "" && underDir(root, p) {
				return errManagedCheckout()
			}
		}
		if managedGitHubCheckout(p) {
			return errManagedCheckout()
		}
	}
	return nil
}

func pathAndResolved(p string) []string {
	if p == "" {
		return nil
	}
	out := []string{filepath.Clean(p)}
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		cleaned := filepath.Clean(resolved)
		if cleaned != out[0] {
			out = append(out, cleaned)
		}
	}
	return out
}

func managedGitHubCheckout(p string) bool {
	sep := string(filepath.Separator)
	return strings.Contains(filepath.Clean(p)+sep, sep+"plugins"+sep+"github"+sep)
}

func underDir(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// goreleaserUpdater talks to GitHub through go-selfupdate. Tests must not use it.
type goreleaserUpdater struct {
	os, arch string
}

func (g goreleaserUpdater) UpdateTo(ctx context.Context, version, dest string) error {
	filter := archiveFilter(g.os, g.arch)
	u, err := selfupdate.NewUpdater(selfupdate.Config{
		Validator: &selfupdate.ChecksumValidator{UniqueFilename: "checksums.txt"},
		OS:        g.os,
		Arch:      g.arch,
		Filters:   []string{filter},
	})
	if err != nil {
		return err
	}
	rel, found, err := u.DetectVersion(ctx, selfupdate.ParseSlug(releaseRepo), "v"+version)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("release v%s has no matching archive for %s/%s", version, g.os, g.arch)
	}
	return u.UpdateTo(ctx, rel, dest)
}

func archiveFilter(goos, goarch string) string {
	arch := goarch
	if goarch == "amd64" {
		arch = "x86_64"
	}
	return fmt.Sprintf(`_%s_%s\.tar\.gz$`, titleOS(goos), arch)
}

func titleOS(goos string) string {
	if goos == "" {
		return goos
	}
	return strings.ToUpper(goos[:1]) + goos[1:]
}
