package update

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"herdr-canvas/internal/version"
)

func releaseVer(t *testing.T, v string) {
	t.Helper()
	prev := version.Version
	version.Version = v
	t.Cleanup(func() { version.Version = prev })
}

func TestCheckSkipsHTTPWhenNotRelease(t *testing.T) {
	releaseVer(t, "dev")
	var hit bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		t.Error("HTTP must not run for a development build")
	}))
	t.Cleanup(srv.Close)
	c := &Client{HTTP: srv.Client(), APIBase: srv.URL}
	_, err := c.Check(context.Background())
	if err == nil {
		t.Fatal("want development-build error")
	}
	if hit {
		t.Fatal("contacted the fixture server")
	}
}

func TestCheckHappyLatest(t *testing.T) {
	releaseVer(t, "0.1.0")
	srv := githubLatest(t, `{"draft":false,"tag_name":"v0.2.0"}`, http.StatusOK)
	c := &Client{HTTP: srv.Client(), APIBase: srv.URL}
	got, err := c.Check(context.Background())
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !got.Newer || got.Latest != "0.2.0" || got.Current != "0.1.0" {
		t.Fatalf("got %+v", got)
	}
}

func TestCheckEqualAndNewerCurrent(t *testing.T) {
	srv := githubLatest(t, `{"draft":false,"tag_name":"v0.2.0"}`, http.StatusOK)
	c := &Client{HTTP: srv.Client(), APIBase: srv.URL}

	releaseVer(t, "0.2.0")
	got, err := c.Check(context.Background())
	if err != nil {
		t.Fatalf("equal: %v", err)
	}
	if got.Newer {
		t.Fatalf("equal should not be newer: %+v", got)
	}

	releaseVer(t, "0.3.0")
	got, err = c.Check(context.Background())
	if err != nil {
		t.Fatalf("current newer: %v", err)
	}
	if got.Newer {
		t.Fatalf("current 0.3.0 vs latest 0.2.0 should not update: %+v", got)
	}
}

func TestCheckHeadersAndToken(t *testing.T) {
	releaseVer(t, "0.1.0")
	var got http.Header
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		path = r.URL.Path
		io.WriteString(w, `{"draft":false,"tag_name":"v0.1.1"}`)
	}))
	t.Cleanup(srv.Close)
	c := &Client{
		HTTP:      srv.Client(),
		APIBase:   srv.URL,
		LookupEnv: func(k string) string { return map[string]string{"GITHUB_TOKEN": "tok"}[k] },
	}
	if _, err := c.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if path != latestPath {
		t.Errorf("path = %s, want %s", path, latestPath)
	}
	if got.Get("Accept") != "application/vnd.github+json" {
		t.Errorf("Accept = %q", got.Get("Accept"))
	}
	if got.Get("User-Agent") != "herdr-canvas" {
		t.Errorf("User-Agent = %q", got.Get("User-Agent"))
	}
	if got.Get("X-GitHub-Api-Version") != githubAPIVer {
		t.Errorf("api version = %q", got.Get("X-GitHub-Api-Version"))
	}
	if got.Get("Authorization") != "Bearer tok" {
		t.Errorf("Authorization = %q", got.Get("Authorization"))
	}
}

func TestCheckErrors(t *testing.T) {
	releaseVer(t, "0.1.0")
	cases := []struct {
		name   string
		status int
		body   string
		sub    string
	}{
		{"404", http.StatusNotFound, `{"message":"no"}`, "HTTP 404"},
		{"draft", http.StatusOK, `{"draft":true,"tag_name":"v0.2.0"}`, "draft"},
		{"malformed", http.StatusOK, `{"draft":false,"tag_name":"nightly"}`, "not v0.x.y"},
		{"v1", http.StatusOK, `{"draft":false,"tag_name":"v1.0.0"}`, "not v0.x.y"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			srv := githubLatest(t, c.body, c.status)
			cl := &Client{HTTP: srv.Client(), APIBase: srv.URL}
			_, err := cl.Check(context.Background())
			if err == nil || !strings.Contains(err.Error(), c.sub) {
				t.Fatalf("err = %v, want substring %q", err, c.sub)
			}
		})
	}
}

func TestCheckTimeout(t *testing.T) {
	releaseVer(t, "0.1.0")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	t.Cleanup(srv.Close)
	c := &Client{HTTP: srv.Client(), APIBase: srv.URL, Timeout: 20 * time.Millisecond}
	_, err := c.Check(context.Background())
	if err == nil {
		t.Fatal("want timeout error")
	}
}

func TestCompareNumeric(t *testing.T) {
	if compare("0.1.0", "0.2.0") >= 0 {
		t.Fatal("0.1.0 < 0.2.0")
	}
	if compare("0.2.0", "0.2.0") != 0 {
		t.Fatal("equal")
	}
	if compare("0.2.0", "0.1.0") <= 0 {
		t.Fatal("0.2.0 > 0.1.0")
	}
	if compare("0.1.9", "0.1.10") >= 0 {
		t.Fatal("0.1.9 < 0.1.10 numerically")
	}
}

func githubLatest(t *testing.T, body string, status int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != latestPath {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv
}
