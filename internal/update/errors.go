package update

import (
	"errors"
	"fmt"
	"os/exec"
)

// ExitError preserves a subprocess exit status for the CLI.
type ExitError struct {
	Code int
	Msg  string
}

func (e *ExitError) Error() string { return e.Msg }

func (e *ExitError) ExitCode() int { return e.Code }

func errDevBuild(v string) error {
	return fmt.Errorf("this is a development build (%s), not a 0.x.y GitHub Release; install from a GitHub Release or herdr plugin install", v)
}

func errLocalLinked() error {
	return errors.New("this is a herdr plugin link / working-tree checkout; rebuild the tree or use herdr plugin link, not herdr-canvas update")
}

func errUnsupportedPlatform(goos, goarch string) error {
	return fmt.Errorf("unsupported platform %s/%s; herdr-canvas ships darwin/linux amd64/arm64 only — on Windows use WSL2 and the Linux archive, or herdr inside WSL2", goos, goarch)
}

func errManagedCheckout() error {
	return errors.New("this executable is in a herdr-managed plugin checkout; use herdr plugin install, not a standalone replace")
}

func wrapRunError(err error) error {
	if err == nil {
		return nil
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return &ExitError{Code: ee.ExitCode(), Msg: "herdr plugin install failed"}
	}
	return err
}
