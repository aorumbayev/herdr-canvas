package update

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"
)

// AssetUpdater downloads, verifies, and replaces a standalone binary.
// Production uses creativeprojects/go-selfupdate; tests inject a fake.
type AssetUpdater interface {
	UpdateTo(ctx context.Context, version, dest string) error
}

// HerdrRun is one herdr invocation. Tests inject this so Apply never execs herdr.
type HerdrRun struct {
	Bin  string
	Args []string
	Env  []string
	Dir  string
}

// Client holds injectable seams for Check, classify, Apply, and dismiss.
type Client struct {
	HTTP      *http.Client
	APIBase   string
	Timeout   time.Duration
	LookupEnv func(string) string
	UserHome  func() (string, error)
	TempDir   func() string

	ListPlugins func(ctx context.Context) ([]byte, error)
	RunHerdr    func(ctx context.Context, run HerdrRun) error
	Executable  func() (string, error)
	Updater     AssetUpdater
	GOOS        string
	GOARCH      string
}

// Default returns production seams. Tests construct a Client instead.
func Default() *Client {
	return &Client{}
}

func (c *Client) lookup(key string) string {
	if c.LookupEnv != nil {
		return c.LookupEnv(key)
	}
	return os.Getenv(key)
}

func (c *Client) home() (string, error) {
	if c.UserHome != nil {
		return c.UserHome()
	}
	return os.UserHomeDir()
}

func (c *Client) tempDir() string {
	if c.TempDir != nil {
		return c.TempDir()
	}
	return os.TempDir()
}

func (c *Client) goos() string {
	if c.GOOS != "" {
		return c.GOOS
	}
	return runtime.GOOS
}

func (c *Client) goarch() string {
	if c.GOARCH != "" {
		return c.GOARCH
	}
	return runtime.GOARCH
}

func (c *Client) executable() (string, error) {
	if c.Executable != nil {
		return c.Executable()
	}
	return os.Executable()
}

func (c *Client) herdrBin() string {
	if p := c.lookup("HERDR_BIN_PATH"); p != "" {
		return p
	}
	return "herdr"
}

func (c *Client) listPlugins(ctx context.Context) ([]byte, error) {
	if c.ListPlugins != nil {
		return c.ListPlugins(ctx)
	}
	cmd := exec.CommandContext(ctx, c.herdrBin(), "plugin", "list", "--json", "--plugin", "herdr-canvas")
	return cmd.Output()
}

func (c *Client) runHerdr(ctx context.Context, run HerdrRun) error {
	if c.RunHerdr != nil {
		return c.RunHerdr(ctx, run)
	}
	cmd := exec.CommandContext(ctx, run.Bin, run.Args...)
	if len(run.Env) > 0 {
		cmd.Env = run.Env
	}
	cmd.Dir = run.Dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
