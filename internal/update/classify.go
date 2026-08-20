package update

import (
	"context"
	"encoding/json"
	"strings"
)

// Kind is how this process is installed.
type Kind int

const (
	KindStandalone Kind = iota
	KindGitHubManaged
	KindLocalLinked
)

type pluginListEnvelope struct {
	Result struct {
		Type    string `json:"type"`
		Plugins []struct {
			PluginID string `json:"plugin_id"`
			Source   struct {
				Kind string `json:"kind"`
			} `json:"source"`
		} `json:"plugins"`
	} `json:"result"`
}

func (c *Client) classify(ctx context.Context) Kind {
	out, err := c.listPlugins(ctx)
	if err != nil {
		return KindStandalone
	}
	var env pluginListEnvelope
	if err := json.Unmarshal(out, &env); err != nil {
		return KindStandalone
	}
	if len(env.Result.Plugins) == 0 {
		return KindStandalone
	}
	kind := strings.ToLower(env.Result.Plugins[0].Source.Kind)
	switch kind {
	case "local":
		return KindLocalLinked
	case "github":
		return KindGitHubManaged
	default:
		return KindStandalone
	}
}
