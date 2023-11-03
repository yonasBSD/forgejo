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
git remote add git@codeberg.org:forgejo/forgejo.git

git checkout forgejo-development
git rebase --onto forgejo/forgejo-development
git push --force

git checkout forgejo-federated-star
git rebase forgejo-development
git push --force
```