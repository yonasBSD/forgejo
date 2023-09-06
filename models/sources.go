package models

const (
	GithubStarred = "github_starred"
)

type Source struct {
	ID             int64  `xorm:"pk autoincr"`
	UserID         string `xorm:"UNIQUE NOT NULL"`
	Type           string `xorm:"NOT NULL"`
	RemoteUsername string `xorm:"NOT NULL"`
	Token          string
}

func GetSourcesByUser(userId int64) []Source {
	return []Source{
		{Type: GithubStarred, RemoteUsername: "cassiozareck"},
	} //todo
}
