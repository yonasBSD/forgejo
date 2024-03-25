package federation

import (
	"context"
	"strings"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/setting"
)

type FederatedHost struct {
	ID        int64 `xorm:"pk autoincr"`
	isBlocked bool
	HostFqdn  string `xorm:"UNIQUE(s) INDEX"`
}

func GetFederatdHost(ctx context.Context, hostFqdn string) (*FederatedHost, error) {
	rec := new(FederatedHost)
	_, err := db.GetEngine(ctx).
		Table("federated_host").Where("host_fqdn = ?", hostFqdn).Get(rec)
	if err != nil {
		return nil, err
	}
	return rec, nil
}

func FederatedHostExists(ctx context.Context, hostFqdn string) (bool, error) {
	rec := new(FederatedHost)
	exists, err := db.GetEngine(ctx).
		Table("federated_host").Where("host_fqdn = ?", hostFqdn).Get(rec)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (host *FederatedHost) Save(ctx context.Context) error {
	_, err := db.GetEngine(ctx).
		Insert(host)
	return err
}

type FederatedUser struct {
	ID               int64  `xorm:"pk autoincr"`
	UserID           int64  `xorm:"INDEX"`
	ExternalID       string `xorm:"UNIQUE(s) INDEX"`
	FederationHostID int64  `xorm:"INDEX"`
}

func CreateFederatedUser(ctx context.Context, u *user.User, host *FederatedHost) error {
	engine := db.GetEngine(ctx)

	federatedUser := new(FederatedUser)
	federatedUser.ExternalID = u.Name
	federatedUser.UserID = u.ID
	federatedUser.FederationHostID = host.ID
	_, err := engine.Insert(federatedUser)
	return err
}

func CreatUser(ctx context.Context, u *user.User) error {
	// set system defaults
	u.Visibility = setting.Service.DefaultUserVisibilityMode
	u.AllowCreateOrganization = setting.Service.DefaultAllowCreateOrganization && !setting.Admin.DisableRegularOrgCreation
	u.EmailNotificationsPreference = setting.Admin.DefaultEmailNotification
	u.MaxRepoCreation = -1
	u.Theme = setting.UI.DefaultTheme
	u.IsRestricted = setting.Service.DefaultUserIsRestricted
	u.IsActive = !(setting.Service.RegisterEmailConfirm || setting.Service.RegisterManualConfirm)

	// Ensure consistency of the dates.
	if u.UpdatedUnix < u.CreatedUnix {
		u.UpdatedUnix = u.CreatedUnix
	}

	// validate data
	if err := user.ValidateUser(u); err != nil {
		return err
	}

	if err := user.ValidateEmail(u.Email); err != nil {
		return err
	}

	ctx, committer, err := db.TxContext(ctx)
	if err != nil {
		return err
	}
	defer committer.Close()

	isExist, err := user.IsUserExist(ctx, 0, u.Name)
	if err != nil {
		return err
	} else if isExist {
		return user.ErrUserAlreadyExist{u.Name}
	}

	isExist, err = user.IsEmailUsed(ctx, u.Email)
	if err != nil {
		return err
	} else if isExist {
		return user.ErrEmailAlreadyUsed{
			Email: u.Email,
		}
	}

	// prepare for database

	u.LowerName = strings.ToLower(u.Name)
	u.AvatarEmail = u.Email
	if u.Rands, err = user.GetUserSalt(); err != nil {
		return err
	}
	if u.Passwd != "" {
		if err = u.SetPassword(u.Passwd); err != nil {
			return err
		}
	} else {
		u.Salt = ""
		u.PasswdHashAlgo = ""
	}

	// save changes to database

	if err = user.DeleteUserRedirect(ctx, u.Name); err != nil {
		return err
	}

	if u.CreatedUnix == 0 {
		// Caller expects auto-time for creation & update timestamps.
		err = db.Insert(ctx, u)
	} else {
		// Caller sets the timestamps themselves. They are responsible for ensuring
		// both `CreatedUnix` and `UpdatedUnix` are set appropriately.
		_, err = db.GetEngine(ctx).NoAutoTime().Insert(u)
	}
	if err != nil {
		return err
	}

	// insert email address
	if err := db.Insert(ctx, &user.EmailAddress{
		UID:         u.ID,
		Email:       u.Email,
		LowerEmail:  strings.ToLower(u.Email),
		IsActivated: u.IsActive,
		IsPrimary:   true,
	}); err != nil {
		return err
	}

	return committer.Commit()

}
