// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package forgejo

import (
	"context"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/storage"

	_ "code.gitea.io/gitea/services/f3/driver" // register the driver
	"code.gitea.io/gitea/services/f3/util"

	"github.com/urfave/cli/v2"
	f3_cmd "lab.forgefriends.org/friendlyforgeformat/gof3/cmd"
	f3_logger "lab.forgefriends.org/friendlyforgeformat/gof3/logger"
	f3_util "lab.forgefriends.org/friendlyforgeformat/gof3/util"
)

func CmdF3(ctx context.Context) *cli.Command {
	ctx = f3_logger.ContextSetLogger(ctx, util.NewF3Logger(nil, log.GetLogger(log.DEFAULT)))
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

		if err := storage.Init(); err != nil {
			return err
		}

		if err := git.InitSimple(ctx); err != nil {
			return err
		}
		if err := models.Init(ctx); err != nil {
			return err
		}
	}

	err := action(c)
	if panicError, ok := err.(f3_util.PanicError); ok {
		log.Debug("F3 Stack trace\n%s", panicError.Stack())
	}
	return err
}
