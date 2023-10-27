TAGS="sqlite" make build generate-swagger
./gitea admin user create --name me --password me --email "buero@meissa.de"
./gitea admin user generate-access-token -u me -t token --scopes write:activitypub,write:repository,write:user
