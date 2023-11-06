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
git rebase --onto forgejo/forgejo-development
git push --force

git checkout forgejo-federated-star
git rebase forgejo-development
git push --force

# continue local development after rebase & force-push has happened
git reset --hard origin/forgejo-federated-star
```
