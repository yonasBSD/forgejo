// Package origins SPDX-License-Identifier: MIT

package models

import (
	"code.gitea.io/gitea/models/db"
	"context"
)

type OriginType string

const (
	GithubStarred = "github_starred"
	Dummy         = "dummy" // just for tests
)

func (ot OriginType) GetName() string {
	switch ot {
	case GithubStarred:
		return "Github Starred Repositories"
	case Dummy:
		return "Dummy"
	}
	return "Not valid type"
}

type Origin struct {
	ID             int64      `xorm:"pk autoincr"`
	UserID         int64      `xorm:"index NOT NULL"`
	Type           OriginType `xorm:"NOT NULL"`
	RemoteUsername string     `xorm:"NOT NULL"`
	Token          string
}

func init() {
	db.RegisterModel(new(Origin))
}

// SaveOrigin inserts a new origin into the database.
func SaveOrigin(ctx context.Context, origin *Origin) error {
	return db.Insert(ctx, origin)
}

// GetOriginsByUserID retrieves all origins associated with a given user ID.
func GetOriginsByUserID(ctx context.Context, userID int64) ([]Origin, error) {
	var origins []Origin
	err := db.GetEngine(ctx).
		Table("origin").
		Where("user_id = ?", userID).
		Find(&origins)

	return origins, err
}
