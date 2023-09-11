package models

import (
	"code.gitea.io/gitea/models/db"
	"context"
)

const (
	GithubStarred = "github_starred"
)

type Source struct {
	ID             int64  `xorm:"pk autoincr"`
	UserID         int64  `xorm:"index UNIQUE NOT NULL"`
	Type           string `xorm:"NOT NULL"`
	RemoteUsername string `xorm:"NOT NULL"`
	Token          string
}

func init() {
	db.RegisterModel(new(Source))
}

func CreateNewSource(userId int64, typ, remoteUser, token string) Source {
	return Source{
		UserID:         userId,
		Type:           typ,
		RemoteUsername: remoteUser,
		Token:          token,
	}
}

// SaveSource insert a new source into the database
func SaveSource(ctx context.Context, source *Source) error {
	return db.Insert(ctx, source)
}

func GetSourcesByUserID(ctx context.Context, userID int64) ([]Source, error) {
	var sources []Source
	// SELECT * FROM sources WHERE id = ?
	err := db.GetEngine(ctx).
		Table("source").
		Where("id = ?", userID).
		Find(&sources)

	return sources, err
}
