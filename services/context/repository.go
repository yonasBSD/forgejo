// Copyright 2023 The forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package context

import (
	"net/http"

	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
)

// RepositoryIDAssignmentAPI returns a middleware to handle context-repo assignment for api routes
func RepositoryIDAssignmentAPI() func(ctx *context.APIContext) {
	return func(ctx *context.APIContext) {
		// TODO: enough validation for security?
		repositoryID := ctx.ParamsInt64(":repository-id")

		log.Info("RepositoryIDAssignmentAPI: %v", repositoryID)

		//TODO: check auth here ?
		//if !ctx.Repo.HasAccess() && !ctx.IsUserSiteAdmin() {
		//	ctx.Error(http.StatusForbidden, "reqAnyRepoReader", "user should have any permission to read repository or permissions of site admin")
		//	return
		//}

		var err error
		repository := new(context.Repository)
		// TODO: does repository struct need more infos?
		repository.Repository, err = repo_model.GetRepositoryByID(ctx, repositoryID)

		// TODO: check & convert errors
		if err != nil {
			ctx.Error(http.StatusInternalServerError, "GetRepositoryByID", err)
		}
		ctx.Repo = repository
	}
}
