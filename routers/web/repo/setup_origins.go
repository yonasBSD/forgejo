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

// TODO: hello frontender here you will can work on UI and forms retrieve

var originSyncer *origins.OriginSyncer

// SetupOriginPost saves a new source into the database and returns relevant responses.
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

// FetchedRepositories returns new repositories found under chosen origins
func FetchedRepositories(ctx *ctx.Context) {
	if originSyncer == nil {
		ctx.Error(http.StatusForbidden, "OriginSyncerNotInitialized", "OriginSyncer not initialized.")
		return
	}

	if originSyncer.InProgress() {
		repos := originSyncer.GetIncomingRepos()
		ctx.JSON(http.StatusOK, repos)
	} else {
		ctx.JSON(http.StatusOK, map[string]string{"message": "OriginSyncer not in synchronization process."})
	}
}

// CancelOriginSyncer cancels the OriginSyncer if it is in progress.
func CancelOriginSyncer(ctx *ctx.Context) {
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

// SyncOriginsPost synchronizes the origins and returns relevant responses.
func SyncOriginsPost(ctx *ctx.Context) {
	if originSyncer != nil && originSyncer.InProgress() {
		ctx.Error(http.StatusBadRequest, "SyncInProgressError", "Origins synchronization already in progress")
		return
	}

	originSyncer = origins.NewOriginSyncer(ctx, ctx.Doer, ctx.ContextUser, 20)

	err := originSyncer.Fetch()
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "FetchOriginsError", fmt.Sprintf("Couldn't fetch origins: %v", err))
		return
	}

	err = originSyncer.Sync()
	if err != nil {
		ctx.Error(500, "SyncOriginsError", fmt.Sprintf("Couldn't sync origins: %v", err))
		return
	}

	go func() {
		for {
			select {
			case err := <-originSyncer.Error():
				log.Error(fmt.Sprintf("Error during migration: %v", err))
				return
			case n := <-originSyncer.Finished():
				log.Info(fmt.Sprintf("Successfully migrated %v repositories", n))
				return
			case repo := <-originSyncer.GetActualMigration():
				log.Info(fmt.Sprintf("Currently migrating: %v", repo))
			case <-time.After(1 * time.Minute):
				log.Warn("Timeout reached while waiting for migration events.")
				return
			}
			time.Sleep(5 * time.Second) // Add sleep to avoid tight looping.
		}
	}()
	ctx.JSON(http.StatusOK, map[string]string{"message": "Origins synchronization initiated."})
}
