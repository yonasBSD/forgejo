// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"

	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/util"
	user_service "code.gitea.io/gitea/services/user"

	"lab.forgefriends.org/friendlyforgeformat/gof3/f3"
	f3_tree "lab.forgefriends.org/friendlyforgeformat/gof3/tree/f3"
	"lab.forgefriends.org/friendlyforgeformat/gof3/tree/generic"
	f3_util "lab.forgefriends.org/friendlyforgeformat/gof3/util"
)

var _ f3_tree.ForgeDriverInterface = &user{}

type user struct {
	common

	forgejoUser *user_model.User
}

func (o *user) SetNative(user any) {
	o.forgejoUser = user.(*user_model.User)
}

func (o *user) GetNativeID() string {
	return fmt.Sprintf("%d", o.forgejoUser.ID)
}

func (o *user) NewFormat() f3.Interface {
	node := o.GetNode()
	return node.GetTree().(f3_tree.TreeInterface).NewFormat(node.GetKind())
}

func (o *user) ToFormat() f3.Interface {
	if o.forgejoUser == nil {
		return o.NewFormat()
	}
	return &f3.User{
		Common:   f3.NewCommon(fmt.Sprintf("%d", o.forgejoUser.ID)),
		UserName: o.forgejoUser.Name,
		Name:     o.forgejoUser.FullName,
		Email:    o.forgejoUser.Email,
		IsAdmin:  o.forgejoUser.IsAdmin,
		Password: o.forgejoUser.Passwd,
	}
}

func (o *user) FromFormat(content f3.Interface) {
	user := content.(*f3.User)
	o.forgejoUser = &user_model.User{
		Type:     user_model.UserTypeF3,
		ID:       f3_util.ParseInt(user.GetID()),
		Name:     user.UserName,
		FullName: user.Name,
		Email:    user.Email,
		IsAdmin:  user.IsAdmin,
		Passwd:   user.Password,
	}
}

func (o *user) Get(ctx context.Context) bool {
	node := o.GetNode()
	o.Trace("%s", node.GetID())
	id := f3_util.ParseInt(string(node.GetID()))
	u, err := user_model.GetUserByID(ctx, id)
	if user_model.IsErrUserNotExist(err) {
		return false
	}
	if err != nil {
		panic(fmt.Errorf("user %v %w", id, err))
	}
	o.forgejoUser = u
	return true
}

func (o *user) Patch(context.Context) {
}

func (o *user) Put(ctx context.Context) generic.NodeID {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	overwriteDefault := &user_model.CreateUserOverwriteOptions{
		IsActive: util.OptionalBoolTrue,
	}
	err := user_model.CreateUser(ctx, o.forgejoUser, overwriteDefault)
	if err != nil {
		panic(err)
	}

	return generic.NodeID(fmt.Sprintf("%d", o.forgejoUser.ID))
}

func (o *user) Delete(ctx context.Context) {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	if err := user_service.DeleteUser(ctx, o.forgejoUser, true); err != nil {
		panic(err)
	}
}

func newUser() generic.NodeDriverInterface {
	return &user{}
}
