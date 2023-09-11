package repo

import (
	"code.gitea.io/gitea/models"
	ctx "code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/services/task"
	"fmt"
)

// TODO: FORM AND UI FOR ADDITION OF NEW SOURCES (God, is hard to live without frontend)

func SetupSourcesPost(ctx *ctx.Context) error {
	// heres where we would retrieve value from a form
	source := models.CreateNewSource(
		ctx.Doer.ID,
		models.GithubStarred,
		"cassiozareck",
		"",
	)

	err := models.SaveSource(ctx, &source)
	if err != nil {
		return fmt.Errorf("Couldn't save source into database: ", err)
	}

	// todo: this should not call SyncSources
	err = task.SyncSources(ctx, ctx.Doer, ctx.ContextUser)
	if err != nil {
		return fmt.Errorf("Couldn't sync source", err)
	}

	return nil
}
