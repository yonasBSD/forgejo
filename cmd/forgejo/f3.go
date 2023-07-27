// SPDX-License-Identifier: MIT

package forgejo

import (
	"context"
	"fmt"

	auth_model "code.gitea.io/gitea/models/auth"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/services/f3/util"

	"github.com/urfave/cli/v2"
	f3_types "lab.forgefriends.org/friendlyforgeformat/gof3/config/types"
	f3_common "lab.forgefriends.org/friendlyforgeformat/gof3/forges/common"
	f3_format "lab.forgefriends.org/friendlyforgeformat/gof3/format"
)

func CmdF3(ctx context.Context) *cli.Command {
	return &cli.Command{
		Name:        "f3",
		Usage:       "Friendly Forge Format (F3) format export/import.",
		Description: "Import or export a repository from or to the Friendly Forge Format (F3) format.",
		Action:      prepareWorkPathAndCustomConf(ctx, func(cliCtx *cli.Context) error { return RunF3(ctx, cliCtx) }),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "directory",
				Value: "./f3",
				Usage: "Path of the directory where the F3 dump is stored",
			},
			&cli.StringFlag{
				Name:  "user",
				Value: "",
				Usage: "The name of the user who owns the repository",
			},
			&cli.StringFlag{
				Name:  "repository",
				Value: "",
				Usage: "The name of the repository",
			},
			&cli.StringFlag{
				Name:  "authentication-source",
				Value: "",
				Usage: "The name of the authentication source matching the forge of origin",
			},
			&cli.BoolFlag{
				Name:  "no-pull-request",
				Usage: "Do not dump pull requests",
			},
			&cli.BoolFlag{
				Name:  "import",
				Usage: "Import from the directory",
			},
			&cli.BoolFlag{
				Name:  "export",
				Usage: "Export to the directory",
			},
		},
	}
}

func getAuthenticationSource(ctx context.Context, authenticationSource string) (*auth_model.Source, error) {
	source, err := auth_model.GetSourceByName(ctx, authenticationSource)
	if err != nil {
		if auth_model.IsErrSourceNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return source, nil
}

func RunF3(ctx context.Context, cliCtx *cli.Context) error {
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

	doer, err := user_model.GetAdminUser(ctx)
	if err != nil {
		return err
	}

	features := f3_types.AllFeatures
	if cliCtx.Bool("no-pull-request") {
		features.PullRequests = false
	}

	var sourceID int64
	sourceName := cliCtx.String("authentication-source")
	source, err := getAuthenticationSource(ctx, sourceName)
	if err != nil {
		return fmt.Errorf("error retrieving the authentication-source %s %v", sourceName, err)
	}
	if source != nil {
		sourceID = source.ID
	}

	forgejo := util.ForgejoForgeRoot(features, doer, sourceID)
	f3 := util.F3ForgeRoot(features, cliCtx.String("directory"))

	if cliCtx.Bool("export") {
		forgejo.Forge.Users.List(ctx)
		user := forgejo.Forge.Users.GetFromFormat(ctx, &f3_format.User{UserName: cliCtx.String("user")})
		if user.IsNil() {
			return fmt.Errorf("%s is not a known user", cliCtx.String("user"))
		}

		user.Projects.List(ctx)
		project := user.Projects.GetFromFormat(ctx, &f3_format.Project{Name: cliCtx.String("repository")})
		if project.IsNil() {
			return fmt.Errorf("%s/%s is not a known repository", cliCtx.String("user"), cliCtx.String("repository"))
		}

		options := f3_common.NewMirrorOptionsRecurse(user, project)
		f3.Forge.Mirror(ctx, forgejo.Forge, options)
		fmt.Fprintln(ContextGetStdout(ctx), "exported")
	} else if cliCtx.Bool("import") {
		options := f3_common.NewMirrorOptionsRecurse()
		forgejo.Forge.Mirror(ctx, f3.Forge, options)
		fmt.Fprintln(ContextGetStdout(ctx), "imported")
	} else {
		return fmt.Errorf("either --import or --export must be specified")
	}

	return nil
}
