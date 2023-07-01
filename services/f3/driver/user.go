// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"

	auth_model "code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/models/db"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/util"
	user_service "code.gitea.io/gitea/services/user"

	"lab.forgefriends.org/friendlyforgeformat/gof3/forges/common"
	"lab.forgefriends.org/friendlyforgeformat/gof3/format"
	f3_util "lab.forgefriends.org/friendlyforgeformat/gof3/util"
)

type User struct {
	user_model.User
}

func UserConverter(f *user_model.User) *User {
	return &User{
		User: *f,
	}
}

func (o User) GetID() int64 {
	return o.ID
}

func (o User) GetIDString() string {
	return fmt.Sprintf("%d", o.GetID())
}

func (o *User) SetID(id int64) {
	o.ID = id
}

func (o *User) SetIDString(id string) {
	o.SetID(f3_util.ParseInt(id))
}

func (o *User) IsNil() bool {
	return o.ID == 0
}

func (o *User) Equals(other *User) bool {
	return (o.ID == other.ID)
}

func (o *User) ToFormatInterface() format.Interface {
	return o.ToFormat()
}

func (o *User) ToFormat() *format.User {
	return &format.User{
		Common:   format.NewCommon(o.ID),
		UserName: o.Name,
		Name:     o.FullName,
		Email:    o.Email,
		Password: o.Passwd,
	}
}

func (o *User) FromFormat(user *format.User) {
	*o = User{
		User: user_model.User{
			ID:       user.Index.GetID(),
			Name:     user.UserName,
			FullName: user.Name,
			Email:    user.Email,
			Passwd:   user.Password,
		},
	}
}

type UserProvider struct {
	BaseProvider
}

func getLocalMatchingRemote(ctx context.Context, authenticationSource int64, id string) *user_model.User {
	u := &user_model.User{
		LoginName:   id,
		LoginSource: authenticationSource,
		LoginType:   auth_model.OAuth2,
		Type:        user_model.UserTypeIndividual,
	}
	has, err := db.GetEngine(ctx).Get(u)
	if err != nil {
		panic(err)
	} else if !has {
		return nil
	}
	return u
}

func (o *UserProvider) GetLocalMatchingRemote(ctx context.Context, format format.Interface, parents ...common.ContainerObjectInterface) (string, bool) {
	authenticationSource := o.g.GetAuthenticationSource()
	if authenticationSource == 0 {
		return "", false
	}
	user := getLocalMatchingRemote(ctx, authenticationSource, format.GetIDString())
	if user != nil {
		o.g.GetLogger().Debug("found existing user %d with a matching authentication source for %s", user.ID, format.GetIDString())
		return fmt.Sprintf("%d", user.ID), true
	}
	o.g.GetLogger().Debug("no pre-existing local user for %s", format.GetIDString())
	return "", false
}

func (o *UserProvider) ToFormat(ctx context.Context, user *User) *format.User {
	return user.ToFormat()
}

func (o *UserProvider) FromFormat(ctx context.Context, p *format.User) *User {
	var user User
	user.FromFormat(p)
	return &user
}

func (o *UserProvider) GetObjects(ctx context.Context, page int) []*User {
	users, _, err := user_model.SearchUsers(&user_model.SearchUserOptions{
		Actor:       o.g.GetDoer(),
		Type:        user_model.UserTypeIndividual,
		ListOptions: db.ListOptions{Page: page, PageSize: o.g.perPage},
	})
	if err != nil {
		panic(fmt.Errorf("error while listing users: %v", err))
	}
	return f3_util.ConvertMap[*user_model.User, *User](users, UserConverter)
}

func (o *UserProvider) ProcessObject(ctx context.Context, user *User) {
}

func (o *UserProvider) Get(ctx context.Context, exemplar *User) *User {
	o.g.GetLogger().Debug("%+v", exemplar)
	var user *user_model.User
	var err error
	if exemplar.GetID() > 0 {
		user, err = user_model.GetUserByID(ctx, exemplar.GetID())
		o.g.GetLogger().Debug("GetUserByID: %+v %v", user, err)
	} else if exemplar.Name != "" {
		user, err = user_model.GetUserByName(ctx, exemplar.Name)
	} else {
		panic("GetID() == 0 and UserName == \"\"")
	}
	if err != nil {
		if user_model.IsErrUserNotExist(err) {
			return &User{}
		}
		panic(fmt.Errorf("user %v %w", exemplar, err))
	}
	return UserConverter(user)
}

func (o *UserProvider) Put(ctx context.Context, user *User) *User {
	overwriteDefault := &user_model.CreateUserOverwriteOptions{
		IsActive: util.OptionalBoolTrue,
	}
	u := user_model.User{
		Name:     user.Name,
		FullName: user.FullName,
		Email:    user.Email,
		Passwd:   user.Passwd,
	}
	err := user_model.CreateUser(&u, overwriteDefault)
	if err != nil {
		panic(err)
	}
	return o.Get(ctx, UserConverter(&u))
}

func (o *UserProvider) Delete(ctx context.Context, user *User) *User {
	u := o.Get(ctx, user)
	if !u.IsNil() {
		if err := user_service.DeleteUser(ctx, &user.User, true); err != nil {
			panic(err)
		}
	}
	return u
}
