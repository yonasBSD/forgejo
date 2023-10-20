package repo

import (
	"code.gitea.io/gitea/models"
	ctx "code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/services/origins"
	"fmt"
	"net/http"
	"strconv"
)

// This should follow a hybrid-singleton. It will be created and reused both in fetch and sync operations.
// But it should be set to nil/recreated after the process finish because contexts are ephemeral
var originSyncer *origins.OriginSyncer

type SetupOriginForm struct {
	Type           string `json:"type" binding:"Required"`
	RemoteUsername string `json:"remote_username" binding:"Required"`
}

// SetupOriginPost saves a new origin into the database and returns
func SetupOriginPost(ctx *ctx.Context) {

	form := web.GetForm(ctx).(*SetupOriginForm)

	err := models.SaveOrigin(ctx, &models.Origin{
		UserID:         ctx.Doer.ID,
		Type:           models.OriginType(form.Type),
		RemoteUsername: form.RemoteUsername,
		Token:          "",
	})

	if err != nil {
		ctx.Error(http.StatusInternalServerError, "SaveOriginError", fmt.Sprintf("Couldn't save source into database: %v", err))
		return
	}
	ctx.Flash.Info("Saved")
	ctx.Redirect("/")
}

// CancelOriginSyncer cancels the OriginSyncer if it is in progress.
func CancelOriginSyncerPost(ctx *ctx.Context) {
	if originSyncer == nil {
		ctx.Error(http.StatusForbidden, "OriginSyncerNotInitialized", "OriginSyncer not initialized.")
		return
	}

	if originSyncer.InProgress() {
		originSyncer.Cancel()
		ctx.Flash.Info("Canceled")
		ctx.Redirect("/")
	} else {
		ctx.Flash.Info("Nothing in progress")
		ctx.Redirect("/")
	}
}

// FetchAndSyncOrigins get and mirror repositories from chosen origins like starred repositories from
// github and others
func FetchAndSyncOrigins(ctx *ctx.Context) {
	// Check if an origin sync is already in progress
	if originSyncer != nil && originSyncer.InProgress() {
		ctx.Error(http.StatusBadRequest, "SyncInProgressError", "Origins synchronization already in progress")
		return
	}
	limit, err := strconv.Atoi(ctx.Params(":limit")) // Convert string to int
	if err != nil {
		ctx.Error(http.StatusBadRequest, "InvalidLimitError", fmt.Sprintf("Invalid limit value: %v", err))
		return
	}
	// Create a new OriginSyncer
	originSyncer = origins.NewOriginSyncer(ctx, ctx.Doer, ctx.Doer, limit)

	// Fetch origins
	err = originSyncer.Fetch()
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "FetchOriginsError", fmt.Sprintf("Couldn't fetch origins: %v", err))
		return
	}

	// Sync origins
	err = originSyncer.Sync()
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "SyncOriginsError", fmt.Sprintf("Couldn't sync origins: %v", err))
		originSyncer.Cancel()
		return
	}

	// Respond to the client
	ctx.Data["IncomingRepos"] = originSyncer.GetIncomingRepos() // use this to show every repo being cloned
	ctx.HTML(http.StatusOK, "repo/fetch_origins")
}

func reset() {
	originSyncer = nil
}
