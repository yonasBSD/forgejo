package user

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/log"
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
			u, err := user.GetUserByID(ctx, f.ID)
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
			u.LoginName = person.PreferredUsername.String()
			u.Name = fmt.Sprintf("@%v@%v", person.PreferredUsername.String(), personUrl.Host)
			if len(person.Name) == 0 {
				u.FullName = u.Name
			} else {
				u.FullName = person.Name.String()
			}

			opts := UpdateOptions{}
			UpdateUser(ctx, u, &opts)

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
