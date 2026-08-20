package update

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"herdr-canvas/internal/store"
)

func TestClassifyFixtures(t *testing.T) {
	cases := []struct {
		name string
		json string
		want Kind
	}{
		{
			name: "github",
			json: `{"result":{"type":"plugin_list","plugins":[{"plugin_id":"herdr-canvas","source":{"kind":"github"}}]}}`,
			want: KindGitHubManaged,
		},
		{
			name: "local",
			json: `{"result":{"type":"plugin_list","plugins":[{"plugin_id":"herdr-canvas","source":{"kind":"local"}}]}}`,
			want: KindLocalLinked,
		},
		{
			name: "empty",
			json: `{"result":{"type":"plugin_list","plugins":[]}}`,
			want: KindStandalone,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cl := &Client{ListPlugins: func(context.Context) ([]byte, error) { return []byte(c.json), nil }}
			if got := cl.classify(context.Background()); got != c.want {
				t.Fatalf("kind = %v, want %v", got, c.want)
			}
		})
	}
}

func TestClassifyMissingHerdrIsStandalone(t *testing.T) {
	cl := &Client{ListPlugins: func(context.Context) ([]byte, error) {
		return nil, errors.New("herdr: executable file not found")
	}}
	if got := cl.classify(context.Background()); got != KindStandalone {
		t.Fatalf("kind = %v, want standalone", got)
	}
}

func TestApplyDevNoNetworkNoHerdr(t *testing.T) {
	releaseVer(t, "dev")
	cl := &Client{
		ListPlugins: func(context.Context) ([]byte, error) {
			t.Fatal("list")
			return nil, nil
		},
		RunHerdr: func(context.Context, HerdrRun) error {
			t.Fatal("herdr")
			return nil
		},
		Updater: fakeUpdater{t: t},
	}
	if err := cl.Apply(context.Background(), "0.2.0"); err == nil {
		t.Fatal("want dev error")
	}
}

func TestApplyManagedUsesHerdrInstall(t *testing.T) {
	releaseVer(t, "0.1.0")
	var got HerdrRun
	cl := &Client{
		LookupEnv: func(k string) string {
			return map[string]string{
				"HERDR_BIN_PATH":    "/opt/herdr",
				"HERDR_PLUGIN_ROOT": "/plugins/github/aorumbayev/herdr-canvas",
			}[k]
		},
		UserHome: func() (string, error) { return "/home/me", nil },
		TempDir:  func() string { return "/tmp" },
		ListPlugins: func(context.Context) ([]byte, error) {
			return []byte(`{"result":{"type":"plugin_list","plugins":[{"plugin_id":"herdr-canvas","source":{"kind":"github"}}]}}`), nil
		},
		RunHerdr: func(_ context.Context, run HerdrRun) error {
			got = run
			return nil
		},
		Updater: fakeUpdater{t: t},
	}
	if err := cl.Apply(context.Background(), "0.2.0"); err != nil {
		t.Fatal(err)
	}
	wantArgs := []string{"plugin", "install", "aorumbayev/herdr-canvas", "--ref", "v0.2.0", "--yes"}
	if got.Bin != "/opt/herdr" || !reflect.DeepEqual(got.Args, wantArgs) {
		t.Fatalf("run = %+v", got)
	}
	if got.Dir == "/plugins/github/aorumbayev/herdr-canvas" {
		t.Fatal("cwd must leave HERDR_PLUGIN_ROOT")
	}
}

func TestApplyLocalRefuses(t *testing.T) {
	releaseVer(t, "0.1.0")
	cl := &Client{
		ListPlugins: func(context.Context) ([]byte, error) {
			return []byte(`{"result":{"type":"plugin_list","plugins":[{"plugin_id":"herdr-canvas","source":{"kind":"local"}}]}}`), nil
		},
		RunHerdr: func(context.Context, HerdrRun) error {
			t.Fatal("must not install")
			return nil
		},
		Updater: fakeUpdater{t: t},
	}
	err := cl.Apply(context.Background(), "0.2.0")
	if err == nil {
		t.Fatal("want refuse")
	}
}

func TestApplyManagedRegularCheckoutRefuses(t *testing.T) {
	releaseVer(t, "0.1.0")
	dir := t.TempDir()
	target := filepath.Join(dir, "plugins", "github", "aorumbayev", "herdr-canvas", "bin", "herdr-canvas")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	u := &recordingUpdater{}
	cl := standaloneApplyClient(target, u)
	if err := cl.Apply(context.Background(), "0.2.0"); err == nil {
		t.Fatal("want managed checkout refuse")
	}
	if u.called {
		t.Fatal("UpdateTo must not run")
	}
}

func TestApplyManagedPluginRootRegularRefuses(t *testing.T) {
	releaseVer(t, "0.1.0")
	root := t.TempDir()
	dest := filepath.Join(root, "bin", "herdr-canvas")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	u := &recordingUpdater{}
	cl := standaloneApplyClient(dest, u)
	cl.LookupEnv = func(k string) string {
		if k == "HERDR_PLUGIN_ROOT" {
			return root
		}
		return ""
	}
	if err := cl.Apply(context.Background(), "0.2.0"); err == nil {
		t.Fatal("want plugin-root refuse")
	}
	if u.called {
		t.Fatal("UpdateTo must not run")
	}
}

