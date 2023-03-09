// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"

	actions_model "code.gitea.io/gitea/models/actions"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/util"

	"github.com/urfave/cli"
)

// CmdActions represents the available actions sub-command.
var CmdActions = cli.Command{
	Name:        "actions",
	Usage:       "Actions",
	Description: "Actions",
	Action:      runActions,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "registration-token-admin",
			Usage: "Show the runner registration admin token",
		},
	},
}

func runActions(ctx *cli.Context) error {
	setting.InitProviderFromExistingFile()
	setting.LoadCommonSettings()
	setting.LoadDBSetting()

	stdCtx, cancel := installSignals()
	defer cancel()

	if err := db.InitEngine(stdCtx); err != nil {
		fmt.Println(err)
		fmt.Println("Check if you are using the right config file. You can use a --config directive to specify one.")
		return nil
	}

	if ctx.Bool("registration-token-admin") {
		// ownid=0,repo_id=0,means this token is used for global
		return runActionsRegistrationToken(stdCtx, 0, 0)
	}
	return nil
}

func runActionsRegistrationToken(stdCtx context.Context, ownerID, repoID int64) error {
	var token *actions_model.ActionRunnerToken
	token, err := actions_model.GetUnactivatedRunnerToken(stdCtx, ownerID, repoID)
	if errors.Is(err, util.ErrNotExist) {
		token, err = actions_model.NewRunnerToken(stdCtx, ownerID, repoID)
		if err != nil {
			log.Fatalf("CreateRunnerToken %v", err)
		}
	} else if err != nil {
		log.Fatalf("GetUnactivatedRunnerToken %v", err)
	}
	fmt.Print(token.Token)
	return nil
}
