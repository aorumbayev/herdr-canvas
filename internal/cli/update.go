package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"herdr-canvas/internal/update"
	"herdr-canvas/internal/version"
)

var (
	updateCheck = update.Check
	updateApply = update.Apply
)

func updateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Install the latest GitHub Release of herdr-canvas",
		Args:  cobra.NoArgs,
		RunE:  runUpdate,
	}
}

func runUpdate(cmd *cobra.Command, args []string) error {
	if !version.IsRelease() {
		return fmt.Errorf("this is a development build (%s), not a 0.x.y GitHub Release; install from a GitHub Release or herdr plugin install", version.Version)
	}
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	res, err := updateCheck(ctx)
	if err != nil {
		return err
	}
	if !res.Newer {
		fmt.Fprintf(cmd.OutOrStdout(), "already up to date (%s)\n", res.Current)
		return nil
	}
	return updateApply(ctx, res.Latest)
}
