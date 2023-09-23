package repo

import (
	"code.gitea.io/gitea/models"
	ctx "code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/services/origins"
	"fmt"
	"time"
)

// TODO: hello frontender here you will can work on UI and forms retrieve

// SetupSourcesPost save a new source into database
func SetupSourcesPost(ctx *ctx.Context) {
	err := models.SaveOrigin(ctx, &models.Origin{
		UserID:         ctx.Doer.ID,
		Type:           models.GithubStarred,
		RemoteUsername: "cassiozareck",
		Token:          ""})

	if err != nil {
		log.Error("Couldn't save source into database: ", err)
	}
}

var originSyncer *origins.OriginSyncer

// SyncOriginsPost synchronizes the origins
func SyncOriginsPost(ctx *ctx.Context) {
	if originSyncer == nil {
		originSyncer = origins.NewOriginSyncer(ctx, ctx.Doer, ctx.Doer, 20)
	}
	if originSyncer.InProgress() {
		log.Error("Origins synchronization already in progress")
	}

	err := originSyncer.Fetch()
	if err != nil {
		log.Error("Couldn't fetch origins", err)
	}

	err = originSyncer.Sync()
	if err != nil {
		log.Error("Couldn't sync origins", err)
	}

	// Flash a message saying origins are being synced and flash when it's done
	go func() {
		time.Sleep(50 * time.Millisecond) // Guarantee it is already working on a repository
		for {
			select {
			case err := <-originSyncer.Error():
				ctx.Flash.Error(fmt.Sprintf("Error during migration: %v", err))
				return
			default:
				if !originSyncer.InProgress() {
					ctx.Flash.Info("Origins synced")
					return
				}
				ctx.Flash.Info(fmt.Sprintf("Currently migrating: %v", originSyncer.GetActualMigration()))
				time.Sleep(5 * time.Second) // Add sleep to avoid tight looping.
			}
		}
	}()
}
