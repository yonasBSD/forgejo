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

```bash
# cleanup
./gitea admin user delete --purge -id 8
./gitea admin user delete --purge -id 9
./gitea admin user delete --purge -id 10

# create a user
./gitea admin user create --username me --password me --email "buero@meissa.de" --admin
./gitea admin user create --username stargoose1 --random-password --email "stargoose1@meissa.de"
./gitea admin user create --username stargoose2 --random-password --email "stargoose2@meissa.de"
./gitea admin user create --username stargoose3 --random-password --email "stargoose3@meissa.de"
./gitea admin user create --username stargoose4 --random-password --email "stargoose4@meissa.de"
./gitea admin user create --username stargoose5 --random-password --email "stargoose5@meissa.de"
./gitea admin user create --username stargoose6 --random-password --email "stargoose6@meissa.de"
./gitea admin user create --username stargoose7 --random-password --email "stargoose7@meissa.de"
./gitea admin user create --username stargoose8 --random-password --email "stargoose8@meissa.de"
./gitea admin user create --username stargoose9 --random-password --email "stargoose9@meissa.de"
./gitea admin user create --username stargoose10 --random-password --email "stargoose10@meissa.de"
./gitea admin user create --username stargoose11 --random-password --email "stargoose11@meissa.de"
./gitea admin user create --username stargoose12 --random-password --email "stargoose12@meissa.de"
./gitea admin user list

# create a token
./gitea admin user generate-access-token -u me -t token --scopes write:activitypub,write:repository,write:user

# create a repo
```bash
curl -X 'POST' \
  'http://localhost:3000/api/v1/user/repos?token=ReplaceThisWithYourGeneratedToken' \
  -H 'accept: application/json' \
  -H 'Content-Type: application/json' \
  -d '{
  "auto_init": false,
  "default_branch": "main",
  "description": "none",
  "gitignores": "none",
  "issue_labels": "",
  "license": "apache",
  "name": "repo",
  "private": true,
  "readme": "This is a readme",
  "template": false,
  "trust_model": "default"
}'
```

# Datastructures handy for local tests

## Star activity

```json
{
  "id": "http://localhost:3000/api/v1/activitypub/user-id/1/outbox/12345",
  "type": "Like",
  "actor": "https://federated-repo.prod.meissa.de/api/v1/activitypub/user-id/13",
  "object": "http://localhost:3000/api/v1/activitypub/repository-id/2",
  "startTime": "2014-12-31T23:00:00-08:00"
}
```

# sync base branch

``` bash
# setup a second repo for excosy implementation
git clone https://git.exozy.me/a/gitea.git exosy

# add remotes
git remote add forgejo git@codeberg.org:forgejo/forgejo.git

# rebase on top of forgejo/forge-development
git switch forgejo-development
git fetch forgejo
git reset --hard origin/forgejo-development
git push --force

git switch forgejo-federated-star
git rebase forgejo-development
git push --force

# continue local development after rebase & force-push has happened
git reset --hard origin/forgejo-federated-star
```

# provide testinstance

``` bash
git switch test-release
git rebase --onto forgejo-federated-star
git merge forgejo/forgejo-branding
git push --force
```

# generate swagger api client

go run github.com/go-swagger/go-swagger/cmd/swagger@v0.30.5 generate client -f './templates/swagger/v1_json.tmpl' -c "modules/activitypub2" --operation 'activitypubPerson' --skip-models --existing-models 'github.com/go-ap/activitypub' --skip-validation

# Documentation for learn & reference


# Thoughts on testing

I would like to be able to quickly test a change in the repo code.
For that i need:
A test server with federation enabled
A test user
A test repo
A test auth token (?)

A test request as input value to the API
An expected result for comparison with the output value.

Tests that provide some examples are:

tests/integration/api_activitypub_person_test.go

tests/integration/api_token_test.go

maybe tests/integration/api_repo_test.go
