// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"fmt"
	"net/url"
	"strings"

	"code.gitea.io/gitea/models/forgefed"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/util"
	"code.gitea.io/gitea/modules/validation"
	"github.com/google/uuid"
	pwd_gen "github.com/sethvargo/go-password/password"
)

func CreateFederatedUserFromAP(ctx *context.APIContext, person forgefed.ForgePerson, personID forgefed.PersonID,
	federationHostID int64) (*User, error) {
	if res, err := validation.IsValid(person); !res {
		return nil, err
	}
	log.Info("RepositoryInbox: validated person: %q", person)

	localFqdn, err := url.ParseRequestURI(setting.AppURL)
	if err != nil {
		return nil, err
	}

	email := fmt.Sprintf("f%v@%v", uuid.New().String(), localFqdn.Hostname())
	loginName := personID.AsLoginName()
	name := fmt.Sprintf("%v%v", person.PreferredUsername.String(), personID.HostSuffix())
	log.Info("RepositoryInbox: person.Name: %v", person.Name)
	fullName := person.Name.String()
	if len(person.Name) == 0 {
		fullName = name
	}

	password, err := pwd_gen.Generate(32, 10, 10, false, true)
	if err != nil {
		return nil, err
	}

	user := &User{
		LowerName:                    strings.ToLower(person.PreferredUsername.String()),
		Name:                         name,
		FullName:                     fullName,
		Email:                        email,
		EmailNotificationsPreference: "disabled",
		Passwd:                       password,
		MustChangePassword:           false,
		LoginName:                    loginName,
		Type:                         UserTypeRemoteUser,
		IsAdmin:                      false,
	}

	overwrite := &CreateUserOverwriteOptions{
		IsActive:     util.OptionalBoolFalse,
		IsRestricted: util.OptionalBoolFalse,
	}

	if err := CreateUser(ctx, user, overwrite); err != nil {
		return nil, err
	}

	federatedUser, err := NewFederatedUser(user.ID, personID.ID, federationHostID)
	if err != nil {
		return nil, err
	}

	err = CreateFederationUser(ctx, federatedUser)
	if err != nil {
		return nil, err
	}

	return user, nil
}
