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

var originSyncer *origins.OriginSyncer

type SetupOriginForm struct {
	Type           string `json:"type" binding:"Required"`
	RemoteUsername string `json:"remote_username" binding:"Required"`
}

// SetupOriginPost saves a new origin into the database and returns
func SetupOriginPost(ctx *ctx.APIContext) {
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
	ctx.JSON(http.StatusOK, map[string]string{"message": "Origin saved successfully."})
}

// CancelOriginSyncer cancels the OriginSyncer if it is in progress.
func CancelOriginSyncerPost(ctx *ctx.APIContext) {
	if originSyncer == nil {
		ctx.Error(http.StatusForbidden, "OriginSyncerNotInitialized", "OriginSyncer not initialized.")
		return
	}

	if originSyncer.InProgress() {
		originSyncer.Cancel()
		ctx.JSON(http.StatusOK, map[string]string{"message": "Origin syncer canceled successfully."})
	} else {
		ctx.JSON(http.StatusOK, map[string]string{"message": "Origin syncer not in progress."})
	}
}

// FetchAndSyncOrigins get and mirror repositories from chosen origins like starred repositories from
// github, codeberg and others
func FetchAndSyncOrigins(ctx *ctx.APIContext) {
	// Check if an origin sync is already in progress
	if originSyncer.InProgress() {
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

	// Respond to the client with the list of incoming repositories
	ctx.JSON(http.StatusOK, map[string]interface{}{
		"IncomingRepos": originSyncer.GetIncomingRepos(),
	})
}
