# enable federation

copy the app.ini in this folder in custom/conf in the forgejo root directory.
Then change the paths in app.ini accordingly to you local environment.

```
; ;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;
; ;
; ; SQLite Configuration
; ;
DB_TYPE = sqlite3
; defaults to data/gitea.db
PATH = /home/jem/repo/opensource/forgejo/data/gitea.db
; Query timeout defaults to: 500
SQLITE_TIMEOUT = 
; defaults to sqlite database default (often DELETE), can be used to enable WAL mode. https://www.sqlite.org/pragma.html#pragma_journal_mode
SQLITE_JOURNAL_MODE = 
HOST = 
NAME = 
USER = 
PASSWD = 
SCHEMA = 
SSL_MODE = disable
LOG_SQL = false


[federation]
ENABLED = true
```

# build

```
TAGS="sqlite" make build generate-swagger
```

# launch local

```
./gitea admin user create --name me --password me --email "buero@meissa.de"
./gitea admin user generate-access-token -u me -t token --scopes write:activitypub,write:repository,write:user
```

# sync base branch

```
# setup a second repo for excosy implementation
git clone https://git.exozy.me/a/gitea.git exosy

# add remotes
git remote add forgejo git@codeberg.org:forgejo/forgejo.git

# rebase on top of forgejo/forge-development
git checkout forgejo-development
git fetch forgejo
git rebase --onto forgejo/forgejo-development
git push --force

git checkout forgejo-federated-star
git rebase forgejo-development
git push --force

# continue local development after rebase & force-push has happened
git reset --hard origin/forgejo-federated-star
```

# generate swagger api client

go run github.com/go-swagger/go-swagger/cmd/swagger@v0.30.5 generate client -f './templates/swagger/v1_json.tmpl' -c "code.gitea.io/sdk" --operation 'activitypubPerson' --skip-validation

