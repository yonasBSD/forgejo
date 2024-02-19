package repo

import (
	"net/http"

	"code.gitea.io/gitea/modules/context"
	repo_service "code.gitea.io/gitea/services/repository"
)

// SyncForkBranch syncs a fork branch with the base branch
func SyncForkBranch(ctx *context.APIContext) {
	// swagger:operation POST /repos/{owner}/{repo}/sync_fork/{branch} repository repoSyncForkBranch
	// ---
	// summary: Syncs a fork
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: branch
	//   in: path
	//   description: The branch
	//   type: string
	//   required: true
	// responses:
	//   "400":
	//     "$ref": "#/responses/error"
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "404":
	//     "$ref": "#/responses/notFound"
	if !ctx.Repo.Repository.IsFork {
		ctx.Error(http.StatusBadRequest, "NoFork", "The Repo must be a fork")
		return
	}

	branch := ctx.Params("branch")

	err := repo_service.SyncFork(ctx, ctx.Doer, ctx.Repo.Repository, branch)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "DeleteReleaseByID", err)
		return
	}

	ctx.Status(http.StatusNoContent)
}
