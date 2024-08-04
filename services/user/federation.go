package user

import (
	"context"
	"fmt"
	"time"

	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/optional"
	"code.gitea.io/gitea/services/federation"
)

func UpdatePersonActor(ctx context.Context) error {
	page := 0
	for {
		federatedUsers, err := user.FindFederatedUsers(ctx, page)
		if len(federatedUsers) == 0 {
			break
		}
		if err != nil {
			log.Trace("Error: UpdatePersonActor: %v", err)
			return err
		}

		for _, f := range federatedUsers {
			log.Info("Updating users, got %s", f.ExternalID)
			u, err := user.GetUserByID(ctx, f.UserID)
			if err != nil {
				log.Error("Got error while getting user: %w", err)
				return err
			}

			person, err := federation.GetActor(f.ExternalID)
			if err != nil {
				log.Error("Got error while fetching actor: %w", err)
				return err
			}

			personUrl, err := person.ID.URL()
			if err != nil {
				log.Error("Updating federated users: %w", err)
				return err

			}

			name := fmt.Sprintf("@%v@%v", person.PreferredUsername.String(), personUrl.Host)

			var fullname string
			if len(person.Name) == 0 {
				fullname = name
			} else {
				fullname = person.Name.String()
			}

			if u.Name != name {
				log.Info("Updating username from %s to :%s", u.Name, name)
				err = renameUserWithoutNameCheck(ctx, u, name)
				if err != nil {
					log.Error("Updating federated users: %w", err)
					return err
				}
			}

			if u.FullName != fullname {
				err = UpdateUser(ctx, u,
					&UpdateOptions{
						FullName: optional.Option[string]{fullname},
					})
				if err != nil {
					log.Error("Updating federated users: %w", err)
					return err
				}
			}

			if u.LoginName != name {
				log.Info("Updating loginname")
				err = UpdateAuth(ctx, u, &UpdateAuthOptions{
					LoginName: optional.Option[string]{name},
				})
				if err != nil {
					log.Error("Updating federated users: %w", err)
					return err

				}
			}

			avatar, err := federation.GetPersonAvatar(person)
			if err != nil {
				log.Error("Got error while fetching avatar: %w", err)
				return err
			}

			if u.IsUploadAvatarChanged(avatar) {
				_ = UploadAvatar(ctx, u, avatar)
			}
		}
		page++
	}
	return nil
}

// Clean up remote actors (persons) without any followers in local instance
func CleanUpRemotePersons(ctx context.Context, olderThan time.Duration) error {
	page := 0
	for {
		users, err := user.FindFederatedUserWithNoFollowersAndNoStars(ctx, olderThan, page)
		//      users, err := forgefed.GetRemoteUsersWithNoLocalFollowers(ctx, olderThan, page)
		if len(users) == 0 {
			break
		}
		if err != nil {
			log.Trace("Error: CleanUpRemotePersons: %v", err)
			return err
		}

		for _, u := range users {
			log.Info("Found user %s", u.Name)
			err = DeleteUser(ctx, u, false)
			if err != nil {
				log.Trace("Error: CleanUpRemotePersons: %v", err)
				return err
			}
		}
		page++
	}
	return nil
}
