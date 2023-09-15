package repo

import (
	"code.gitea.io/gitea/models"
	ctx "code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/services/task"
)

// TODO: FORM AND UI FOR ADDITION OF NEW SOURCES (God, is hard to live without frontend)

func SetupSourcesPost(ctx *ctx.Context) {
	// Here is where we would retrieve value from a form
	source := models.CreateNewSource(
		ctx.Doer.ID,
		models.GithubStarred,
		"cassiozareck",
		"",
	)

	err := models.SaveSource(ctx, &source)
	if err != nil {
		log.Error("Couldn't save source into database: ", err)
	}

	// todo: this should not call SyncSources
	ss := task.NewSourceSyncer(ctx, ctx.Doer, ctx.Doer)
	err = ss.SyncSources(1)
	if err != nil {
		log.Error("Couldn't sync source", err)
	}

}