func TestApplyStandaloneRecordsUpdateTo(t *testing.T) {
	releaseVer(t, "0.1.0")
	dest := filepath.Join(t.TempDir(), "herdr-canvas")
	if err := os.WriteFile(dest, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	u := &recordingUpdater{}
	cl := standaloneApplyClient(dest, u)
	cl.RunHerdr = func(context.Context, HerdrRun) error {
		t.Fatal("standalone must not run herdr")
		return nil
	}
	if err := cl.Apply(context.Background(), "0.2.0"); err != nil {
		t.Fatal(err)
	}
	if u.version != "0.2.0" || u.dest != dest {
		t.Fatalf("UpdateTo(%q, %q)", u.version, u.dest)
	}
}

func TestApplyUnsupportedPlatform(t *testing.T) {
	releaseVer(t, "0.1.0")
	u := &recordingUpdater{}
	cl := &Client{
		GOOS:       "windows",
		GOARCH:     "amd64",
		Executable: func() (string, error) { return "/tmp/x", nil },
		ListPlugins: func(context.Context) ([]byte, error) {
			return []byte(`{"result":{"type":"plugin_list","plugins":[]}}`), nil
		},
		Updater: u,
	}
	err := cl.Apply(context.Background(), "0.2.0")
	if err == nil {
		t.Fatal("want WSL2 error")
	}
	if u.called {
		t.Fatal("UpdateTo must not run")
	}
}

func TestApplyManagedSymlinkRefuses(t *testing.T) {
	releaseVer(t, "0.1.0")
	dir := t.TempDir()
	target := filepath.Join(dir, "plugins", "github", "aorumbayev", "herdr-canvas", "bin", "herdr-canvas")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "herdr-canvas")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	u := &recordingUpdater{}
	cl := standaloneApplyClient(link, u)
	if err := cl.Apply(context.Background(), "0.2.0"); err == nil {
		t.Fatal("want managed symlink refuse")
	}
	if u.called {
		t.Fatal("UpdateTo must not run")
	}
}

func TestApplyHerdrExitCode(t *testing.T) {
	releaseVer(t, "0.1.0")
	cl := &Client{
		ListPlugins: func(context.Context) ([]byte, error) {
			return []byte(`{"result":{"type":"plugin_list","plugins":[{"plugin_id":"herdr-canvas","source":{"kind":"github"}}]}}`), nil
		},
		RunHerdr: func(context.Context, HerdrRun) error {
			return &exec.ExitError{ProcessState: fakeFailState(t)}
		},
	}
	err := cl.Apply(context.Background(), "0.2.0")
	var ee *ExitError
	if !errors.As(err, &ee) {
		t.Fatalf("err = %v, want ExitError", err)
	}
}

func fakeFailState(t *testing.T) *os.ProcessState {
	t.Helper()
	cmd := exec.Command("sh", "-c", "exit 7")
	_ = cmd.Run()
	return cmd.ProcessState
}

func TestDismissReadWrite(t *testing.T) {
	state := t.TempDir()
	data := t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)
	t.Setenv("XDG_DATA_HOME", data)
	c := &Client{}
	got, err := c.DismissedVersion()
	if err != nil || got != "" {
		t.Fatalf("missing file: %q %v", got, err)
	}
	if err := c.Dismiss("0.2.0"); err != nil {
		t.Fatal(err)
	}
	got, err = c.DismissedVersion()
	if err != nil || got != "0.2.0" {
		t.Fatalf("got %q %v", got, err)
	}
	hidden, err := c.Hidden("0.2.0")
	if err != nil || !hidden {
		t.Fatalf("same tag should stay hidden: %v %v", hidden, err)
	}
	hidden, err = c.Hidden("0.3.0")
	if err != nil || hidden {
		t.Fatalf("newer tag should show: %v %v", hidden, err)
	}
	entries, err := os.ReadDir(store.Dir())
	if err == nil && len(entries) != 0 {
		t.Fatalf("store dir has files: %v", entries)
	}
}

func standaloneApplyClient(exe string, u AssetUpdater) *Client {
	return &Client{
		GOOS:       "linux",
		GOARCH:     "amd64",
		Executable: func() (string, error) { return exe, nil },
		LookupEnv:  func(string) string { return "" },
		ListPlugins: func(context.Context) ([]byte, error) {
			return []byte(`{"result":{"type":"plugin_list","plugins":[]}}`), nil
		},
		Updater: u,
	}
}

type recordingUpdater struct {
	called  bool
	version string
	dest    string
}

func (u *recordingUpdater) UpdateTo(_ context.Context, version, dest string) error {
	u.called = true
	u.version = version
	u.dest = dest
	return nil
}

type fakeUpdater struct{ t *testing.T }

func (f fakeUpdater) UpdateTo(context.Context, string, string) error {
	f.t.Fatal("UpdateTo must not run")
	return nil
}
