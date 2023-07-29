// SPDX-License-Identifier: MIT

package forgejo

import (
	"context"

	"code.gitea.io/gitea/modules/git"

	"github.com/urfave/cli/v2"
	f3_cmd "lab.forgefriends.org/friendlyforgeformat/gof3/cmd"
)

func CmdF3(ctx context.Context) *cli.Command {
	return &cli.Command{
		Name:  "f3",
		Usage: "F3",
		Subcommands: []*cli.Command{
			SubcmdF3Mirror(ctx),
		},
	}
}

func SubcmdF3Mirror(ctx context.Context) *cli.Command {
	mirrorCmd := f3_cmd.CreateCmdMirror(ctx)
	mirrorCmd.Before = prepareWorkPathAndCustomConf(ctx)
	f3Action := mirrorCmd.Action
	mirrorCmd.Action = func(c *cli.Context) error { return runMirror(ctx, c, f3Action) }
	return mirrorCmd
}

func runMirror(ctx context.Context, c *cli.Context, action cli.ActionFunc) error {
	var cancel context.CancelFunc
	if !ContextGetNoInit(ctx) {
		ctx, cancel = installSignals(ctx)
		defer cancel()

		if err := initDB(ctx); err != nil {
			return err
		}

		if err := git.InitSimple(ctx); err != nil {
			return err
		}
	}

	return action(c)
}
