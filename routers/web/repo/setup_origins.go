package repo

import (
	"code.gitea.io/gitea/models"
	ctx "code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/services/origins"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// This should follow a hybrid-singleton. It will be created and reused both in fetch and sync operations.
// But it should be set to nil/recreated after the process finish because contexts are ephemeral
var originSyncer *origins.OriginSyncer

// SetupOriginPost saves a new source into the database and returns
func SetupOriginPost(ctx *ctx.Context) {

	// of course retrieve from forms when there is a ui
	originType := models.OriginType(ctx.Params(":type"))
	remoteUsername := ctx.Params(":username")

	// in theory, here we would retrieve values from a form
	err := models.SaveOrigin(ctx, &models.Origin{
		UserID:         ctx.Doer.ID,
		Type:           originType,
		RemoteUsername: remoteUsername,
	})

	if err != nil {
		ctx.Error(http.StatusInternalServerError, "SaveOriginError", fmt.Sprintf("Couldn't save source into database: %v", err))
		return
	}
	ctx.JSON(http.StatusOK, map[string]string{"message": "Source saved successfully."})
}

// CancelOriginSyncer cancels the OriginSyncer if it is in progress.
func CancelOriginSyncerPost(ctx *ctx.Context) {
	if originSyncer == nil {
		ctx.Error(http.StatusForbidden, "OriginSyncerNotInitialized", "OriginSyncer not initialized.")
		return
	}

	if originSyncer.InProgress() {
		originSyncer.Cancel()
		ctx.JSON(http.StatusOK, map[string]string{"message": "OriginSyncer cancelled."})
	} else {
		ctx.JSON(http.StatusOK, map[string]string{"message": "OriginSyncer not in progress. Nothing to cancel."})
	}
}

// FetchAndSyncOrigins handles fetching and syncing operations in one function.
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
		return
	}

	// Respond to the client
	ctx.Data["IncomingRepos"] = originSyncer.GetIncomingRepos() // if the frontend wants he can use this to show every repo being cloned
	ctx.Redirect("/")                                           // in my mind here should be some ui showing these incomin_repos
}

// Utility function to monitor sync progress
func monitorSyncProgress(ctx *ctx.Context, syncer *origins.OriginSyncer) {
	for {
		select {
		case err := <-syncer.Error():
			ctx.Flash.Info(fmt.Sprintf("Error during migration: %v", err))
			return
		case n := <-syncer.Finished():
			ctx.Flash.Info(fmt.Sprintf("Successfully migrated %v repositories", n))
			reset()
			return
		case repo := <-syncer.GetActualMigration():
			ctx.Flash.Info(fmt.Sprintf("Currently migrating: %v", repo))
		case <-time.After(5 * time.Minute):
			ctx.Flash.Info("Timeout reached while waiting for migration events.")
			syncer.Cancel()
			return
		}
		time.Sleep(5 * time.Second) // Add sleep to avoid tight looping.
	}
}

func reset() {
	originSyncer = nil
}
