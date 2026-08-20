package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"herdr-canvas/internal/version"
)

const (
	defaultAPIBase = "https://api.github.com"
	latestPath     = "/repos/aorumbayev/herdr-canvas/releases/latest"
	checkTimeout   = 8 * time.Second
	githubAPIVer   = "2022-11-28"
	userAgent      = "herdr-canvas"
)

var tagName = regexp.MustCompile(`^v(0\.\d+\.\d+)$`)

// Result is a latest-release check against a 0.x.y current version.
type Result struct {
	Current string
	Latest  string
	Newer   bool
}

// Check fetches GitHub releases/latest. Callers that see !version.IsRelease
// must skip Check so a development build never opens the network.
func Check(ctx context.Context) (Result, error) {
	return Default().Check(ctx)
}

// Check fetches GitHub releases/latest using c's HTTP client.
func (c *Client) Check(ctx context.Context) (Result, error) {
	if !version.IsRelease() {
		return Result{}, errDevBuild(version.Version)
	}
	base := c.APIBase
	if base == "" {
		base = defaultAPIBase
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+latestPath, nil)
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("X-GitHub-Api-Version", githubAPIVer)
	if tok := c.token(); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}

	res, err := c.httpClient().Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("latest release: %w", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return Result{}, fmt.Errorf("latest release: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		return Result{}, fmt.Errorf("latest release: HTTP %d", res.StatusCode)
	}

	var payload struct {
		Draft   bool   `json:"draft"`
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Result{}, fmt.Errorf("latest release: %w", err)
	}
	if payload.Draft {
		return Result{}, fmt.Errorf("latest release is a draft")
	}
	m := tagName.FindStringSubmatch(payload.TagName)
	if m == nil {
		return Result{}, fmt.Errorf("latest release tag %q is not v0.x.y", payload.TagName)
	}
	latest := m[1]
	current := version.Version
	return Result{
		Current: current,
		Latest:  latest,
		Newer:   compare(current, latest) < 0,
	}, nil
}

func (c *Client) token() string {
	if v := c.lookup("GITHUB_TOKEN"); v != "" {
		return v
	}
	return c.lookup("GH_TOKEN")
}

func (c *Client) httpClient() *http.Client {
	timeout := c.Timeout
	if timeout == 0 {
		timeout = checkTimeout
	}
	if c.HTTP != nil {
		cl := *c.HTTP
		if cl.Timeout == 0 {
			cl.Timeout = timeout
		}
		return &cl
	}
	return &http.Client{Timeout: timeout}
}

// compare returns -1 if current is older than latest, 0 if equal, 1 if current
// is newer. Both sides must be 0.x.y. Comparison is numeric, not string order.
func compare(current, latest string) int {
	a, okA := parseRelease(current)
	b, okB := parseRelease(latest)
	if !okA || !okB {
		return 0
	}
	for i := range a {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}

func parseRelease(v string) ([3]int, bool) {
	var out [3]int
	if !tagName.MatchString("v" + v) {
		return out, false
	}
	parts := strings.Split(v, ".")
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return out, false
		}
		out[i] = n
	}
	return out, true
}
