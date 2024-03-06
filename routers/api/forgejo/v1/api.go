// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1

import (
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/api/shared"
	"code.gitea.io/gitea/routers/common"
	"code.gitea.io/gitea/services/context"
	"github.com/go-chi/cors"
)

func Routes() *web.Route {
	m := web.NewRoute()

	m.Use(shared.SecurityHeaders())
	if setting.CORSConfig.Enabled {
		m.Use(cors.Handler(cors.Options{
			AllowedOrigins:   setting.CORSConfig.AllowDomain,
			AllowedMethods:   setting.CORSConfig.Methods,
			AllowCredentials: setting.CORSConfig.AllowCredentials,
			AllowedHeaders:   append([]string{"Authorization", "X-Gitea-OTP", "X-Forgejo-OTP"}, setting.CORSConfig.Headers...),
			MaxAge:           int(setting.CORSConfig.MaxAge.Seconds()),
		}))
	}
	m.Use(context.APIContexter())

	m.Use(shared.CheckDeprecatedAuthMethods)

	// Get user from session if logged in.
	m.Use(shared.APIAuth(shared.BuildAuthGroup()))

	m.Use(shared.VerifyAuthWithOptions(&common.VerifyOptions{
		SignInRequired: setting.Service.RequireSignInView,
	}))

	forgejo := NewForgejo()
	m.Get("", Root)
	m.Get("/version", forgejo.GetVersion)
	return m
}
