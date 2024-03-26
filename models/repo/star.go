// Copyright 2016 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"context"

	"code.gitea.io/gitea/models/db"
	//"code.gitea.io/gitea/models/forgefed"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/timeutil"
)

// Star represents a starred repo by an user.
type Star struct {
	ID          int64              `xorm:"pk autoincr"`
	UID         int64              `xorm:"UNIQUE(s)"`
	RepoID      int64              `xorm:"UNIQUE(s)"`
	CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
}

func init() {
	db.RegisterModel(new(Star))
}

// TODO: why is this ctx not type *context.APIContext, as in routers/api/v1/user/star.go?
// StarRepo or unstar repository.
func StarRepo(ctx context.Context, userID, repoID int64, star bool) error {
	ctx, committer, err := db.TxContext(ctx)
	if err != nil {
		return err
	}
	defer committer.Close()
	staring := IsStaring(ctx, userID, repoID)

	if star {
		if staring {
			return nil
		}

		if err := db.Insert(ctx, &Star{UID: userID, RepoID: repoID}); err != nil {
			return err
		}
		if _, err := db.Exec(ctx, "UPDATE `repository` SET num_stars = num_stars + 1 WHERE id = ?", repoID); err != nil {
			return err
		}
		if _, err := db.Exec(ctx, "UPDATE `user` SET num_stars = num_stars + 1 WHERE id = ?", userID); err != nil {
			return err
		}
		if err := SendLikeActivities(ctx, userID); err != nil {
			return err
		}
	} else {
		if !staring {
			return nil
		}

		if _, err := db.DeleteByBean(ctx, &Star{UID: userID, RepoID: repoID}); err != nil {
			return err
		}
		if _, err := db.Exec(ctx, "UPDATE `repository` SET num_stars = num_stars - 1 WHERE id = ?", repoID); err != nil {
			return err
		}
		if _, err := db.Exec(ctx, "UPDATE `user` SET num_stars = num_stars - 1 WHERE id = ?", userID); err != nil {
			return err
		}
	}

	return committer.Commit()
}

// TODO: solve circular dependencies caused by import of forgefed!
func SendLikeActivities(ctx context.Context, userID int64) error {
	if setting.Federation.Enabled {

		/*likeActivity, err := forgefed.NewForgeLike(ctx)
		if err != nil {
			return err
		}

		json, err := likeActivity.MarshalJSON()
		if err != nil {
			return err
		}

		// TODO: is user loading necessary here?
		user, err := user_model.GetUserByID(ctx, userID)
		if err != nil {
			return err
		}

		apclient, err := activitypub.NewClient(ctx, user, user.APAPIURL())
		if err != nil {
			return err
		}

		// TODO: set timeouts for outgoing request in oder to mitigate DOS by slow lories
		// ToDo: Change this to the standalone table of FederatedRepos
		for _, target := range strings.Split(ctx.Repo.Repository.FederationRepos, ";") {
			apclient.Post([]byte(json), target)
		}
		*/
		// Send to list of federated repos
	}
	return nil
}

// IsStaring checks if user has starred given repository.
func IsStaring(ctx context.Context, userID, repoID int64) bool {
	has, _ := db.GetEngine(ctx).Get(&Star{UID: userID, RepoID: repoID})
	return has
}

// GetStargazers returns the users that starred the repo.
func GetStargazers(ctx context.Context, repo *Repository, opts db.ListOptions) ([]*user_model.User, error) {
	sess := db.GetEngine(ctx).Where("star.repo_id = ?", repo.ID).
		Join("LEFT", "star", "`user`.id = star.uid")
	if opts.Page > 0 {
		sess = db.SetSessionPagination(sess, &opts)

		users := make([]*user_model.User, 0, opts.PageSize)
		return users, sess.Find(&users)
	}

	users := make([]*user_model.User, 0, 8)
	return users, sess.Find(&users)
}

// ClearRepoStars clears all stars for a repository and from the user that starred it.
// Used when a repository is set to private.
func ClearRepoStars(ctx context.Context, repoID int64) error {
	if _, err := db.Exec(ctx, "UPDATE `user` SET num_stars=num_stars-1 WHERE id IN (SELECT `uid` FROM `star` WHERE repo_id = ?)", repoID); err != nil {
		return err
	}

	if _, err := db.Exec(ctx, "UPDATE `repository` SET num_stars = 0 WHERE id = ?", repoID); err != nil {
		return err
	}

	return db.DeleteBeans(ctx, Star{RepoID: repoID})
}
