package repo

import (
	"code.gitea.io/gitea/models"
	ctx "code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/services/origins"
	"fmt"
	"net/http"
	"time"
)

const MIRROR_LIMIT = 50 // The user may not want mirror too much at single time

// This should follow a hybrid-singleton. It will be created and reused both in fetch and sync operations.
// But it should be set to nil/recreated after the process finish because contexts are ephemeral
var originSyncer *origins.OriginSyncer

// SetupOriginPost saves a new source into the database and returns
func SetupOriginPost(ctx *ctx.Context) {
	err := models.SaveOrigin(ctx, &models.Origin{
		UserID:         ctx.Doer.ID,
		Type:           models.GithubStarred,
		RemoteUsername: "cassiozareck",
		Token:          ""})

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

// FetchOriginsPost fetches the origins without syncing.
func FetchOriginsPost(ctx *ctx.Context) {
	if originSyncer != nil && originSyncer.InProgress() {
		ctx.Error(http.StatusBadRequest, "FetchInProgressError", "Origins fetching already in progress")
		return
	}

	originSyncer = origins.NewOriginSyncer(ctx, ctx.Doer, ctx.ContextUser, MIRROR_LIMIT)

	err := originSyncer.Fetch()
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "FetchOriginsError", fmt.Sprintf("Couldn't fetch origins: %v", err))
		return
	}
	//todo write fetched repositories in response
	ctx.JSON(http.StatusOK, map[string]string{"message": "Origins fetching initiated."})
}

// SyncOriginsPost syncs the fetched origins.
func SyncOriginsPost(ctx *ctx.Context) {
	if originSyncer == nil {
		ctx.Error(http.StatusBadRequest, "SyncError", "Origins have not been fetched yet.")
		return
	}

	if originSyncer.InProgress() {
		ctx.Error(http.StatusBadRequest, "SyncInProgressError", "Origins synchronization already in progress")
		return
	}

	err := originSyncer.Sync()
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "SyncOriginsError", fmt.Sprintf("Couldn't sync origins: %v", err))
		return
	}

	go monitorSyncProgress(originSyncer)
	ctx.JSON(http.StatusOK, map[string]string{"message": "Origins synchronization initiated."})
}

// Utility function to monitor sync progress
func monitorSyncProgress(syncer *origins.OriginSyncer) {
	for {
		select {
		case err := <-syncer.Error():
			log.Error(fmt.Sprintf("Error during migration: %v", err))
			return
		case n := <-syncer.Finished():
			log.Info(fmt.Sprintf("Successfully migrated %v repositories", n))
			reset()
			return
		case repo := <-syncer.GetActualMigration():
			log.Info(fmt.Sprintf("Currently migrating: %v", repo))
		case <-time.After(5 * time.Minute):
			log.Warn("Timeout reached while waiting for migration events.")
			syncer.Cancel()
			return
		}
		time.Sleep(5 * time.Second) // Add sleep to avoid tight looping.
	}
}

func reset() {
	originSyncer = nil
}
