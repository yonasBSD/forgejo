package models

import (
	"code.gitea.io/gitea/models/db"
	"context"
)

type SourceType string

const (
	GithubStarred = "github_starred"
	Dummy         = "dummy" // just for tests
)

type Source struct {
	ID             int64      `xorm:"pk autoincr"`
	UserID         int64      `xorm:"index UNIQUE NOT NULL"`
	Type           SourceType `xorm:"NOT NULL"`
	RemoteUsername string     `xorm:"NOT NULL"`
	Token          string
}

func init() {
	db.RegisterModel(new(Source))
}

// SaveSource insert a new source into the database
func SaveSource(ctx context.Context, source *Source) error {
	return db.Insert(ctx, source)
}

func GetSourcesByUserID(ctx context.Context, userID int64) ([]Source, error) {
	var sources []Source
	err := db.GetEngine(ctx).
		Table("source").
		Where("user_id = ?", userID).
		Find(&sources)

	return sources, err
}
